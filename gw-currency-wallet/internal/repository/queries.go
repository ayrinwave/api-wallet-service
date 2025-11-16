package repository

const (
	// Wallet queries
	GetWalletByIDQuery = `
		SELECT id, user_id, currency, balance, version, created_at, updated_at
		FROM wallets
		WHERE id = $1
	`

	// Получить конкретный кошелек пользователя по валюте
	GetWalletByUserAndCurrencyQuery = `
		SELECT id, user_id, currency, balance, version, created_at, updated_at
		FROM wallets
		WHERE user_id = $1 AND currency = $2
	`

	// Получить все кошельки пользователя
	GetAllUserWalletsQuery = `
		SELECT id, user_id, currency, balance, version, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
		ORDER BY currency
	`

	// Создать новый кошелек
	CreateWalletQuery = `
		INSERT INTO wallets (id, user_id, currency, balance)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, currency, balance, version, created_at, updated_at
	`

	// Transaction queries (с FOR UPDATE для блокировки)
	GetWalletStateQuery = `
		SELECT balance 
		FROM wallets
		WHERE id = $1 
		FOR UPDATE
	`

	// Обновление баланса
	UpdateWalletBalanceQuery = `
		UPDATE wallets 
		SET balance = $1
		WHERE id = $2
	`

	// Operation queries
	CreateOperationQuery = `
		INSERT INTO operations (wallet_id, amount, request_id) 
		VALUES ($1, $2, $3)
	`

	CheckOperationExistsQuery = `
		SELECT EXISTS(
			SELECT 1 
			FROM operations 
			WHERE request_id = $1
		)
	`

	// User queries
	CreateUserQuery = `
		INSERT INTO users (id, username, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, created_at, updated_at
	`

	GetUserByUsernameQuery = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	GetUserByEmailQuery = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	GetUserByIDQuery = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	CheckUserExistsByUsernameQuery = `
		SELECT EXISTS(
			SELECT 1 
			FROM users 
			WHERE username = $1
		)
	`

	CheckUserExistsByEmailQuery = `
		SELECT EXISTS(
			SELECT 1 
			FROM users 
			WHERE email = $1
		)
	`
)
