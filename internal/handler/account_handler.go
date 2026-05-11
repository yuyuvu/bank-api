package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"bank-api/internal/validator"
	"net/http"

	"github.com/sirupsen/logrus"
)

// AccountHandler обслуживает операции со счетами пользователя
type AccountHandler struct {
	service service.AccountService
	log     *logrus.Logger
}

// NewAccountHandler создаёт обработчик операций со счетами
func NewAccountHandler(s service.AccountService, log *logrus.Logger) *AccountHandler {
	return &AccountHandler{service: s, log: log}
}

// CreateAccount создаёт новый рублёвый счёт для текущего пользователя
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var req dto.CreateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.service.Create(r.Context(), userID, req.Currency)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusCreated, dto.APIResponse{Success: true, Data: res})
}

// ListAccounts возвращает все счета пользователя
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	accounts, err := h.service.List(r.Context(), userID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: accounts})
}

// GetAccount возвращает один определённый счёт пользователя
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	acc, err := h.service.GetByID(r.Context(), userID, id)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: acc})
}

// Deposit пополняет счёт пользователя
func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	var body dto.AmountRequest
	if err := decodeJSON(r, &body); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(body); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	if err := h.service.Deposit(r.Context(), userID, id, body.Amount); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "Пополнение успешно"})
}

// Withdraw списывает деньги со счёта пользователя
func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	var body dto.AmountRequest
	if err := decodeJSON(r, &body); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(body); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	if err := h.service.Withdraw(r.Context(), userID, id, body.Amount); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "Списание успешно"})
}
