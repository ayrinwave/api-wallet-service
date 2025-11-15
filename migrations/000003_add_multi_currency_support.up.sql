-- Добавляем таблицу пользователей (нужна для связи кошельков с пользователями)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );

-- Добавляем индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Добавляем триггер для обновления updated_at
CREATE TRIGGER trigger_update_user_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_wallet_timestamp();

-- Добавляем поля currency и user_id в таблицу wallets
ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS user_id UUID;

-- Создаем временный constraint для существующих записей
-- (при первом запуске у вас могут быть тестовые кошельки без user_id)
ALTER TABLE wallets
    ALTER COLUMN user_id DROP DEFAULT;

-- Добавляем внешний ключ на users (можно сделать после того как заполним user_id)
-- Пока оставляем nullable, позже сделаем NOT NULL
-- ALTER TABLE wallets
--     ADD CONSTRAINT fk_wallets_user_id
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Создаем уникальный индекс: один пользователь может иметь только один кошелек каждой валюты
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_currency
    ON wallets(user_id, currency)
    WHERE user_id IS NOT NULL;

-- Добавляем constraint на допустимые валюты
ALTER TABLE wallets
    ADD CONSTRAINT check_currency
        CHECK (currency IN ('USD', 'RUB', 'EUR'));

-- Добавляем индекс для быстрого поиска кошельков пользователя
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

-- Комментарии для документации
COMMENT ON COLUMN wallets.currency IS 'Currency code: USD, RUB, or EUR';
COMMENT ON COLUMN wallets.user_id IS 'Reference to the user who owns this wallet';
COMMENT ON TABLE users IS 'User accounts for the wallet service';