package models

import (
	"time"

	"github.com/google/uuid"
)

// событие о крупном денежном переводе (>= 30000)
type LargeTransferEvent struct {
	TransactionID string    `json:"transaction_id"`   // Уникальный ID транзакции
	UserID        uuid.UUID `json:"user_id"`          // ID пользователя
	FromCurrency  string    `json:"from_currency"`    // Исходная валюта
	ToCurrency    string    `json:"to_currency"`      // Целевая валюта
	Amount        float64   `json:"amount"`           // Сумма в исходной валюте
	ExchangedAmt  float64   `json:"exchanged_amount"` // Сумма после обмена
	Rate          float64   `json:"rate"`             // Курс обмена
	Timestamp     time.Time `json:"timestamp"`        // Время операции
}
