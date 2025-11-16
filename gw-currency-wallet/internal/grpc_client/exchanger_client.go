package grpc_client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "gw-exchanger/proto-exchange"
)

// ExchangeRatesResponse структура для ответа с курсами валют
type ExchangeRatesResponse struct {
	Rates map[string]float64 // USD: 1.0, RUB: 95.5, EUR: 0.92
}

// ExchangeRateResponse структура для ответа с курсом для конкретной пары
type ExchangeRateResponse struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
}

// ExchangerClient интерфейс для работы с gRPC сервисом exchanger
type ExchangerClient interface {
	GetExchangeRates(ctx context.Context) (*ExchangeRatesResponse, error)
	GetExchangeRateForCurrency(ctx context.Context, from, to string) (*ExchangeRateResponse, error)
	Close() error
}

type grpcExchangerClient struct {
	conn    *grpc.ClientConn
	client  pb.ExchangeServiceClient // Раскомментируйте после генерации proto
	timeout time.Duration
	log     *slog.Logger
}

// NewExchangerClient создает новый gRPC клиент для exchanger
func NewExchangerClient(addr string, timeout time.Duration, log *slog.Logger) (ExchangerClient, error) {
	const op = "grpc_client.NewExchangerClient"

	log.Info("подключение к gRPC exchanger сервису", slog.String("addr", addr))

	// Устанавливаем соединение с gRPC сервером
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect: %w", op, err)
	}

	client := pb.NewExchangeServiceClient(conn) // Раскомментируйте после генерации proto

	log.Info("успешное подключение к exchanger сервису")

	return &grpcExchangerClient{
		conn:    conn,
		client:  client,
		timeout: timeout,
		log:     log,
	}, nil
}

// GetExchangeRates получает все курсы валют
func (c *grpcExchangerClient) GetExchangeRates(ctx context.Context) (*ExchangeRatesResponse, error) {
	const op = "grpc_client.GetExchangeRates"

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.log.Debug("запрос курсов валют через gRPC")

	// TODO: Раскомментируйте после генерации proto файлов

	resp, err := c.client.GetExchangeRates(ctx, &pb.Empty{})
	if err != nil {
		c.log.Error("ошибка получения курсов", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	c.log.Debug("получены курсы валют", slog.Any("rates", resp.Rates))

	return &ExchangeRatesResponse{
		Rates: resp.Rates,
	}, nil

	// ВРЕМЕННАЯ ЗАГЛУШКА - удалите после интеграции с реальным gRPC сервисом
	c.log.Warn("используется заглушка для курсов валют")
	return &ExchangeRatesResponse{
		Rates: map[string]float64{
			"USD": 1.0,
			"RUB": 95.5,
			"EUR": 0.92,
		},
	}, nil
}

// GetExchangeRateForCurrency получает курс для конкретной пары валют
func (c *grpcExchangerClient) GetExchangeRateForCurrency(ctx context.Context, from, to string) (*ExchangeRateResponse, error) {
	const op = "grpc_client.GetExchangeRateForCurrency"

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.log.Debug("запрос курса валюты",
		slog.String("from", from),
		slog.String("to", to))

	// TODO: Раскомментируйте после генерации proto файлов

	resp, err := c.client.GetExchangeRateForCurrency(ctx, &pb.CurrencyRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		c.log.Error("ошибка получения курса",
			slog.String("op", op),
			slog.String("from", from),
			slog.String("to", to),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	c.log.Debug("получен курс валюты",
		slog.String("from", resp.FromCurrency),
		slog.String("to", resp.ToCurrency),
		slog.Float64("rate", float64(resp.Rate)))

	return &ExchangeRateResponse{
		FromCurrency: resp.FromCurrency,
		ToCurrency:   resp.ToCurrency,
		Rate:         float64(resp.Rate),
	}, nil

	// ВРЕМЕННАЯ ЗАГЛУШКА - удалите после интеграции с реальным gRPC сервисом
	c.log.Warn("используется заглушка для курса валюты")

	// Простая логика расчета курса из базовых значений
	rates := map[string]float64{
		"USD": 1.0,
		"RUB": 95.5,
		"EUR": 0.92,
	}

	fromRate, okFrom := rates[from]
	toRate, okTo := rates[to]

	if !okFrom || !okTo {
		return nil, fmt.Errorf("%s: unsupported currency pair %s/%s", op, from, to)
	}

	// Курс = toRate / fromRate
	// Например: USD -> RUB = 95.5 / 1.0 = 95.5
	// EUR -> USD = 1.0 / 0.92 = 1.087
	rate := toRate / fromRate

	return &ExchangeRateResponse{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
	}, nil
}

// Close закрывает соединение с gRPC сервером
func (c *grpcExchangerClient) Close() error {
	c.log.Info("закрытие соединения с exchanger сервисом")
	return c.conn.Close()
}
