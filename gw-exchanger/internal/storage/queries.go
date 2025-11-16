package storage

const (
	GetAllRatesQuery = `
		SELECT id, currency, rate, updated_at
		FROM exchange_rates
		ORDER BY currency
	`

	GetRateByCurrencyQuery = `
		SELECT id, currency, rate, updated_at
		FROM exchange_rates		
		WHERE currency = $1
	`
)
