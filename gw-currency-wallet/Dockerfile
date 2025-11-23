# Build stage
FROM golang:1.24.4-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git make ca-certificates

WORKDIR /app

# ✅ Копируем go.mod и go.sum ОБОИХ модулей (т.к. context теперь корень)
COPY gw-currency-wallet/go.mod gw-currency-wallet/go.sum ./gw-currency-wallet/
COPY gw-exchanger/go.mod gw-exchanger/go.sum ./gw-exchanger/

# ✅ Переходим в папку wallet для работы
WORKDIR /app/gw-currency-wallet

# Скачиваем зависимости (replace директива теперь работает)
RUN go mod download

# ✅ Копируем исходный код wallet
COPY gw-currency-wallet/ .

# ✅ Копируем нужные файлы из gw-exchanger (для replace директивы)
COPY gw-exchanger/ ../gw-exchanger/

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/wallet-service \
    ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Установка ca-certificates для HTTPS и tzdata для таймзон
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /app/wallet-service .

# Копируем миграции
COPY --from=builder /app/gw-currency-wallet/migrations ./migrations

# Создаем непривилегированного пользователя
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Порт приложения
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Запуск приложения
CMD ["./wallet-service"]