package repository

const (
	GetWalletByIDQuery = `
        SELECT id, balance, created_at, updated_at
        FROM wallets
        WHERE id = $1
    `

	// ✅ УБРАЛИ version - он больше не нужен
	GetWalletStateQuery = `
        SELECT balance 
        FROM wallets
        WHERE id = $1 
        FOR UPDATE
    `

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

	// ✅ УПРОСТИЛИ - убрали проверку version
	UpdateWalletBalanceQuery = `
        UPDATE wallets 
        SET balance = $1
        WHERE id = $2
    `
)
