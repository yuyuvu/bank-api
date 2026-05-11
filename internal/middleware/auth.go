package middleware

import (
	"bank-api/internal/dto"
	"bank-api/internal/repository"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// contextKey нужен, чтобы не конфликтовать с чужими ключами в request context
type contextKey string

const (
	// UserIDKey хранит ID аутентифицированного пользователя в контексте запроса
	UserIDKey contextKey = "userID"
	// RoleKey хранит роль пользователя из JWT
	RoleKey contextKey = "role"
)

// NewAuthMiddleware проверяет JWT, сверяет пользователя с базой и помещает его данные в контекст
func NewAuthMiddleware(jwtSecret string, userRepo repository.UserRepo) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Требуется заголовок Authorization",
				})
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Заголовок Authorization должен содержать префикс Bearer",
				})
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if tokenString == "" {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "JWT-токен не передан",
				})
				return
			}

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("неверный метод подписи")
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Токен недействителен",
				})
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Не удалось прочитать claims токена",
				})
				return
			}

			subject, err := extractSubject(claims["sub"])
			if err != nil {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Некорректный идентификатор пользователя в токене",
				})
				return
			}

			user, err := userRepo.GetByID(r.Context(), subject)
			if err != nil {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{
					Success: false,
					Error:   "Пользователь из токена не найден",
				})
				return
			}
			if user.IsBlocked {
				dto.WriteJSON(w, http.StatusForbidden, dto.APIResponse{
					Success: false,
					Error:   "Пользователь заблокирован",
				})
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, subject)
			ctx = context.WithValue(ctx, RoleKey, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractSubject приводит claims subject (sub) к int64
func extractSubject(rawSubject interface{}) (int64, error) {
	switch value := rawSubject.(type) {
	case string:
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, err
		}
		return userID, nil
	case float64:
		return int64(value), nil
	default:
		return 0, fmt.Errorf("неподдерживаемый тип sub")
	}
}

// GetUserID достаёт идентификатор пользователя из контекста
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

// GetRole достаёт роль пользователя из контекста
func GetRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(RoleKey).(string)
	return role, ok
}
