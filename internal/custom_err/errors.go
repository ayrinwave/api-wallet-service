package custom_err

import "errors"

var (
	// Wallet errors
	ErrNotFound          = errors.New("resource not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrDuplicateRequest  = errors.New("duplicate request")

	// User errors
	ErrUsernameExists     = errors.New("username already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUnauthorized       = errors.New("unauthorized")

	// Validation errors
	ErrInvalidInput    = errors.New("invalid input")
	ErrInvalidCurrency = errors.New("invalid currency")
	ErrInvalidAmount   = errors.New("invalid amount")
)
