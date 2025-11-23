# Build stage
FROM golang:1.24.4-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git make ca-certificates

WORKDIR /app

# ✅ Копируем go.mod и go.sum (context теперь корень проекта)
COPY gw-exchanger/go.mod gw-exchanger/go.sum ./gw-exchanger/

# ✅ Переходим в папку exchanger
WORKDIR /app/gw-exchanger

# Скачиваем зависимости
RUN go mod download

# ✅ Копируем исходный код
COPY gw-exchanger/ .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/exchanger-service \
    ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Установка ca-certificates для HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /app/exchanger-service .

# Копируем миграции
COPY --from=builder /app/gw-exchanger/migrations ./migrations

# Создаем непривилегированного пользователя
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# gRPC порт
EXPOSE 50051

# Запуск приложения
CMD ["./exchanger-service"]