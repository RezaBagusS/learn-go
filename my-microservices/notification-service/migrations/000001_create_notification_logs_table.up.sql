-- 000001_create_notification_logs_table.up.sql

CREATE TABLE IF NOT EXISTS notification_logs (
    id              VARCHAR(36)     PRIMARY KEY,
    event_type      VARCHAR(100)    NOT NULL,
    reference_id    VARCHAR(255)    NOT NULL,
    partner_id      VARCHAR(100)    NOT NULL,
    callback_url    TEXT            NOT NULL,
    payload         JSONB           NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'pending',  -- pending | success | failed
    retry_count     INT             NOT NULL DEFAULT 0,
    last_error      TEXT            NOT NULL DEFAULT '',
    http_status_code INT            NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ     NULL
);

-- Index untuk query retry (FindPendingRetries)
CREATE INDEX IF NOT EXISTS idx_notification_logs_status_retry
    ON notification_logs (status, retry_count, updated_at);

-- Index untuk idempotency check (FindByReferenceID)
CREATE INDEX IF NOT EXISTS idx_notification_logs_reference_id
    ON notification_logs (reference_id);

-- Index untuk monitoring per partner
CREATE INDEX IF NOT EXISTS idx_notification_logs_partner_id
    ON notification_logs (partner_id, created_at DESC);