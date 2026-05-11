package middleware

import (
	"bank-api/internal/dto"
	"net/http"
)

// AdminMiddleware пропускает только пользователей с ролью администратора
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := GetRole(r.Context())
		if !ok || role != "admin" {
			dto.WriteJSON(w, http.StatusForbidden, dto.APIResponse{
				Success: false,
				Error:   "Доступ запрещён",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
