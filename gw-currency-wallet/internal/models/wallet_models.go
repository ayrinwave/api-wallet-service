package models

import (
	"time"

	"github.com/google/uuid"
)

// Wallet представляет кошелек пользователя в определенной валюте
type Wallet struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Currency  string    `json:"currency" db:"currency"`
	Balance   int64     `json:"balance" db:"balance"`
	Version   int64     `json:"version" db:"version"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Currency типы
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyRUB Currency = "RUB"
	CurrencyEUR Currency = "EUR"
)

// IsValid проверяет валидность валюты
func (c Currency) IsValid() bool {
	return c == CurrencyUSD || c == CurrencyRUB || c == CurrencyEUR
}

// SupportedCurrencies возвращает список поддерживаемых валют
func SupportedCurrencies() []Currency {
	return []Currency{CurrencyUSD, CurrencyRUB, CurrencyEUR}
}

type OperationType string

const (
	OperationDeposit  OperationType = "DEPOSIT"
	OperationWithdraw OperationType = "WITHDRAW"
)

func (ot OperationType) IsValid() bool {
	return ot == OperationDeposit || ot == OperationWithdraw
}

type WalletOperationRequest struct {
	WalletID      uuid.UUID     `json:"walletID"`
	OperationType OperationType `json:"operationType"`
	Amount        int64         `json:"amount"`
	RequestID     string        `json:"requestID"`
}

// UserBalanceResponse ответ с балансами пользователя по всем валютам
type UserBalanceResponse struct {
	USD float64 `json:"USD"`
	RUB float64 `json:"RUB"`
	EUR float64 `json:"EUR"`
}

// DepositRequest запрос на пополнение
type DepositRequest struct {
	Amount    float64  `json:"amount"`
	Currency  Currency `json:"currency"`
	RequestID string   `json:"requestID"`
}

// WithdrawRequest запрос на вывод средств
type WithdrawRequest struct {
	Amount    float64  `json:"amount"`
	Currency  Currency `json:"currency"`
	RequestID string   `json:"requestID"`
}

// BalanceOperationResponse ответ на операцию пополнения/вывода
type BalanceOperationResponse struct {
	Message    string              `json:"message"`
	NewBalance UserBalanceResponse `json:"new_balance"`
}

// AmountToMinorUnits конвертирует сумму в основных единицах в минимальные единицы
func AmountToMinorUnits(amount float64) int64 {
	return int64(amount * 100)
}

// AmountFromMinorUnits конвертирует минимальные единицы в основные
func AmountFromMinorUnits(amount int64) float64 {
	return float64(amount) / 100.0
}
