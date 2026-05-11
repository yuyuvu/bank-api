package service

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/model"
	"bank-api/internal/repository"
	"bank-api/internal/security"
	"context"
	"crypto/rand"
	"database/sql"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// cardService хранит логику для работы с операциями выпуска и использования банковских карт
type cardService struct {
	cardRepo            repository.CardRepo
	accountRepo         repository.AccountRepo
	transactionRepo     repository.TransactionRepo
	userRepo            repository.UserRepo
	pgp                 *security.PGPService
	hmacSecret          []byte
	db                  *sql.DB
	notificationService *NotificationService
}

// NewCardService создаёт сервис для работы с операциями выпуска и использования банковских карт
func NewCardService(
	repos *repository.Repositories,
	pgp *security.PGPService,
	hmacSecret []byte,
	db *sql.DB,
	notification *NotificationService,
) CardService {
	return &cardService{
		cardRepo:            repos.Card,
		accountRepo:         repos.Account,
		transactionRepo:     repos.Transaction,
		userRepo:            repos.User,
		pgp:                 pgp,
		hmacSecret:          hmacSecret,
		db:                  db,
		notificationService: notification,
	}
}

// IssueCard выпускает новую банковскую карту для счёта владельца
func (s *cardService) IssueCard(ctx context.Context, userID, accountID int64) (*dto.CardResponse, error) {
	// Проверка принадлежности счета
	acc, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт для выпуска карты не найден")
	}
	if acc.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя выпустить карту для чужого счёта", nil)
	}

	// Генерация номера карты (16 цифр по алгоритму Луна)
	number := generateValidCardNumber()

	// Срок действия (5 лет от текущей даты)
	expiry := time.Now().AddDate(5, 0, 0).Format("01/2006") // MM/YYYY

	// CVV (3 цифры)
	cvv, err := generateRandomDigits(3)
	if err != nil {
		return nil, errors.NewAppError(500, "Не удалось сгенерировать CVV", err)
	}

	// Шифрование
	encNumber, err := s.pgp.Encrypt(number)
	if err != nil {
		return nil, errors.ErrInternal
	}
	encExpiry, err := s.pgp.Encrypt(expiry)
	if err != nil {
		return nil, errors.ErrInternal
	}
	encCVV, err := s.pgp.Encrypt(cvv)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// HMAC для номера
	hmacNumber := security.ComputeHMAC(number, s.hmacSecret)

	// Хеширование CVV
	bcryptCVV, err := security.HashCVV(cvv)
	if err != nil {
		return nil, errors.ErrInternal
	}

	card := &model.Card{
		AccountID:       accountID,
		UserID:          userID,
		EncryptedNumber: encNumber,
		HmacNumber:      hmacNumber,
		EncryptedExpiry: encExpiry,
		EncryptedCVV:    encCVV,
		BcryptCVV:       bcryptCVV,
		Status:          "active",
	}

	if err := s.cardRepo.Create(ctx, card); err != nil {
		return nil, err
	}

	masked := maskCardNumber(number)
	return &dto.CardResponse{
		ID:           card.ID,
		AccountID:    card.AccountID,
		MaskedNumber: masked,
		Status:       card.Status,
	}, nil
}

// ListCards возвращает список карт пользователя в маскированном виде
func (s *cardService) ListCards(ctx context.Context, userID int64) ([]*dto.CardResponse, error) {
	cards, err := s.cardRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var res []*dto.CardResponse
	for _, c := range cards {
		number, err := s.decryptAndVerifyCardNumber(c)
		if err != nil {
			return nil, err
		}

		res = append(res, &dto.CardResponse{
			ID:           c.ID,
			AccountID:    c.AccountID,
			MaskedNumber: maskCardNumber(number),
			Status:       c.Status,
		})
	}
	return res, nil
}

// GetCard возвращает владельцу расшифрованные реквизиты карты
func (s *cardService) GetCard(ctx context.Context, userID, cardID int64) (*dto.CardDetailResponse, error) {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Карта не найдена")
	}
	if card.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя просматривать чужую карту", nil)
	}

	number, err := s.decryptAndVerifyCardNumber(card)
	if err != nil {
		return nil, err
	}

	expiry, err := s.pgp.Decrypt(card.EncryptedExpiry)
	if err != nil {
		return nil, errors.ErrInternal
	}
	cvv, err := s.decryptCVV(card)
	if err != nil {
		return nil, err
	}

	return &dto.CardDetailResponse{
		ID:     card.ID,
		Number: number,
		Expiry: expiry,
		CVV:    cvv,
		Status: card.Status,
	}, nil
}

