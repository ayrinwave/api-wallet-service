package models

// ExchangeRequest запрос на обмен валют
type ExchangeRequest struct {
	FromCurrency Currency `json:"from_currency"` // USD, RUB, EUR
	ToCurrency   Currency `json:"to_currency"`   // USD, RUB, EUR
	Amount       float64  `json:"amount"`        // Сумма в исходной валюте
	RequestID    string   `json:"requestID"`     // Для идемпотентности
}

// ExchangeResponse ответ на обмен валют
type ExchangeResponse struct {
	Message         string  `json:"message"`
	ExchangedAmount float64 `json:"exchanged_amount"` // Сумма после обмена
	Rate            float64 `json:"rate,omitempty"`   // Курс обмена (опционально)
	// NewBalance можно добавить позже, если нужно возвращать балансы
}

// ExchangeRatesResponse ответ с курсами валют
type ExchangeRatesResponse struct {
	Rates map[string]float64 `json:"rates"` // USD: 1.0, RUB: 95.5, EUR: 0.92
}
