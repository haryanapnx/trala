CREATE TABLE idempotency_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key             TEXT NOT NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_hash    TEXT NOT NULL,
    response_status SMALLINT NOT NULL,
    response_body   JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The unique constraint is the race guard: concurrent inserts of the same
    -- key cannot both succeed.
    CONSTRAINT idempotency_keys_key_key UNIQUE (key)
);

-- Supports pruning rows outside the 24h replay window.
CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);
