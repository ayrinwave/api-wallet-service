-- Таблица для хранения курсов валют
CREATE TABLE IF NOT EXISTS exchange_rates (
                                              id SERIAL PRIMARY KEY,
                                              currency VARCHAR(3) NOT NULL UNIQUE,
    rate DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

-- Индекс для быстрого поиска по валюте
CREATE INDEX IF NOT EXISTS idx_exchange_rates_currency ON exchange_rates(currency);

-- Вставляем начальные данные (курсы относительно USD)
INSERT INTO exchange_rates (currency, rate) VALUES
                                                ('USD', 1.0),
                                                ('RUB', 95.5),
                                                ('EUR', 0.92)
    ON CONFLICT (currency) DO NOTHING;

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_exchange_rates_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_exchange_rates_timestamp
    BEFORE UPDATE ON exchange_rates
    FOR EACH ROW
    EXECUTE FUNCTION update_exchange_rates_timestamp();