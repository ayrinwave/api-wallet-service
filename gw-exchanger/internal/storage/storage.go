package storage

import (
	"context"
	"gw-exchanger/internal/models"
)

// Storage интерфейс для работы с курсами валют
type Storage interface {
	// GetAllRates возвращает все курсы валют
	GetAllRates(ctx context.Context) ([]models.ExchangeRate, error)

	// GetRateByCurrency возвращает курс для конкретной валюты
	GetRateByCurrency(ctx context.Context, currency string) (*models.ExchangeRate, error)
}
