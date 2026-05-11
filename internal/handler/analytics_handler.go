package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// AnalyticsHandler обслуживает все запросы пользователя на получение аналитики
type AnalyticsHandler struct {
	service service.AnalyticsService
	log     *logrus.Logger
}

// NewAnalyticsHandler создаёт обработчик запросов пользователя на получение аналитики
func NewAnalyticsHandler(s service.AnalyticsService, log *logrus.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{service: s, log: log}
}

// IncomeExpense возвращает статистику доходов и расходов за месяц
func (h *AnalyticsHandler) IncomeExpense(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	yearMonth := r.URL.Query().Get("year_month")
	if yearMonth == "" {
		writeBadRequest(h.log, w, r, "Параметр year_month обязателен")
		return
	}

	data, err := h.service.IncomeExpense(r.Context(), userID, yearMonth)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}

// IncomeExpenseByAccount возвращает статистику доходов и расходов по одному счёту
func (h *AnalyticsHandler) IncomeExpenseByAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	accountID, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	yearMonth := r.URL.Query().Get("year_month")
	if yearMonth == "" {
		writeBadRequest(h.log, w, r, "Параметр year_month обязателен")
		return
	}

	data, err := h.service.IncomeExpenseByAccount(r.Context(), userID, accountID, yearMonth)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}

// CreditLoad возвращает текущую долговую нагрузку пользователя
func (h *AnalyticsHandler) CreditLoad(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}
	if r.URL.Query().Get("year_month") != "" {
		writeBadRequest(h.log, w, r, "Этот эндпоинт показывает текущую кредитную нагрузку. Параметр year_month для него не поддерживается")
		return
	}

	data, err := h.service.CreditLoad(r.Context(), userID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}

// CreditLoadByAccount возвращает текущую кредитную нагрузку по одному счёту
func (h *AnalyticsHandler) CreditLoadByAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	accountID, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}
	if r.URL.Query().Get("year_month") != "" {
		writeBadRequest(h.log, w, r, "Этот эндпоинт показывает текущую кредитную нагрузку по счёту. Параметр year_month для него не поддерживается")
		return
	}

	data, err := h.service.CreditLoadByAccount(r.Context(), userID, accountID)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}

// PredictBalance строит прогноз по выбранному счёту на определённое количество дней вперёд
func (h *AnalyticsHandler) PredictBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	accountID, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		d, convErr := strconv.Atoi(daysStr)
		if convErr != nil {
			writeBadRequest(h.log, w, r, "Параметр days должен быть числом")
			return
		}
		days = d
	}

	if days <= 0 {
		writeBadRequest(h.log, w, r, "Параметр days должен быть положительным")
		return
	}

	predictions, err := h.service.PredictBalance(r.Context(), userID, accountID, days)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: predictions})
}

// PredictAllBalances строит прогноз по всем счетам пользователя
func (h *AnalyticsHandler) PredictAllBalances(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	days := 30
	daysRaw := r.URL.Query().Get("days")
	if daysRaw != "" {
		value, err := strconv.Atoi(daysRaw)
		if err != nil {
			writeBadRequest(h.log, w, r, "Параметр days должен быть числом")
			return
		}
		days = value
	}

	predictions, err := h.service.PredictAllBalances(r.Context(), userID, days)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: predictions})
}

// Summary возвращает сводную аналитику по всем счетам пользователя
func (h *AnalyticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	yearMonth := r.URL.Query().Get("year_month")
	if yearMonth == "" {
		yearMonth = time.Now().Format("2006-01")
	}

	data, err := h.service.Summary(r.Context(), userID, yearMonth)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}

// AccountSummary возвращает сводную аналитику по одному счёту
func (h *AnalyticsHandler) AccountSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := getAuthenticatedUserID(r)
	if !ok {
		writeUnauthorized(h.log, w, r, "Не удалось определить пользователя из токена")
		return
	}

	accountID, err := getPathID(r, "id")
	if err != nil {
		writeBadRequest(h.log, w, r, "Неверный ID счёта")
		return
	}

	yearMonth := r.URL.Query().Get("year_month")
	if yearMonth == "" {
		yearMonth = time.Now().Format("2006-01")
	}

	data, err := h.service.AccountSummary(r.Context(), userID, accountID, yearMonth)
	if err != nil {
		handleServiceError(h.log, w, r, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: data})
}
