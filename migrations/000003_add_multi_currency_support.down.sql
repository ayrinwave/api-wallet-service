-- Удаляем constraint на валюты
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS check_currency;

-- Удаляем индексы
DROP INDEX IF EXISTS idx_wallets_user_id;
DROP INDEX IF EXISTS idx_wallets_user_currency;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

-- Удаляем внешний ключ (если был добавлен)
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS fk_wallets_user_id;

-- Удаляем добавленные колонки из wallets
ALTER TABLE wallets
DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS currency;

-- Удаляем триггер для users
DROP TRIGGER IF EXISTS trigger_update_user_timestamp ON users;

-- Удаляем таблицу users
DROP TABLE IF EXISTS users;