
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api_wallet ./cmd/

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /api_wallet .

COPY migrations ./migrations

EXPOSE 8080

CMD ["./api_wallet"]