package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// responseRecorder перехватывает статус и тело ответа, чтобы потом записать их в лог.
type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	responseBody bytes.Buffer
	errorMessage string
}

// WriteHeader запоминает HTTP-статус до передачи его реальному ResponseWriter
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write перехватывает тело ответа, чтобы достать текст ошибки для логов
func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	r.responseBody.Write(data)
	r.captureErrorMessage()

	return r.ResponseWriter.Write(data)
}

// Flush пробрасывает flush, если его поддерживает исходный writer
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack пробрасывает захват соединения для совместимости с net/http
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, http.ErrNotSupported
}

// Push пробрасывает HTTP/2 server push
func (r *responseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

// captureErrorMessage пытается вытащить поле error из JSON-ответа
func (r *responseRecorder) captureErrorMessage() {
	if r.errorMessage != "" {
		return
	}

	var payload struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(r.responseBody.Bytes(), &payload); err == nil && payload.Error != "" {
		r.errorMessage = payload.Error
	}
}

// NewLoggingMiddleware пишет в лог метод, путь, статус и время обработки запроса
func NewLoggingMiddleware(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(recorder, r)

			entry := log.WithFields(logrus.Fields{
				"method":   r.Method,
				"path":     r.URL.Path,
				"status":   recorder.statusCode,
				"duration": time.Since(start),
				"error":    recorder.errorMessage,
			})

			switch {
			case recorder.statusCode >= 500:
				entry.Error("Запрос завершился ошибкой сервера")
			case recorder.statusCode >= 400:
				entry.Warn("Запрос завершился клиентской ошибкой")
			default:
				entry.Info("Запрос обработан")
			}
		})
	}
}
