package custom_err

import "errors"

var (
	ErrNotFound          = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrDuplicateRequest  = errors.New("duplicate request")
	ErrInvalidAmount     = errors.New("amount must be positive")
)
