package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/errors"
	"bank-api/internal/middleware"
	"bank-api/internal/validator"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Вспомогательные функции

// decodeJSON проверяет JSON, чтобы в нём не было лишних полей и второго объекта в теле
func decodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return humanizeDecodeError(err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.NewAppError(http.StatusBadRequest, "В теле запроса должен быть один JSON-объект", nil)
	}

	return nil
}

// decodeJSONIfPresent разрешает пустое тело, если у эндпоинта все поля необязательные
func decodeJSONIfPresent(r *http.Request, dst interface{}) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return nil
	}

	return decodeJSON(r, dst)
}

// getAuthenticatedUserID извлекает ID пользователя из контекста
func getAuthenticatedUserID(r *http.Request) (int64, bool) {
	return middleware.GetUserID(r.Context())
}

// getPathID достаёт числовой идентификатор из параметров
func getPathID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)[key], 10, 64)
}

// writeBadRequest создаёт единообразный ответ на ошибки входных данных
func writeBadRequest(log *logrus.Logger, w http.ResponseWriter, r *http.Request, message string) {
	log.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"error":  message,
	}).Warn("Ошибка входных данных")

	dto.WriteJSON(w, http.StatusBadRequest, dto.APIResponse{Success: false, Error: message})
}

// writeBadRequestError берёт сообщение из AppError и отдельно пишет причину в лог
func writeBadRequestError(log *logrus.Logger, w http.ResponseWriter, r *http.Request, err error) {
	message := err.Error()
	if appErr, ok := err.(*errors.AppError); ok {
		message = appErr.Message
		log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  appErr.Message,
			"cause":  appErr.Err,
		}).Warn("Ошибка разбора запроса")
		dto.WriteJSON(w, appErr.Code, dto.APIResponse{Success: false, Error: message})
		return
	}

	writeBadRequest(log, w, r, message)
}

// writeUnauthorized возвращает ответ для запроса без аутентификации
func writeUnauthorized(log *logrus.Logger, w http.ResponseWriter, r *http.Request, message string) {
	log.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"error":  message,
	}).Warn("Запрос без аутентификации")

	dto.WriteJSON(w, http.StatusUnauthorized, dto.APIResponse{Success: false, Error: message})
}

// handleServiceError переводит ошибки сервисов в HTTP-ответ
func handleServiceError(log *logrus.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		entry := log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": appErr.Code,
			"error":  appErr.Message,
			"cause":  appErr.Err,
		})
		if appErr.Code >= 500 {
			entry.Error("Ошибка обработки запроса")
		} else {
			entry.Warn("Ошибка обработки запроса")
		}

		dto.WriteJSON(w, appErr.Code, dto.APIResponse{Success: false, Error: appErr.Message})
		return
	}

	log.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"error":  err.Error(),
	}).Error("Необработанная ошибка")

	dto.WriteJSON(w, http.StatusInternalServerError, dto.APIResponse{Success: false, Error: err.Error()})
}

// writeValidationError переводит техническую ошибку валидатора в понятный текст
func writeValidationError(log *logrus.Logger, w http.ResponseWriter, r *http.Request, err error) {
	writeBadRequest(log, w, r, validator.HumanizeError(err))
}

// humanizeDecodeError переводит технические ошибки JSON-декодера в понятный текст
func humanizeDecodeError(err error) error {
	var syntaxErr *json.SyntaxError
	if stdErrors.As(err, &syntaxErr) {
		return errors.NewAppError(http.StatusBadRequest, "JSON в теле запроса содержит синтаксическую ошибку", err)
	}

	var typeErr *json.UnmarshalTypeError
	if stdErrors.As(err, &typeErr) {
		return errors.NewAppError(http.StatusBadRequest, "Одно из полей запроса имеет неверный тип данных", err)
	}

	if err == io.EOF {
		return errors.NewAppError(http.StatusBadRequest, "Тело запроса не должно быть пустым", err)
	}

	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf("Поле %s не поддерживается этим эндпоинтом", field), err)
	}

	return errors.NewAppError(http.StatusBadRequest, "Не удалось разобрать JSON-запрос", err)
}
