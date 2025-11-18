-- Таблица для истории операций обмена валют
CREATE TABLE IF NOT EXISTS exchange_operations (
                                                   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_currency VARCHAR(3) NOT NULL CHECK (from_currency IN ('USD', 'RUB', 'EUR')),
    to_currency VARCHAR(3) NOT NULL CHECK (to_currency IN ('USD', 'RUB', 'EUR')),
    amount BIGINT NOT NULL CHECK (amount > 0),           -- Сумма в минимальных единицах (копейки/центы)
    exchanged_amount BIGINT NOT NULL CHECK (exchanged_amount > 0),
    rate NUMERIC(20, 10) NOT NULL CHECK (rate > 0),      -- Курс обмена с высокой точностью
    request_id TEXT NOT NULL UNIQUE,                     -- Для идемпотентности
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),

    -- Проверка: нельзя обменять одну валюту на себя же
    CONSTRAINT check_different_currencies CHECK (from_currency != to_currency)
    );

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_exchange_operations_user_id ON exchange_operations(user_id);
CREATE INDEX IF NOT EXISTS idx_exchange_operations_created_at ON exchange_operations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exchange_operations_request_id ON exchange_operations(request_id);

-- Комментарии для документации
COMMENT ON TABLE exchange_operations IS 'История операций обмена валют';
COMMENT ON COLUMN exchange_operations.amount IS 'Сумма в исходной валюте (в минимальных единицах)';
COMMENT ON COLUMN exchange_operations.exchanged_amount IS 'Сумма после обмена (в минимальных единицах)';
COMMENT ON COLUMN exchange_operations.rate IS 'Курс обмена на момент операции';
COMMENT ON COLUMN exchange_operations.request_id IS 'Уникальный идентификатор запроса для идемпотентности';