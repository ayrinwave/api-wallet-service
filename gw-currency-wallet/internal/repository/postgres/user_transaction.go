package postgres

import (
	"context"
	"fmt"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/repository"

	"github.com/jackc/pgx/v5"
)

// CreateTx создает пользователя внутри транзакции
func (r *PgUserRepository) CreateTx(ctx context.Context, tx pgx.Tx, user *models.User) (*models.User, error) {
	const op = "repository.CreateTx"

	var createdUser models.User
	err := tx.QueryRow(
		ctx,
		repository.CreateUserQuery,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
	).Scan(
		&createdUser.ID,
		&createdUser.Username,
		&createdUser.Email,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	createdUser.PasswordHash = user.PasswordHash
	return &createdUser, nil
}
