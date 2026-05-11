package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"net/http"

	"github.com/sirupsen/logrus"
)

// AdminHandler обслуживает эндпоинты для администратора
type AdminHandler struct {
	adminService service.AdminService
	log          *logrus.Logger
}

// NewAdminHandler создаёт обработчик функций для администратора
func NewAdminHandler(s service.AdminService, log *logrus.Logger) *AdminHandler {
	return &AdminHandler{adminService: s, log: log}
}

// ListUsers возвращает список пользователей для администратора
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.adminService.ListUsers(r.Context())
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: users})
}

// BlockUser меняет статус блокировки выбранного пользователя
func (h *AdminHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	id, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID пользователя")
		return
	}

	var body dto.BlockRequest
	if err := decodeJSON(r, &body); err != nil {
		writeBadRequestError(h.log, w, r, err)
		return
	}

	if body.Block == nil {
		writeBadRequest(h.log, w, r, "Поле 'block' со значением типа bool обязательно")
		return
	}

	if err := h.adminService.BlockUser(r.Context(), id, *body.Block); err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: "Статус блокировки пользователя изменён"})
}
