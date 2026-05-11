package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"bank-api/internal/security"
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
)

// authService отвечает за логику регистрации, входа и 2FA
type authService struct {
	userRepo  repository.UserRepo
	jwtSecret []byte
	jwtTTL    time.Duration
}

// NewAuthService создаёт сервис для работы с функциями регистрации, логина и двухфакторной аутентификации
func NewAuthService(repos *repository.Repositories, jwtSecret []byte, jwtTTL time.Duration) AuthService {
	return &authService{
		userRepo:  repos.User,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

// Register создаёт нового пользователя
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	if existing, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil && existing != nil {
		return nil, errors.NewAppError(409, "Пользователь с таким email уже существует", nil)
	} else if err != nil && err != errors.ErrNotFound {
		return nil, err
	}

	if existing, err := s.userRepo.GetByUsername(ctx, req.Username); err == nil && existing != nil {
		return nil, errors.NewAppError(409, "Пользователь с таким username уже существует", nil)
	} else if err != nil && err != errors.ErrNotFound {
		return nil, err
	}

	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.ErrInternal
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{Token: token}, nil
}

// Login проверяет учётные данные и при необходимости требует код 2FA для входа и повторного получения токена
func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.NewAppError(401, "Неверный email или пароль", nil)
	}
	if user.IsBlocked {
		return nil, errors.NewAppError(403, "Пользователь заблокирован", nil)
	}
	if err := security.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, errors.NewAppError(401, "Неверный email или пароль", nil)
	}
	if user.TwoFactorEnabled {
		if req.OTPCode == "" {
			return nil, errors.NewAppError(401, "Для входа требуется код 2FA", nil)
		}
		if !totp.Validate(req.OTPCode, user.TwoFactorKey) {
			return nil, errors.NewAppError(401, "Неверный код 2FA", nil)
		}
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{Token: token}, nil
}

// generateToken собирает JWT-токен с идентификатором пользователя и его ролью
func (s *authService) generateToken(userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  strconv.FormatInt(userID, 10),
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(s.jwtTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// Generate2FA генерирует секретный ключ и ссылку для приложения-аутентификатора
func (s *authService) Generate2FA(ctx context.Context, userID int64) (*dto.TwoFactorSetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.TwoFactorEnabled {
		return nil, errors.NewAppError(409, "2FA уже включён", nil)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "BankAPI",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	user.TwoFactorKey = key.Secret()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	qrBase64, err := buildQRCodeBase64(key)
	if err != nil {
		return nil, errors.NewAppError(500, "Не удалось подготовить QR-код для 2FA", err)
	}

	return &dto.TwoFactorSetupResponse{
		Secret:          key.Secret(),
		URL:             key.URL(),
		Issuer:          "BankAPI",
		AccountName:     user.Email,
		QRCodePNGBase64: qrBase64,
	}, nil
}

// Enable2FA окончательно включает двухфакторную аутентификацию после подтверждения кода
func (s *authService) Enable2FA(ctx context.Context, userID int64, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.TwoFactorKey == "" {
		return errors.NewAppError(400, "2FA не настроен", nil)
	}
	if !totp.Validate(code, user.TwoFactorKey) {
		return errors.NewAppError(400, "Неверный код", nil)
	}
	user.TwoFactorEnabled = true
	return s.userRepo.Update(ctx, user)
}

// Disable2FA отключает двухфакторную аутентификацию после подтверждения текущим кодом
func (s *authService) Disable2FA(ctx context.Context, userID int64, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.TwoFactorEnabled {
		return errors.NewAppError(400, "2FA уже выключен", nil)
	}
	if user.TwoFactorKey == "" {
		return errors.NewAppError(400, "Секретный ключ 2FA не найден, отключение невозможно", nil)
	}
	if !totp.Validate(code, user.TwoFactorKey) {
		return errors.NewAppError(400, "Неверный код", nil)
	}

	user.TwoFactorEnabled = false
	user.TwoFactorKey = ""
	return s.userRepo.Update(ctx, user)
}

// Verify2FA проверяет одноразовый код по сохранённому секретному ключу пользователя
func (s *authService) Verify2FA(ctx context.Context, userID int64, code string) bool {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	if !user.TwoFactorEnabled {
		return false
	}
	return totp.Validate(code, user.TwoFactorKey)
}

// BootstrapAdmin назначает первого вошедшего пользователя единственным администратором системы
func (s *authService) BootstrapAdmin(ctx context.Context, userID int64) error {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Role == "admin" {
			if user.ID == userID {
				return errors.NewAppError(409, "Этот пользователь уже является администратором", nil)
			}
			return errors.NewAppError(409, "Администратор уже создан, повторная инициализация запрещена", nil)
		}
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.WrapNotFound(err, "Пользователь для назначения администратором не найден")
	}
	if user.IsBlocked {
		return errors.NewAppError(403, "Заблокированный пользователь не может стать администратором", nil)
	}

	user.Role = "admin"
	return s.userRepo.Update(ctx, user)
}

// buildQRCodeBase64 кодирует QR-код в base64, чтобы его можно было удобно отдать через JSON
func buildQRCodeBase64(key interface {
	Image(int, int) (image.Image, error)
}) (string, error) {
	imageValue, err := key.Image(256, 256)
	if err != nil {
		return "", err
	}

	buffer := bytes.NewBuffer(nil)
	if err := png.Encode(buffer, imageValue); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
