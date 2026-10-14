-- Store the exact response bytes so a replayed response is identical to the
-- first one (jsonb would normalize key order and whitespace).
ALTER TABLE idempotency_keys
    ALTER COLUMN response_body TYPE TEXT USING response_body::text;
