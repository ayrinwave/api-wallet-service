package models

import (
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя системы
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Email    string `json:"email" validate:"required,email"`
}

// RegisterResponse ответ на регистрацию
type RegisterResponse struct {
	Message string `json:"message"`
}

// LoginRequest запрос на авторизацию
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse ответ на авторизацию
type LoginResponse struct {
	Token string `json:"token"`
}

// JWTClaims кастомные claims для JWT токена
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
}
