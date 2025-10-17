package models

import "github.com/google/uuid"

type OperationType string

const (
	DepositOperation  OperationType = "DEPOSIT"
	WithdrawOperation OperationType = "WITHDRAW"
)

func (ot OperationType) IsValid() bool {
	switch ot {
	case DepositOperation, WithdrawOperation:
		return true
	}
	return false
}

type WalletOperationRequest struct {
	WalletID      uuid.UUID     `json:"walletId"`
	OperationType OperationType `json:"operationType"`
	Amount        int64         `json:"amount"`
	RequestID     uuid.UUID     `json:"requestId"`
}
