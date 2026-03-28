-- ══════════════════════════════════════════════════════════════
--  000020 — Оплаты приходных накладных
-- ══════════════════════════════════════════════════════════════

-- ── 1. Добавляем поля оплаты в накладную ─────────────────────
ALTER TABLE warehouse_receipts
    ADD COLUMN IF NOT EXISTS paid_amount    NUMERIC(14,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payment_status VARCHAR(20)   DEFAULT 'unpaid'
        CHECK (payment_status IN ('unpaid', 'partial', 'paid')),
    ADD COLUMN IF NOT EXISTS payment_notes  TEXT;

-- ── 2. История платежей по накладной ─────────────────────────
CREATE TABLE warehouse_payments (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    receipt_id  UUID          NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    amount      NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    paid_at     DATE          NOT NULL DEFAULT CURRENT_DATE,
    notes       TEXT,
    created_by  UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_warehouse_payments_receipt ON warehouse_payments(receipt_id);

-- ── 3. Функция пересчёта статуса оплаты ──────────────────────
CREATE OR REPLACE FUNCTION recalc_receipt_payment()
RETURNS TRIGGER AS $$
DECLARE
    v_total NUMERIC;
    v_paid  NUMERIC;
BEGIN
    -- Считаем сумму накладной и сумму оплат
    SELECT total_amount INTO v_total
    FROM warehouse_receipts
    WHERE id = COALESCE(NEW.receipt_id, OLD.receipt_id);

    SELECT COALESCE(SUM(amount), 0) INTO v_paid
    FROM warehouse_payments
    WHERE receipt_id = COALESCE(NEW.receipt_id, OLD.receipt_id);

    -- Обновляем paid_amount и payment_status
    UPDATE warehouse_receipts
    SET
        paid_amount    = v_paid,
        payment_status = CASE
            WHEN v_paid <= 0            THEN 'unpaid'
            WHEN v_paid >= v_total      THEN 'paid'
            ELSE                             'partial'
        END
    WHERE id = COALESCE(NEW.receipt_id, OLD.receipt_id);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_warehouse_payment_insert
    AFTER INSERT OR DELETE ON warehouse_payments
    FOR EACH ROW EXECUTE FUNCTION recalc_receipt_payment();
