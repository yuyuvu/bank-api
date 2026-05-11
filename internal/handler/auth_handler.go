package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"bank-api/internal/validator"
	"net/http"

	"github.com/sirupsen/logrus"
)

// AuthHandler обслуживает регистрацию, логин и настройку 2FA
type AuthHandler struct {
	authService service.AuthService
	log         *logrus.Logger
}

// NewAuthHandler создаёт обработчик для регистрации, логина и настройки 2FA
func NewAuthHandler(service service.AuthService, log *logrus.Logger) *AuthHandler {
	return &AuthHandler{authService: service, log: log}
}

// Register регистрирует нового пользователя и выдаёт JWT-токен
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest

	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusCreated, dto.APIResponse{Success: true, Data: res})
}

// Login проверяет учётные данные и возвращает JWT-токен
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest

	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// Generate2FA готовит секретный ключ для приложения-аутентификатора
func (h *AuthHandler) Generate2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	res, err := h.authService.Generate2FA(r.Context(), userID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// Enable2FA подтверждает включение 2FA кодом из приложения
func (h *AuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var req dto.Enable2FARequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	if err := h.authService.Enable2FA(r.Context(), userID, req.Code); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "2FA успешно включён"})
}

// Disable2FA подтверждает отключение 2FA кодом из приложения
func (h *AuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var req dto.Disable2FARequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	if err := h.authService.Disable2FA(r.Context(), userID, req.Code); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "2FA успешно отключён"})
}

// BootstrapAdmin даёт текущему пользователю роль первого и единственного администратора, если администратора ещё нет
func (h *AuthHandler) BootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	if err := h.authService.BootstrapAdmin(r.Context(), userID); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "Пользователь назначен администратором"})
}
