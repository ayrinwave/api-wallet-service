package repository

const (
	GetWalletByIDQuery = `
        SELECT id, balance, created_at, updated_at
        FROM wallets
        WHERE id = $1
    `

	GetWalletStateQuery = `
    SELECT balance, version 
    FROM wallets
    WHERE id = $1 
    FOR UPDATE 
	`

	CreateOperationQuery = `
		INSERT INTO operations (wallet_id, amount, request_id) 
		VALUES ($1, $2, $3)
	`

	CheckOperationExistsQuery = `
	SELECT 
	EXISTS(SELECT 1 FROM operations 
	WHERE request_id = $1)
	`

	UpsertWalletBalanceQuery = `
       INSERT INTO wallets (id, balance, version, created_at, updated_at)
       VALUES ($1, $2, 1, NOW(), NOW())
       ON CONFLICT (id) DO UPDATE 
       SET 
           balance = EXCLUDED.balance,
           updated_at = NOW()
   `

	UpdateWalletBalanceWithLockQuery = `
    UPDATE wallets 
    SET 
        balance = $1, 
        version = $2 + 1, 
        updated_at = NOW()
    WHERE id = $3     
      AND version = $2    
    RETURNING version
	`
)
