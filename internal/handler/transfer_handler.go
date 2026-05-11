package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"bank-api/internal/validator"
	"net/http"

	"github.com/sirupsen/logrus"
)

// TransferHandler обслуживает переводы между счетами
type TransferHandler struct {
	service service.TransferService
	log     *logrus.Logger
}

// NewTransferHandler создаёт обработчик переводов между счетами
func NewTransferHandler(s service.TransferService, log *logrus.Logger) *TransferHandler {
	return &TransferHandler{service: s, log: log}
}

// Transfer переводит деньги между счетами
func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var req dto.TransferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.service.Transfer(r.Context(), userID, &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}
