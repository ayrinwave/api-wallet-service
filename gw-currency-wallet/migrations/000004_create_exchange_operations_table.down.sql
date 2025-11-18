-- Откат миграции: удаление таблицы exchange_operations
DROP INDEX IF EXISTS idx_exchange_operations_request_id;
DROP INDEX IF EXISTS idx_exchange_operations_created_at;
DROP INDEX IF EXISTS idx_exchange_operations_user_id;

DROP TABLE IF EXISTS exchange_operations;