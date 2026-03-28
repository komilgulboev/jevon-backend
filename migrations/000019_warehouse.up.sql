-- ══════════════════════════════════════════════════════════════
--  000019 — Склад (только материалы)
-- ══════════════════════════════════════════════════════════════

-- ── 1. Поставщики ─────────────────────────────────────────────
CREATE TABLE suppliers (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    phone       VARCHAR(50),
    email       VARCHAR(100),
    address     TEXT,
    notes       TEXT,
    is_active   BOOLEAN      DEFAULT TRUE,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

-- ── 2. Единицы измерения ──────────────────────────────────────
CREATE TABLE units (
    id      SERIAL      PRIMARY KEY,
    name    VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO units (name) VALUES
    ('шт'),
    ('м²'),
    ('м.п.'),
    ('кг'),
    ('л'),
    ('упак'),
    ('лист');

-- ── 3. Номенклатура ───────────────────────────────────────────
CREATE TABLE warehouse_items (
    id        UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    name      VARCHAR(255)  NOT NULL,
    article   VARCHAR(100),
    category  VARCHAR(100),
    unit_id   INTEGER       REFERENCES units(id),
    min_stock NUMERIC(12,3) DEFAULT 0,
    notes     TEXT,
    is_active BOOLEAN       DEFAULT TRUE,
    created_at TIMESTAMPTZ  DEFAULT NOW(),
    updated_at TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_warehouse_items_category ON warehouse_items(category);
CREATE INDEX idx_warehouse_items_article  ON warehouse_items(article);

-- ── 4. Приходные накладные ────────────────────────────────────
CREATE TABLE warehouse_receipts (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    number       VARCHAR(100),
    supplier_id  UUID          REFERENCES suppliers(id) ON DELETE SET NULL,
    receipt_date DATE          NOT NULL DEFAULT CURRENT_DATE,
    total_amount NUMERIC(14,2) DEFAULT 0,
    notes        TEXT,
    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_warehouse_receipts_supplier ON warehouse_receipts(supplier_id);
CREATE INDEX idx_warehouse_receipts_date     ON warehouse_receipts(receipt_date);

-- ── 5. Строки приходной накладной ─────────────────────────────
CREATE TABLE warehouse_receipt_items (
    id         UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    receipt_id UUID          NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    item_id    UUID          NOT NULL REFERENCES warehouse_items(id),
    quantity   NUMERIC(12,3) NOT NULL CHECK (quantity > 0),
    price      NUMERIC(12,2) NOT NULL DEFAULT 0,
    total      NUMERIC(14,2) GENERATED ALWAYS AS (quantity * price) STORED,
    notes      TEXT
);

CREATE INDEX idx_receipt_items_receipt ON warehouse_receipt_items(receipt_id);
CREATE INDEX idx_receipt_items_item    ON warehouse_receipt_items(item_id);

-- ── 6. Расход материалов ──────────────────────────────────────
CREATE TABLE warehouse_expenses (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id      UUID          NOT NULL REFERENCES warehouse_items(id),
    quantity     NUMERIC(12,3) NOT NULL CHECK (quantity > 0),
    price        NUMERIC(12,2) DEFAULT 0,
    total        NUMERIC(14,2) GENERATED ALWAYS AS (quantity * price) STORED,
    order_id     UUID          REFERENCES orders(id)   ON DELETE SET NULL,
    project_id   UUID          REFERENCES projects(id) ON DELETE SET NULL,
    expense_date DATE          NOT NULL DEFAULT CURRENT_DATE,
    notes        TEXT,
    created_by   UUID          REFERENCES users(id)    ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_warehouse_expenses_item    ON warehouse_expenses(item_id);
CREATE INDEX idx_warehouse_expenses_order   ON warehouse_expenses(order_id);
CREATE INDEX idx_warehouse_expenses_project ON warehouse_expenses(project_id);
CREATE INDEX idx_warehouse_expenses_date    ON warehouse_expenses(expense_date);

-- ── 7. Триггеры updated_at ────────────────────────────────────
CREATE OR REPLACE FUNCTION warehouse_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_suppliers_updated
    BEFORE UPDATE ON suppliers
    FOR EACH ROW EXECUTE FUNCTION warehouse_set_updated_at();

CREATE TRIGGER trg_warehouse_items_updated
    BEFORE UPDATE ON warehouse_items
    FOR EACH ROW EXECUTE FUNCTION warehouse_set_updated_at();

CREATE TRIGGER trg_warehouse_receipts_updated
    BEFORE UPDATE ON warehouse_receipts
    FOR EACH ROW EXECUTE FUNCTION warehouse_set_updated_at();

CREATE TRIGGER trg_warehouse_expenses_updated
    BEFORE UPDATE ON warehouse_expenses
    FOR EACH ROW EXECUTE FUNCTION warehouse_set_updated_at();

-- ── 8. VIEW: баланс по материалам ────────────────────────────
CREATE VIEW warehouse_balance AS
SELECT
    wi.id,
    wi.name,
    wi.article,
    wi.category,
    u.name                                              AS unit,
    wi.min_stock,
    COALESCE(r.total_in,  0)                            AS total_in,
    COALESCE(e.total_out, 0)                            AS total_out,
    COALESCE(r.total_in, 0) - COALESCE(e.total_out, 0) AS balance,
    COALESCE(r.avg_price, 0)                            AS avg_price,
    wi.is_active
FROM warehouse_items wi
LEFT JOIN units u ON u.id = wi.unit_id
LEFT JOIN (
    SELECT
        ri.item_id,
        SUM(ri.quantity)                             AS total_in,
        SUM(ri.total) / NULLIF(SUM(ri.quantity), 0) AS avg_price
    FROM warehouse_receipt_items ri
    GROUP BY ri.item_id
) r ON r.item_id = wi.id
LEFT JOIN (
    SELECT item_id, SUM(quantity) AS total_out
    FROM warehouse_expenses
    GROUP BY item_id
) e ON e.item_id = wi.id;
