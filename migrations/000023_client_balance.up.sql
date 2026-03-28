-- ══════════════════════════════════════════════════════════════
--  000023 — Баланс и платежи клиентов
-- ══════════════════════════════════════════════════════════════

-- ── 1. Расширяем order_payments ───────────────────────────────
ALTER TABLE order_payments
    ADD COLUMN IF NOT EXISTS client_id         UUID        REFERENCES clients(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS payment_method    VARCHAR(50) DEFAULT 'cash'
        CHECK (payment_method IN ('cash', 'card', 'bank', 'wallet', 'other')),
    ADD COLUMN IF NOT EXISTS is_client_payment BOOLEAN     DEFAULT FALSE;
-- is_client_payment = TRUE — общий платёж клиента (не к конкретному заказу)
-- order_id может быть NULL для таких платежей

-- order_id уже NOT NULL — делаем nullable
ALTER TABLE order_payments
    ALTER COLUMN order_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_order_payments_client ON order_payments(client_id);

-- ── 2. Таблица баланса клиента (переплата) ────────────────────
-- Хранит накопленный баланс клиента сверх оплаченных заказов
CREATE TABLE IF NOT EXISTS client_balance (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id   UUID          NOT NULL UNIQUE REFERENCES clients(id) ON DELETE CASCADE,
    balance     NUMERIC(14,2) DEFAULT 0,   -- положительный = переплата, 0 = без долга
    updated_at  TIMESTAMPTZ   DEFAULT NOW()
);

-- ── 3. VIEW: общий долг и баланс клиентов ────────────────────
CREATE OR REPLACE VIEW client_debt AS
SELECT
    c.id                                                AS client_id,
    c.full_name                                         AS client_name,
    c.phone,
    COUNT(DISTINCT o.id)                                AS total_orders,
    COALESCE(SUM(
        CASE WHEN o.status != 'cancelled'
             THEN COALESCE(o.final_cost, o.estimated_cost, 0) END
    ), 0)                                               AS total_amount,
    COALESCE(SUM(
        CASE WHEN o.status != 'cancelled'
             THEN o.paid_amount END
    ), 0)                                               AS total_paid,
    COALESCE(SUM(
        CASE WHEN o.status != 'cancelled'
             THEN GREATEST(0,
                COALESCE(o.final_cost, o.estimated_cost, 0) - COALESCE(o.paid_amount, 0))
             END
    ), 0)                                               AS total_debt,
    COALESCE(cb.balance, 0)                             AS credit_balance,  -- переплата
    COALESCE(SUM(
        CASE WHEN o.status != 'cancelled'
             THEN GREATEST(0,
                COALESCE(o.final_cost, o.estimated_cost, 0) - COALESCE(o.paid_amount, 0))
             END
    ), 0) - COALESCE(cb.balance, 0)                    AS net_debt   -- долг минус переплата
FROM clients c
LEFT JOIN orders o ON o.client_id = c.id
LEFT JOIN client_balance cb ON cb.client_id = c.id
GROUP BY c.id, c.full_name, c.phone, cb.balance;

-- ── 4. Функция распределения платежа по заказам ───────────────
CREATE OR REPLACE FUNCTION distribute_client_payment(
    p_client_id   UUID,
    p_amount      NUMERIC,
    p_method      VARCHAR,
    p_paid_at     DATE,
    p_notes       TEXT,
    p_user_id     UUID
) RETURNS VOID AS $$
DECLARE
    v_remaining   NUMERIC := p_amount;
    v_order_id    UUID;
    v_order_debt  NUMERIC;
    v_apply       NUMERIC;
    v_pay_id      UUID;
BEGIN
    -- Берём неоплаченные заказы ORDER BY created_at ASC (старые первые)
    FOR v_order_id, v_order_debt IN
        SELECT o.id,
               GREATEST(0, COALESCE(o.final_cost, o.estimated_cost, 0) - COALESCE(o.paid_amount, 0))
        FROM orders o
        WHERE o.client_id = p_client_id
          AND o.status != 'cancelled'
          AND COALESCE(o.final_cost, o.estimated_cost, 0) > COALESCE(o.paid_amount, 0)
        ORDER BY o.created_at ASC
    LOOP
        EXIT WHEN v_remaining <= 0;

        v_apply := LEAST(v_remaining, v_order_debt);

        -- Вставляем платёж к заказу
        v_pay_id := gen_random_uuid();
        INSERT INTO order_payments (id, order_id, client_id, amount, payment_method, paid_at, notes, created_by, is_client_payment)
        VALUES (v_pay_id, v_order_id, p_client_id, v_apply, p_method, p_paid_at, p_notes, p_user_id, TRUE);

        -- Обновляем paid_amount заказа
        UPDATE orders
        SET paid_amount = COALESCE(paid_amount, 0) + v_apply,
            payment_status = CASE
                WHEN COALESCE(paid_amount, 0) + v_apply >= COALESCE(final_cost, estimated_cost, 0) THEN 'paid'
                WHEN COALESCE(paid_amount, 0) + v_apply > 0 THEN 'partial'
                ELSE 'unpaid'
            END
        WHERE id = v_order_id;

        v_remaining := v_remaining - v_apply;
    END LOOP;

    -- Остаток — на баланс клиента
    IF v_remaining > 0 THEN
        INSERT INTO client_balance (id, client_id, balance)
        VALUES (gen_random_uuid(), p_client_id, v_remaining)
        ON CONFLICT (client_id)
        DO UPDATE SET balance = client_balance.balance + v_remaining,
                      updated_at = NOW();

        -- Сохраняем как платёж без заказа
        v_pay_id := gen_random_uuid();
        INSERT INTO order_payments (id, order_id, client_id, amount, payment_method, paid_at, notes, created_by, is_client_payment)
        VALUES (v_pay_id, NULL, p_client_id, v_remaining, p_method, p_paid_at, p_notes, p_user_id, TRUE);
    END IF;
END;
$$ LANGUAGE plpgsql;
