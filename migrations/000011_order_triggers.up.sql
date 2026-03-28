-- ================================================================
-- ТРИГГЕР: автоматическое создание этапов при создании заказа
-- Этапы зависят от типа заказа (order_type)
-- ================================================================

CREATE OR REPLACE FUNCTION create_order_stages()
RETURNS TRIGGER AS $$
BEGIN
    -- Заказ цеха (мебель под заказ)
    IF NEW.order_type = 'workshop' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',      1, 'in_progress'),
            (NEW.id, 'measure',     2, 'pending'),
            (NEW.id, 'design',      3, 'pending'),
            (NEW.id, 'purchase',    4, 'pending'),
            (NEW.id, 'production',  5, 'pending'),
            (NEW.id, 'assembly',    6, 'pending'),
            (NEW.id, 'delivery',    7, 'pending'),
            (NEW.id, 'handover',    8, 'pending');

    -- Услуга распила
    ELSIF NEW.order_type = 'cutting' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',    1, 'in_progress'),
            (NEW.id, 'material',  2, 'pending'),
            (NEW.id, 'sawing',    3, 'pending'),
            (NEW.id, 'edging',    4, 'pending'),
            (NEW.id, 'drilling',  5, 'pending'),
            (NEW.id, 'packing',   6, 'pending'),
            (NEW.id, 'shipment',  7, 'pending');

    -- Услуга покраски
    ELSIF NEW.order_type = 'painting' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',    1, 'in_progress'),
            (NEW.id, 'calculate', 2, 'pending'),
            (NEW.id, 'sanding',   3, 'pending'),
            (NEW.id, 'priming',   4, 'pending'),
            (NEW.id, 'painting',  5, 'pending'),
            (NEW.id, 'delivery',  6, 'pending');

    -- Услуга ЧПУ
    ELSIF NEW.order_type = 'cnc' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',    1, 'in_progress'),
            (NEW.id, 'calculate', 2, 'pending'),
            (NEW.id, 'cnc_work',  3, 'pending'),
            (NEW.id, 'delivery',  4, 'pending');

    -- Мягкая мебель обивка
    ELSIF NEW.order_type = 'soft_fabric' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',    1, 'in_progress'),
            (NEW.id, 'calculate', 2, 'pending'),
            (NEW.id, 'assign',    3, 'pending'),
            (NEW.id, 'work',      4, 'pending'),
            (NEW.id, 'delivery',  5, 'pending');

    -- Производство мягкой мебели
    ELSIF NEW.order_type = 'soft_furniture' THEN
        INSERT INTO order_stages (order_id, stage, stage_order, status) VALUES
            (NEW.id, 'intake',      1, 'in_progress'),
            (NEW.id, 'design',      2, 'pending'),
            (NEW.id, 'purchase',    3, 'pending'),
            (NEW.id, 'production',  4, 'pending'),
            (NEW.id, 'delivery',    5, 'pending');
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_create_order_stages
    AFTER INSERT ON orders
    FOR EACH ROW EXECUTE FUNCTION create_order_stages();

-- ================================================================
-- ТРИГГЕР: автоматический переход на следующий этап
-- При завершении этапа (status → done) активирует следующий
-- ================================================================

CREATE OR REPLACE FUNCTION advance_order_stage()
RETURNS TRIGGER AS $$
DECLARE
    v_next_stage   order_stages%ROWTYPE;
    v_current_order orders%ROWTYPE;
BEGIN
    -- Только при переходе в 'done'
    IF NEW.status = 'done' AND OLD.status != 'done' THEN
        -- Устанавливаем время завершения
        NEW.finished_at := NOW();

        -- Ищем следующий этап
        SELECT * INTO v_next_stage
        FROM order_stages
        WHERE order_id = NEW.order_id
          AND stage_order = NEW.stage_order + 1
          AND status = 'pending';

        IF FOUND THEN
            -- Активируем следующий этап
            UPDATE order_stages
            SET status = 'in_progress', started_at = NOW()
            WHERE id = v_next_stage.id;

            -- Обновляем текущий этап в заказе
            UPDATE orders
            SET current_stage = v_next_stage.stage,
                status = 'in_progress'
            WHERE id = NEW.order_id;
        ELSE
            -- Все этапы завершены → заказ выполнен
            UPDATE orders
            SET status = 'done',
                finished_at = NOW()
            WHERE id = NEW.order_id;
        END IF;

        -- Записываем в историю
        INSERT INTO order_history (order_id, from_stage, to_stage, changed_by)
        VALUES (
            NEW.order_id,
            NEW.stage,
            COALESCE(v_next_stage.stage, 'done'),
            NEW.assigned_to
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_advance_order_stage
    BEFORE UPDATE ON order_stages
    FOR EACH ROW EXECUTE FUNCTION advance_order_stage();

-- ================================================================
-- ИСТОРИЯ ПЕРЕХОДОВ ЗАКАЗА
-- ================================================================

CREATE TABLE order_history (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id    UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_stage  VARCHAR(50),
    to_stage    VARCHAR(50),
    changed_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    comment     TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_history_order_id ON order_history(order_id);

-- ================================================================
-- ТРИГГЕР: автоматическое обновление оплаты заказа
-- При добавлении платежа пересчитывает paid_amount и payment_status
-- ================================================================

CREATE OR REPLACE FUNCTION update_order_payment_status()
RETURNS TRIGGER AS $$
DECLARE
    v_total_paid NUMERIC(12,2);
    v_final_cost NUMERIC(12,2);
BEGIN
    -- Считаем общую сумму оплат
    SELECT COALESCE(SUM(amount), 0) INTO v_total_paid
    FROM order_payments
    WHERE order_id = NEW.order_id;

    -- Получаем итоговую стоимость
    SELECT COALESCE(final_cost, estimated_cost, 0) INTO v_final_cost
    FROM orders
    WHERE id = NEW.order_id;

    -- Обновляем заказ
    UPDATE orders SET
        paid_amount    = v_total_paid,
        payment_status = CASE
            WHEN v_total_paid <= 0         THEN 'unpaid'
            WHEN v_total_paid < v_final_cost THEN 'partial'
            ELSE 'paid'
        END
    WHERE id = NEW.order_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_payment_status
    AFTER INSERT OR UPDATE ON order_payments
    FOR EACH ROW EXECUTE FUNCTION update_order_payment_status();
