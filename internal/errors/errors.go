package errors

import (
	"errors"
	"fmt"
)

// AppError хранит HTTP-код, сообщение и исходную причину ошибки
type AppError struct {
	Code    int
	Message string
	Err     error
}

// Error собирает понятное описание ошибки
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewAppError создаёт ошибку AppError с HTTP-кодом, сообщением и причиной
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// HasCode проверяет, что ошибка относится к определённому HTTP-коду
func HasCode(err error, code int) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}

	return false
}

// WrapNotFound заменяет обычную ошибку 404 на ошибку 404 с более подробным описанием причины
func WrapNotFound(err error, message string) error {
	if HasCode(err, 404) {
		return NewAppError(404, message, err)
	}

	return err
}

var (
	// Набор базовых ошибок для API
	ErrNotFound          = &AppError{Code: 404, Message: "Ресурс не найден"}
	ErrUnauthorized      = &AppError{Code: 401, Message: "Требуется аутентификация"}
	ErrForbidden         = &AppError{Code: 403, Message: "Доступ запрещён"}
	ErrValidation        = &AppError{Code: 400, Message: "Ошибка валидации"}
	ErrInternal          = &AppError{Code: 500, Message: "Внутренняя ошибка сервера"}
	ErrInsufficientFunds = &AppError{Code: 400, Message: "Недостаточно средств"}
	ErrConflict          = &AppError{Code: 409, Message: "Конфликт данных"}
)
