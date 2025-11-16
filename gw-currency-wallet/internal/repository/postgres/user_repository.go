package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	// Создание пользователя
	Create(ctx context.Context, user *models.User) (*models.User, error)
	CreateTx(ctx context.Context, tx pgx.Tx, user *models.User) (*models.User, error)

	// Получение пользователей
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// Проверка существования
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
type PgUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &PgUserRepository{db: db}
}

func (r *PgUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	const op = "repository.Create"

	var createdUser models.User
	err := r.db.QueryRow(
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
		// Проверка на нарушение уникальности
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique_violation
				if pgErr.ConstraintName == "users_username_key" {
					return nil, custom_err.ErrUsernameExists
				}
				if pgErr.ConstraintName == "users_email_key" {
					return nil, custom_err.ErrEmailExists
				}
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	createdUser.PasswordHash = user.PasswordHash // Не возвращается из RETURNING
	return &createdUser, nil
}

// GetByID получает пользователя по ID
func (r *PgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	const op = "repository.GetByID"

	var user models.User
	err := r.db.QueryRow(ctx, repository.GetUserByIDQuery, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

// GetByUsername получает пользователя по username
func (r *PgUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	const op = "repository.GetByUsername"

	var user models.User
	err := r.db.QueryRow(ctx, repository.GetUserByUsernameQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

// GetByEmail получает пользователя по email
func (r *PgUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const op = "repository.GetByEmail"

	var user models.User
	err := r.db.QueryRow(ctx, repository.GetUserByEmailQuery, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

// ExistsByUsername проверяет существование пользователя по username
func (r *PgUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	const op = "repository.ExistsByUsername"

	var exists bool
	err := r.db.QueryRow(ctx, repository.CheckUserExistsByUsernameQuery, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// ExistsByEmail проверяет существование пользователя по email
func (r *PgUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const op = "repository.ExistsByEmail"

	var exists bool
	err := r.db.QueryRow(ctx, repository.CheckUserExistsByEmailQuery, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}
