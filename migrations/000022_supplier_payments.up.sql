-- ══════════════════════════════════════════════════════════════
--  000022 — Общие платежи поставщикам
-- ══════════════════════════════════════════════════════════════

-- ── 1. Расширяем warehouse_payments ──────────────────────────
ALTER TABLE warehouse_payments
    ADD COLUMN IF NOT EXISTS supplier_id     UUID        REFERENCES suppliers(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS payment_method  VARCHAR(50) DEFAULT 'cash'
        CHECK (payment_method IN ('cash', 'card', 'bank', 'wallet', 'other')),
    ADD COLUMN IF NOT EXISTS is_supplier_payment BOOLEAN DEFAULT FALSE;
-- is_supplier_payment = TRUE означает что это общий платёж поставщику (не к конкретной накладной)
-- receipt_id может быть NULL для таких платежей

-- receipt_id уже NOT NULL — делаем nullable
ALTER TABLE warehouse_payments
    ALTER COLUMN receipt_id DROP NOT NULL;

-- ── 2. Индексы ────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_warehouse_payments_supplier ON warehouse_payments(supplier_id);

-- ── 3. Обновляем триггер — теперь он срабатывает только для платежей с receipt_id ──
CREATE OR REPLACE FUNCTION recalc_receipt_payment()
RETURNS TRIGGER AS $$
DECLARE
    v_receipt_id UUID;
    v_total      NUMERIC;
    v_paid       NUMERIC;
BEGIN
    v_receipt_id := COALESCE(NEW.receipt_id, OLD.receipt_id);

    -- Пересчитываем только если платёж привязан к накладной
    IF v_receipt_id IS NOT NULL THEN
        SELECT total_amount INTO v_total
        FROM warehouse_receipts
        WHERE id = v_receipt_id;

        SELECT COALESCE(SUM(amount), 0) INTO v_paid
        FROM warehouse_payments
        WHERE receipt_id = v_receipt_id;

        UPDATE warehouse_receipts
        SET
            paid_amount    = v_paid,
            payment_status = CASE
                WHEN v_paid <= 0       THEN 'unpaid'
                WHEN v_paid >= v_total THEN 'paid'
                ELSE                        'partial'
            END
        WHERE id = v_receipt_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ── 4. VIEW: общий долг по поставщикам ───────────────────────
CREATE OR REPLACE VIEW supplier_debt AS
SELECT
    s.id                                            AS supplier_id,
    s.name                                          AS supplier_name,
    COUNT(wr.id)                                    AS total_receipts,
    COALESCE(SUM(wr.total_amount), 0)               AS total_amount,
    COALESCE(SUM(wr.paid_amount),  0)               AS total_paid,
    COALESCE(SUM(wr.total_amount), 0)
        - COALESCE(SUM(wr.paid_amount), 0)          AS total_debt,
    COUNT(CASE WHEN wr.payment_status != 'paid' AND wr.payment_status IS NOT NULL
               THEN 1 END)                          AS unpaid_count
FROM suppliers s
LEFT JOIN warehouse_receipts wr ON wr.supplier_id = s.id
GROUP BY s.id, s.name;
