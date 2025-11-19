package models

import "time"

// LargeTransferNotification представляет уведомление о крупном денежном переводе
type LargeTransferNotification struct {
	ID            string    `bson:"_id,omitempty" json:"id"`
	TransactionID string    `bson:"transaction_id" json:"transaction_id"`
	UserID        string    `bson:"user_id" json:"user_id"`
	FromCurrency  string    `bson:"from_currency" json:"from_currency"`
	ToCurrency    string    `bson:"to_currency" json:"to_currency"`
	Amount        float64   `bson:"amount" json:"amount"`
	ExchangedAmt  float64   `bson:"exchanged_amount" json:"exchanged_amount"`
	Rate          float64   `bson:"rate" json:"rate"`
	Timestamp     time.Time `bson:"timestamp" json:"timestamp"`
	ProcessedAt   time.Time `bson:"processed_at" json:"processed_at"` // Время обработки consumer'ом
}

// KafkaMessage представляет сообщение из Kafka (соответствует формату из gw-currency-wallet)
type KafkaMessage struct {
	TransactionID string    `json:"transaction_id"`
	UserID        string    `json:"user_id"`
	FromCurrency  string    `json:"from_currency"`
	ToCurrency    string    `json:"to_currency"`
	Amount        float64   `json:"amount"`
	ExchangedAmt  float64   `json:"exchanged_amount"`
	Rate          float64   `json:"rate"`
	Timestamp     time.Time `json:"timestamp"`
}
