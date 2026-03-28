-- ================================================================
-- МАТЕРИАЛЫ ЗАКАЗА
-- Накладная на материалы — привязана к заказу и этапу
-- ================================================================

CREATE TABLE order_materials (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage_id     UUID          REFERENCES order_stages(id) ON DELETE SET NULL,
    stage_name   VARCHAR(50),  -- название этапа (для отображения)
    name         VARCHAR(300)  NOT NULL,
    quantity     NUMERIC(10,3) NOT NULL DEFAULT 1,
    unit         VARCHAR(30)   DEFAULT 'шт'
                     CHECK (unit IN ('шт','м','м²','м³','кг','л','упак','лист')),
    unit_price   NUMERIC(12,2) DEFAULT 0,
    total_price  NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    supplier     VARCHAR(200),
    notes        TEXT,
    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_materials_order_id ON order_materials(order_id);
CREATE INDEX idx_order_materials_stage_id ON order_materials(stage_id);
