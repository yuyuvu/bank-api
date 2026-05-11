package middleware

import (
	"bank-api/internal/dto"
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// NewRecoveryMiddleware перехватывает panic и возвращает JSON-ошибку вместо выключения приложения
func NewRecoveryMiddleware(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Errorf("Panic: %v\n%s", rec, debug.Stack())
					dto.WriteJSON(w, http.StatusInternalServerError, dto.APIResponse{
						Success: false,
						Error:   "Внутренняя ошибка сервера",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
