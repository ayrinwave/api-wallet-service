ALTER TABLE wallets ADD CONSTRAINT balance_non_negative CHECK (balance >= 0);

-- Создаём новый уникальный индекс:
CREATE UNIQUE INDEX IF NOT EXISTS idx_operations_request_id ON operations(request_id);
