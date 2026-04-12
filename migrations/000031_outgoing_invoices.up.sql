-- ══════════════════════════════════════════════════════════════
--  000031 — Расходные накладные (списание со склада)
-- ══════════════════════════════════════════════════════════════

-- ── Расходные накладные ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS outgoing_invoices (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50) NOT NULL UNIQUE,  -- ORD-123 или EXT-001
    invoice_type   VARCHAR(20) NOT NULL DEFAULT 'order'
                   CHECK (invoice_type IN ('order', 'external')),
    order_id       UUID REFERENCES orders(id) ON DELETE SET NULL,
    order_number   INT,                          -- кэш номера заказа
    client_name    VARCHAR(255),                 -- для внешней продажи
    notes          TEXT,
    total_cost     NUMERIC(12,2) NOT NULL DEFAULT 0,   -- себестоимость
    total_price    NUMERIC(12,2) NOT NULL DEFAULT 0,   -- цена продажи
    status         VARCHAR(20) NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'confirmed', 'cancelled')),
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Позиции расходной накладной ───────────────────────────────
CREATE TABLE IF NOT EXISTS outgoing_invoice_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      UUID NOT NULL REFERENCES outgoing_invoices(id) ON DELETE CASCADE,
    item_id         UUID NOT NULL REFERENCES warehouse_items(id) ON DELETE RESTRICT,
    item_name       VARCHAR(255) NOT NULL,        -- кэш названия
    unit            VARCHAR(50)  NOT NULL,        -- кэш единицы измерения
    quantity        NUMERIC(12,3) NOT NULL,
    cost_price      NUMERIC(12,2) NOT NULL DEFAULT 0,  -- себестоимость за ед.
    sale_price      NUMERIC(12,2) NOT NULL DEFAULT 0,  -- цена продажи за ед.
    total_cost      NUMERIC(12,2) GENERATED ALWAYS AS (quantity * cost_price) STORED,
    total_price     NUMERIC(12,2) GENERATED ALWAYS AS (quantity * sale_price) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outgoing_invoices_order    ON outgoing_invoices(order_id);
CREATE INDEX IF NOT EXISTS idx_outgoing_invoices_status   ON outgoing_invoices(status);
CREATE INDEX IF NOT EXISTS idx_outgoing_invoice_items_inv ON outgoing_invoice_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_outgoing_invoice_items_item ON outgoing_invoice_items(item_id);

-- ── Добавляем sale_price в warehouse_items если нет ───────────
ALTER TABLE warehouse_items ADD COLUMN IF NOT EXISTS sale_price NUMERIC(12,2) DEFAULT 0;

-- ── Счётчик для EXT накладных ─────────────────────────────────
CREATE SEQUENCE IF NOT EXISTS outgoing_invoice_ext_seq START 1;
