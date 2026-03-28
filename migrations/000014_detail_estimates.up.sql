-- ================================================================
-- СМЕТА ДЕТАЛЕЙ — ЧПУ, Покраска, Мягкая мебель
-- Общая таблица для всех типов, тип определяется service_type
-- ================================================================

CREATE TABLE order_detail_estimates (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    -- Тип услуги
    service_type VARCHAR(30)   NOT NULL
                     CHECK (service_type IN ('cnc', 'painting', 'soft', 'cutting')),

    -- Заголовок раздела (вводится вручную или из настроек)
    section_title VARCHAR(200),  -- "От идеи к идеальной детали", "От эскиза до идеального цвета"
    section_notes TEXT,          -- "Цвет: Под AGT398 2кг", примечания к разделу

    -- Строка детали
    row_order    INT           DEFAULT 0,
    detail_name  VARCHAR(300)  NOT NULL,  -- "Фаска 2д 1800×1200=2шт"
    width_mm     NUMERIC(8,1),            -- ширина мм
    height_mm    NUMERIC(8,1),            -- высота мм
    quantity     INT           DEFAULT 1, -- количество штук
    area_m2      NUMERIC(10,4) GENERATED ALWAYS AS (
                     ROUND((width_mm / 1000.0) * (height_mm / 1000.0) * quantity, 4)
                 ) STORED,               -- авторасчёт м²
    unit_price   NUMERIC(10,2) DEFAULT 0, -- цена за м²
    total_price  NUMERIC(12,2) GENERATED ALWAYS AS (
                     ROUND((width_mm / 1000.0) * (height_mm / 1000.0) * quantity * unit_price, 2)
                 ) STORED,               -- итог строки

    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_detail_estimates_order_id      ON order_detail_estimates(order_id);
CREATE INDEX idx_order_detail_estimates_service_type  ON order_detail_estimates(order_id, service_type);

-- ================================================================
-- НАСТРОЙКИ СМЕТЫ ПО ЗАКАЗУ
-- Заголовок, подзаголовок, примечания для каждого типа услуги
-- ================================================================

CREATE TABLE order_estimate_settings (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    service_type VARCHAR(30)  NOT NULL
                     CHECK (service_type IN ('cnc', 'painting', 'soft', 'cutting')),
    section_title VARCHAR(200),
    section_subtitle VARCHAR(200),  -- "От эскиза до идеального цвета"
    deadline      VARCHAR(100),     -- "10-руз", "2-руз"
    delivery_date DATE,             -- Дата сдачи
    notes         TEXT,             -- Цвет, примечания
    UNIQUE (order_id, service_type)
);

CREATE INDEX idx_order_estimate_settings_order_id ON order_estimate_settings(order_id);
