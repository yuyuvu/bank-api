package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"bank-api/internal/validator"
	"net/http"

	"github.com/sirupsen/logrus"
)

// CreditHandler обслуживает выдачу кредитов и платежи по ним
type CreditHandler struct {
	service service.CreditService
	log     *logrus.Logger
}

// NewCreditHandler создаёт обработчик кредитных операций
func NewCreditHandler(s service.CreditService, log *logrus.Logger) *CreditHandler {
	return &CreditHandler{service: s, log: log}
}

// Apply оформляет новый кредит на выбранный счёт пользователя
func (h *CreditHandler) Apply(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var req dto.CreditApplication
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.service.Apply(r.Context(), userID, &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusCreated, dto.APIResponse{Success: true, Data: res})
}

// List возвращает все кредиты текущего пользователя
func (h *CreditHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	credits, err := h.service.List(r.Context(), userID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: credits})
}

// GetSchedule возвращает график платежей по кредиту
func (h *CreditHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID кредита")
		return
	}

	schedule, err := h.service.GetSchedule(r.Context(), userID, id)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: schedule})
}

// PayNext запускает ручную оплату ближайшего непогашенного платежа по кредиту
func (h *CreditHandler) PayNext(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID кредита")
		return
	}

	var req dto.CreditPaymentRequest
	if err := decodeJSONIfPresent(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.service.PayNext(r.Context(), userID, id, &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}
