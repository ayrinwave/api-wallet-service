package models

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Balance   int64     `json:"balance" db:"balance"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
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
