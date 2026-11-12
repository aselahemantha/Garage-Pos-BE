-- +goose Up

-- Parts master
CREATE TABLE parts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    category TEXT,
    unit TEXT NOT NULL DEFAULT 'pcs',
    cost_price BIGINT NOT NULL DEFAULT 0,
    sell_price BIGINT NOT NULL DEFAULT 0,
    reorder_level NUMERIC(12,3) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_parts_sku ON parts(sku);
CREATE INDEX idx_parts_name ON parts(name);

-- Stock movements ledger (on-hand = SUM(qty))
CREATE TABLE stock_movements (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    part_id BIGINT NOT NULL REFERENCES parts(id) ON DELETE RESTRICT,
    qty NUMERIC(12,3) NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('stock_in', 'job_use', 'job_return', 'purchase', 'adjustment')),
    ref_type TEXT,
    ref_id BIGINT,
    unit_cost BIGINT,
    note TEXT,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_movements_part_id ON stock_movements(part_id);

-- Job Cards
CREATE TABLE job_cards (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_no TEXT NOT NULL UNIQUE,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    advisor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    mileage_in INT NOT NULL DEFAULT 0,
    complaint TEXT,
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'delivered', 'cancelled')),
    promised_by TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_cards_status ON job_cards(status);
CREATE INDEX idx_job_cards_vehicle_id ON job_cards(vehicle_id);
CREATE INDEX idx_job_cards_created_at ON job_cards(created_at);

-- Job Items
CREATE TABLE job_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_card_id BIGINT NOT NULL REFERENCES job_cards(id) ON DELETE CASCADE,
    item_type TEXT NOT NULL CHECK (item_type IN ('service', 'part')),
    service_id BIGINT REFERENCES services(id) ON DELETE RESTRICT,
    part_id BIGINT REFERENCES parts(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    qty NUMERIC(12,3) NOT NULL DEFAULT 1,
    unit_price BIGINT NOT NULL DEFAULT 0,
    discount BIGINT NOT NULL DEFAULT 0,
    line_total BIGINT NOT NULL DEFAULT 0,
    mechanic_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_items_job_card_id ON job_items(job_card_id);

-- +goose Down
DROP TABLE IF EXISTS job_items;
DROP TABLE IF EXISTS job_cards;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS parts;
