CREATE TABLE products (
    sku TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE inventory_movements (
    event_id TEXT PRIMARY KEY,
    sku TEXT NOT NULL REFERENCES products (sku),
    movement_type TEXT NOT NULL CHECK (movement_type IN ('IN', 'OUT')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_inventory_movements_sku_occurred_at
    ON inventory_movements (sku, occurred_at DESC, event_id);

CREATE TABLE product_stock (
    sku TEXT PRIMARY KEY REFERENCES products (sku),
    quantity BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ingest_errors (
    id BIGSERIAL PRIMARY KEY,
    source_file TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    raw_line TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ingest_errors_source_file_line_number
    ON ingest_errors (source_file, line_number);
