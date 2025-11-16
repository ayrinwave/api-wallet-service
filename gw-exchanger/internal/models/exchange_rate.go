package models

import "time"

// ExchangeRate представляет курс обмена валюты
type ExchangeRate struct {
	ID        int       `db:"id"`
	Currency  string    `db:"currency"` // USD, RUB, EUR
	Rate      float64   `db:"rate"`     // Курс относительно базовой валюты (USD = 1.0)
	UpdatedAt time.Time `db:"updated_at"`
}
