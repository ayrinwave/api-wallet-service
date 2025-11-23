# Build stage
FROM golang:1.24.4-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git make ca-certificates

WORKDIR /app

# ✅ Копируем go.mod и go.sum (context теперь корень проекта)
COPY gw-notification/go.mod gw-notification/go.sum ./gw-notification/

# ✅ Переходим в папку notification
WORKDIR /app/gw-notification

# Скачиваем зависимости
RUN go mod download

# ✅ Копируем исходный код
COPY gw-notification/ .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/notification-service \
    ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Установка ca-certificates
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /app/notification-service .

# Создаем непривилегированного пользователя
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Запуск приложения
CMD ["./notification-service"]