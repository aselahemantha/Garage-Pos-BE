-- +goose Up
ALTER TABLE parts
    ADD COLUMN IF NOT EXISTS barcode TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_parts_barcode_unique
    ON parts (barcode)
    WHERE barcode IS NOT NULL AND barcode <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_parts_barcode_unique;
ALTER TABLE parts DROP COLUMN IF EXISTS barcode;
