package grpc_server

import (
	"context"
	"fmt"
	"gw-exchanger/internal/storage"
	"log/slog"

	pb "gw-exchanger/proto-exchange"
)

// ExchangeServer реализует pb.ExchangeServiceServer
type ExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
	storage storage.Storage
	log     *slog.Logger
}

func NewExchangeServer(storage storage.Storage, log *slog.Logger) *ExchangeServer {
	return &ExchangeServer{
		storage: storage,
		log:     log,
	}
}

// GetExchangeRates возвращает все курсы валют
func (s *ExchangeServer) GetExchangeRates(ctx context.Context, req *pb.Empty) (*pb.ExchangeRatesResponse, error) {
	const op = "grpc_server.GetExchangeRates"

	s.log.Info("получен запрос на все курсы валют", slog.String("op", op))

	rates, err := s.storage.GetAllRates(ctx)
	if err != nil {
		s.log.Error("ошибка получения курсов", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Преобразуем в proto response
	ratesMap := make(map[string]float32)
	for _, rate := range rates {
		ratesMap[rate.Currency] = float32(rate.Rate)
	}

	s.log.Info("отправлены курсы валют",
		slog.String("op", op),
		slog.Int("count", len(ratesMap)))

	return &pb.ExchangeRatesResponse{
		Rates: ratesMap,
	}, nil
}

// GetExchangeRateForCurrency возвращает курс для конкретной пары валют
func (s *ExchangeServer) GetExchangeRateForCurrency(ctx context.Context, req *pb.CurrencyRequest) (*pb.ExchangeRateResponse, error) {
	const op = "grpc_server.GetExchangeRateForCurrency"

	s.log.Info("получен запрос на курс валюты",
		slog.String("op", op),
		slog.String("from", req.FromCurrency),
		slog.String("to", req.ToCurrency))

	validCurrencies := map[string]bool{"USD": true, "RUB": true, "EUR": true}
	if !validCurrencies[req.FromCurrency] || !validCurrencies[req.ToCurrency] {
		return nil, fmt.Errorf("invalid currency")
	}

	// Получаем курсы обеих валют
	fromRate, err := s.storage.GetRateByCurrency(ctx, req.FromCurrency)
	if err != nil {
		s.log.Error("ошибка получения курса исходной валюты",
			slog.String("op", op),
			slog.String("currency", req.FromCurrency),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	toRate, err := s.storage.GetRateByCurrency(ctx, req.ToCurrency)
	if err != nil {
		s.log.Error("ошибка получения курса целевой валюты",
			slog.String("op", op),
			slog.String("currency", req.ToCurrency),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Вычисляем курс обмена
	// Например: USD -> RUB
	// USD rate = 1.0, RUB rate = 95.5
	// Курс = 95.5 / 1.0 = 95.5
	rate := toRate.Rate / fromRate.Rate

	s.log.Info("отправлен курс обмена",
		slog.String("op", op),
		slog.String("from", req.FromCurrency),
		slog.String("to", req.ToCurrency),
		slog.Float64("rate", rate))

	return &pb.ExchangeRateResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         float32(rate),
	}, nil
}
