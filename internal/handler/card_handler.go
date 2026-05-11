package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"bank-api/internal/validator"
	"net/http"

	"github.com/sirupsen/logrus"
)

// CardHandler обслуживает выпуск, просмотр и оплату банковскими картами
type CardHandler struct {
	service service.CardService
	log     *logrus.Logger
}

// NewCardHandler создаёт обработчик операций с банковскими картами
func NewCardHandler(s service.CardService, log *logrus.Logger) *CardHandler {
	return &CardHandler{service: s, log: log}
}

// IssueCard выпускает банковскую карту для выбранного счёта
func (h *CardHandler) IssueCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	var body dto.IssueCardRequest
	if err := decodeJSON(r, &body); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(body); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	card, err := h.service.IssueCard(r.Context(), userID, body.AccountID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusCreated, dto.APIResponse{Success: true, Data: card})
}

// ListCards возвращает список банковских карт пользователя в маскированном виде
func (h *CardHandler) ListCards(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	cards, err := h.service.ListCards(r.Context(), userID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: cards})
}

// GetCard возвращает владельцу расшифрованные данные банковской карты
func (h *CardHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID карты")
		return
	}

	card, err := h.service.GetCard(r.Context(), userID, id)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: card})
}

// Pay списывает деньги со счёта, к которому привязана карта
func (h *CardHandler) Pay(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	cardID, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID карты")
		return
	}

	var req dto.CardPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		writeValidationError(h.log, w, r, err)
		return
	}

	res, err := h.service.Pay(r.Context(), userID, cardID, &req)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}
