-- +goose Up
ALTER TABLE job_cards
    ADD COLUMN IF NOT EXISTS tracking_token VARCHAR(64);

UPDATE job_cards
SET tracking_token = substr(md5(random()::text || clock_timestamp()::text || id::text), 1, 20)
WHERE tracking_token IS NULL OR tracking_token = '';

ALTER TABLE job_cards
    ALTER COLUMN tracking_token SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_job_cards_tracking_token
    ON job_cards (tracking_token);

-- +goose Down
DROP INDEX IF EXISTS idx_job_cards_tracking_token;
ALTER TABLE job_cards DROP COLUMN IF EXISTS tracking_token;
