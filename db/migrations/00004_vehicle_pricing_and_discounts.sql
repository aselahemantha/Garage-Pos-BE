-- +goose Up
-- Vehicle-specific service pricing overrides (e.g. Full Service for Wagon R vs Alto vs Premio)
CREATE TABLE service_vehicle_prices (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_id BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    vehicle_make TEXT NOT NULL DEFAULT '',
    vehicle_model TEXT NOT NULL,
    price BIGINT NOT NULL CHECK (price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_service_vehicle_prices_unique
    ON service_vehicle_prices(service_id, LOWER(TRIM(vehicle_make)), LOWER(TRIM(vehicle_model)));

CREATE INDEX idx_service_vehicle_prices_service_id
    ON service_vehicle_prices(service_id);

-- Track overall bill-level discount separately from individual line-item discounts
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS bill_discount BIGINT NOT NULL DEFAULT 0;

-- Backfill bill_discount on existing invoices from (discount_total - sum(item discounts))
UPDATE invoices i
SET bill_discount = GREATEST(0, i.discount_total - COALESCE((
    SELECT SUM(ii.discount)
    FROM invoice_items ii
    WHERE ii.invoice_id = i.id
), 0))
WHERE i.discount_total > 0;

-- +goose Down
ALTER TABLE invoices DROP COLUMN IF EXISTS bill_discount;
DROP TABLE IF EXISTS service_vehicle_prices;
