package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-exchanger/internal/models"
	"gw-exchanger/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

// GetAllRates возвращает все курсы валют
func (s *PostgresStorage) GetAllRates(ctx context.Context) ([]models.ExchangeRate, error) {

	rows, err := s.pool.Query(ctx, storage.GetAllRatesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query rates: %w", err)
	}
	defer rows.Close()

	var rates []models.ExchangeRate
	for rows.Next() {
		var rate models.ExchangeRate
		if err := rows.Scan(&rate.ID, &rate.Currency, &rate.Rate, &rate.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rate: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return rates, nil
}

// GetRateByCurrency возвращает курс для конкретной валюты
func (s *PostgresStorage) GetRateByCurrency(ctx context.Context, currency string) (*models.ExchangeRate, error) {

	var rate models.ExchangeRate
	err := s.pool.QueryRow(ctx, storage.GetRateByCurrencyQuery, currency).Scan(
		&rate.ID,
		&rate.Currency,
		&rate.Rate,
		&rate.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("currency %s not found", currency)
		}
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}

	return &rate, nil
}
