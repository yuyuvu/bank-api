# Стадия сборки
FROM golang:1.26.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# Финальный образ
FROM alpine:3.23.4
RUN apk --no-cache add ca-certificates tzdata
# Указываем рабочую директорию
WORKDIR /app
# Копируем файлы внутрь этой директории
COPY --from=builder /api .
COPY configs/ ./configs/
COPY migrations/ ./migrations/
EXPOSE 8080
#Запускаем бинарный файл
ENTRYPOINT ["./api"]