// Pay выполняет оплату картой после проверки владельца, CVV, HMAC и опционально 2FA
func (s *cardService) Pay(ctx context.Context, userID, cardID int64, req *dto.CardPaymentRequest) (*dto.CardPaymentResponse, error) {
	if err := validateTwoFactorIfEnabled(ctx, s.userRepo, userID, req.OTPCode); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, errors.NewAppError(400, "Сумма оплаты должна быть положительной", nil)
	}

	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Карта для оплаты не найдена")
	}
	if card.UserID != userID {
		return nil, errors.NewAppError(403, "Нельзя оплачивать чужой картой", nil)
	}
	if card.Status != "active" {
		return nil, errors.NewAppError(400, "Карта недоступна для оплаты", nil)
	}
	if _, err := s.decryptAndVerifyCardNumber(card); err != nil {
		return nil, err
	}
	if err := security.VerifyPassword(card.BcryptCVV, req.CVV); err != nil {
		return nil, errors.NewAppError(401, "Неверный CVV", nil)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	account, err := s.accountRepo.GetForUpdate(ctx, tx, card.AccountID)
	if err != nil {
		return nil, errors.WrapNotFound(err, "Счёт, привязанный к карте, не найден")
	}
	if account.UserID != userID {
		return nil, errors.NewAppError(403, "Карта привязана к чужому счёту", nil)
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, account.ID, -req.Amount); err != nil {
		if errors.HasCode(err, 400) {
			return nil, errors.NewAppError(400, "На счёте недостаточно средств для оплаты", err)
		}
		return nil, err
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "Оплата банковской картой"
	}

	txn := &model.Transaction{
		FromAccountID: &account.ID,
		Amount:        req.Amount,
		Type:          "card_payment",
		Description:   description,
	}
	if err := s.transactionRepo.CreateTx(ctx, tx, txn); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if user, err := s.userRepo.GetByID(ctx, userID); err == nil && s.notificationService != nil {
		_ = s.notificationService.SendCardPaymentEmail(user.Email, txn.CreatedAt, req.Amount, description, account.ID, txn.ID)
	}

	return &dto.CardPaymentResponse{
		TransactionID: txn.ID,
		AccountID:     account.ID,
		Amount:        req.Amount,
	}, nil
}

// decryptAndVerifyCardNumber расшифровывает номер карты и дополнительно проверяет его HMAC.
func (s *cardService) decryptAndVerifyCardNumber(card *model.Card) (string, error) {
	number, err := s.pgp.Decrypt(card.EncryptedNumber)
	if err != nil {
		return "", errors.ErrInternal
	}
	if !security.VerifyHMAC(number, card.HmacNumber, s.hmacSecret) {
		return "", errors.NewAppError(500, "Нарушена целостность данных карты", nil)
	}
	return number, nil
}

// decryptCVV расшифровывает CVV для владельца карты
func (s *cardService) decryptCVV(card *model.Card) (string, error) {
	if strings.TrimSpace(card.EncryptedCVV) == "" {
		return "", errors.NewAppError(409, "Для этой карты CVV недоступен. Выпустите новую карту, чтобы увидеть CVV в API", nil)
	}

	cvv, err := s.pgp.Decrypt(card.EncryptedCVV)
	if err != nil {
		return "", errors.ErrInternal
	}

	return cvv, nil
}

// maskCardNumber оставляет только последние четыре цифры номера
func maskCardNumber(number string) string {
	if len(number) < 4 {
		return "****"
	}

	return "**** **** **** " + number[len(number)-4:]
}

// Алгоритм Луна
func generateValidCardNumber() string {
	for {
		digits := make([]int, 16)
		// Первая цифра 4 (Visa)
		digits[0] = 4
		for i := 1; i < 15; i++ {
			n, _ := rand.Int(rand.Reader, big.NewInt(10))
			digits[i] = int(n.Int64())
		}
		// Вычисление контрольной цифры
		sum := 0
		for i := 0; i < 15; i++ {
			d := digits[i]
			if i%2 == 0 {
				d *= 2
				if d > 9 {
					d -= 9
				}
			}
			sum += d
		}
		checkDigit := (10 - (sum % 10)) % 10
		digits[15] = checkDigit

		// Преобразование в строку
		sb := strings.Builder{}
		for _, d := range digits {
			sb.WriteString(strconv.Itoa(d))
		}
		number := sb.String()
		if isValidLuhn(number) {
			return number
		}
	}
}

// isValidLuhn перепроверяет номер карты по алгоритму Луна
func isValidLuhn(number string) bool {
	sum := 0
	doubleDigit := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}

		if doubleDigit {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		doubleDigit = !doubleDigit
	}

	return sum%10 == 0
}

// generateRandomDigits генерирует строку из нужного количества цифр
func generateRandomDigits(length int) (string, error) {
	sb := strings.Builder{}
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		sb.WriteString(strconv.Itoa(int(n.Int64())))
	}
	return sb.String(), nil
}
