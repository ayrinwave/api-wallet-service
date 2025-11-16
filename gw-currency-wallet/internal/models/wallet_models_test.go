package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOperationType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		opType   OperationType
		expected bool
	}{
		{
			name:     "Valid DEPOSIT",
			opType:   OperationDeposit,
			expected: true,
		},
		{
			name:     "Valid WITHDRAW",
			opType:   OperationWithdraw,
			expected: true,
		},
		{
			name:     "Invalid empty",
			opType:   OperationType(""),
			expected: false,
		},
		{
			name:     "Invalid random string",
			opType:   OperationType("TRANSFER"),
			expected: false,
		},
		{
			name:     "Invalid lowercase",
			opType:   OperationType("deposit"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opType.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWalletOperationRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     WalletOperationRequest
		shouldValid bool
		description string
	}{
		{
			name: "Valid deposit request",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationDeposit,
				Amount:        100,
				RequestID:     "req-001",
			},
			shouldValid: true,
			description: "All fields are valid",
		},
		{
			name: "Valid withdraw request",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationWithdraw,
				Amount:        50,
				RequestID:     "req-002",
			},
			shouldValid: true,
			description: "All fields are valid",
		},
		{
			name: "Invalid - zero UUID",
			request: WalletOperationRequest{
				WalletID:      uuid.Nil,
				OperationType: OperationDeposit,
				Amount:        100,
				RequestID:     "req-003",
			},
			shouldValid: false,
			description: "WalletID is nil UUID",
		},
		{
			name: "Invalid - empty requestID",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationDeposit,
				Amount:        100,
				RequestID:     "",
			},
			shouldValid: false,
			description: "RequestID is empty",
		},
		{
			name: "Invalid - zero amount",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationDeposit,
				Amount:        0,
				RequestID:     "req-004",
			},
			shouldValid: false,
			description: "Amount is zero",
		},
		{
			name: "Invalid - negative amount",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationDeposit,
				Amount:        -100,
				RequestID:     "req-005",
			},
			shouldValid: false,
			description: "Amount is negative",
		},
		{
			name: "Invalid - invalid operation type",
			request: WalletOperationRequest{
				WalletID:      uuid.New(),
				OperationType: OperationType("INVALID"),
				Amount:        100,
				RequestID:     "req-006",
			},
			shouldValid: false,
			description: "OperationType is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем WalletID
			isValidWallet := tt.request.WalletID != uuid.Nil

			// Проверяем RequestID
			isValidRequestID := tt.request.RequestID != ""

			// Проверяем Amount
			isValidAmount := tt.request.Amount > 0

			// Проверяем OperationType
			isValidOpType := tt.request.OperationType.IsValid()

			isValid := isValidWallet && isValidRequestID && isValidAmount && isValidOpType

			assert.Equal(t, tt.shouldValid, isValid, tt.description)
		})
	}
}

func TestWallet_Fields(t *testing.T) {
	// Проверяем что структура Wallet имеет нужные поля
	walletID := uuid.New()
	wallet := Wallet{
		ID:      walletID,
		Balance: 1000,
	}

	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, int64(1000), wallet.Balance)
	assert.NotNil(t, wallet.CreatedAt)
	assert.NotNil(t, wallet.UpdatedAt)
}

func TestOperationType_String(t *testing.T) {
	tests := []struct {
		name     string
		opType   OperationType
		expected string
	}{
		{
			name:     "DEPOSIT to string",
			opType:   OperationDeposit,
			expected: "DEPOSIT",
		},
		{
			name:     "WITHDRAW to string",
			opType:   OperationWithdraw,
			expected: "WITHDRAW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(tt.opType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
