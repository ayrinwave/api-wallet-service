ALTER TABLE operations DROP CONSTRAINT operations_request_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_request_id ON operations (wallet_id, request_id);