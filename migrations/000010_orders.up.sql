-- ================================================================
-- ЗАКАЗЫ — главная таблица
-- Единая таблица для всех 6 типов заказов
-- ================================================================

CREATE TABLE orders (
    id             UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number   INT          UNIQUE NOT NULL DEFAULT nextval('project_number_seq'),
    order_type     VARCHAR(30)  NOT NULL
                       CHECK (order_type IN (
                           'workshop',       -- заказ цеха (мебель под заказ)
                           'cutting',        -- услуга распила
                           'painting',       -- услуга покраски
                           'cnc',            -- услуга ЧПУ
                           'soft_fabric',    -- мягкая мебель обивка
                           'soft_furniture'  -- производство мягкой мебели
                       )),

    -- Клиент
    client_id      UUID         REFERENCES clients(id) ON DELETE SET NULL,
    client_name    VARCHAR(200),  -- если клиент не в базе
    client_phone   VARCHAR(30),

    -- Основные поля
    title          VARCHAR(300) NOT NULL,
    description    TEXT,
    address        TEXT,                    -- адрес объекта/доставки

    -- Статус
    current_stage  VARCHAR(50)  DEFAULT 'intake',
    status         VARCHAR(30)  DEFAULT 'new'
                       CHECK (status IN ('new','in_progress','on_hold','done','cancelled')),
    priority       VARCHAR(20)  DEFAULT 'medium'
                       CHECK (priority IN ('low','medium','high','urgent')),

    -- Даты
    deadline       DATE,
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ,

    -- Финансы
    estimated_cost NUMERIC(14,2) DEFAULT 0,  -- предварительная стоимость
    final_cost     NUMERIC(14,2) DEFAULT 0,  -- итоговая стоимость
    paid_amount    NUMERIC(14,2) DEFAULT 0,  -- оплачено
    payment_status VARCHAR(20)   DEFAULT 'unpaid'
                       CHECK (payment_status IN ('unpaid','partial','paid','refund')),

    -- Транспорт (для доставки)
    driver_id      UUID         REFERENCES users(id) ON DELETE SET NULL,
    vehicle        VARCHAR(100),
    distance_km    NUMERIC(8,2),
    fuel_expense   NUMERIC(10,2),

    -- Служебные
    manager_id     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_orders_order_type    ON orders(order_type);
CREATE INDEX idx_orders_status        ON orders(status);
CREATE INDEX idx_orders_client_id     ON orders(client_id);
CREATE INDEX idx_orders_manager_id    ON orders(manager_id);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_deadline      ON orders(deadline);

CREATE TRIGGER trg_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ================================================================
-- ЭТАПЫ ЗАКАЗА
-- Этапы определяются типом заказа
-- ================================================================

CREATE TABLE order_stages (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage        VARCHAR(50) NOT NULL,
    stage_order  INT         NOT NULL,    -- порядковый номер этапа
    status       VARCHAR(30) DEFAULT 'pending'
                     CHECK (status IN ('pending','in_progress','done','skipped')),
    assigned_to  UUID        REFERENCES users(id) ON DELETE SET NULL,
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    notes        TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (order_id, stage)
);

CREATE INDEX idx_order_stages_order_id     ON order_stages(order_id);
CREATE INDEX idx_order_stages_assigned_to  ON order_stages(assigned_to);
CREATE INDEX idx_order_stages_status       ON order_stages(status);

CREATE TRIGGER trg_order_stages_updated_at
    BEFORE UPDATE ON order_stages
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ================================================================
-- ПОЗИЦИИ ЗАКАЗА
-- Детали, изделия — то что производится в рамках заказа
-- ================================================================

CREATE TABLE order_items (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    name         VARCHAR(300)  NOT NULL,   -- название детали/изделия
    quantity     NUMERIC(10,3) DEFAULT 1,
    unit         VARCHAR(30)   DEFAULT 'шт',
    width        NUMERIC(8,2),             -- ширина мм
    height       NUMERIC(8,2),             -- высота мм
    depth        NUMERIC(8,2),             -- глубина мм
    area_m2      NUMERIC(10,4),            -- площадь м² (авто или вручную)
    material_id  UUID          REFERENCES materials_catalog(id) ON DELETE SET NULL,
    material_name VARCHAR(200),            -- если не из каталога
    unit_price   NUMERIC(12,2) DEFAULT 0,
    total_price  NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    notes        TEXT,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- ================================================================
-- РАСЧЁТЫ ЗАКАЗА
-- Специфические расчёты по типу (м², кг краски и т.д.)
-- ================================================================

CREATE TABLE order_calculations (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage_id        UUID          REFERENCES order_stages(id) ON DELETE SET NULL,

    -- Площадь (для покраски, ЧПУ, обивки)
    total_area_m2   NUMERIC(10,4) DEFAULT 0,

    -- Покраска
    paint_kg        NUMERIC(8,3)  DEFAULT 0,   -- краска (м² × 0.35)
    primer_kg       NUMERIC(8,3)  DEFAULT 0,   -- грунт (м² × 0.4)
    paint_type      VARCHAR(50),               -- тип краски

    -- ЧПУ
    cnc_type        VARCHAR(30)
                        CHECK (cnc_type IN ('simple_cut','2d','3d')),
    cnc_time_hours  NUMERIC(6,2),

    -- Распил
    sheet_count     INT           DEFAULT 0,   -- кол-во листов
    cut_length_m    NUMERIC(10,2) DEFAULT 0,   -- метраж распила

    -- Материал клиента (для распила и покраски)
    client_material BOOLEAN       DEFAULT FALSE,
    client_material_desc TEXT,

    -- Итог расчёта
    calculated_cost NUMERIC(12,2) DEFAULT 0,
    calculated_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    calculated_at   TIMESTAMPTZ   DEFAULT NOW(),
    notes           TEXT
);

CREATE INDEX idx_order_calculations_order_id ON order_calculations(order_id);

-- ================================================================
-- ОПЛАТЫ ПО ЗАКАЗУ
-- История платежей
-- ================================================================

CREATE TABLE order_payments (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount       NUMERIC(12,2) NOT NULL,
    payment_type VARCHAR(30)   DEFAULT 'cash'
                     CHECK (payment_type IN ('cash','card','transfer','other')),
    paid_at      TIMESTAMPTZ   DEFAULT NOW(),
    notes        TEXT,
    received_by  UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_payments_order_id ON order_payments(order_id);

-- ================================================================
-- КОММЕНТАРИИ К ЗАКАЗУ
-- Мастера и менеджеры добавляют комментарии
-- ================================================================

CREATE TABLE order_comments (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id   UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage_id   UUID        REFERENCES order_stages(id) ON DELETE SET NULL,
    text       TEXT        NOT NULL,
    author_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_comments_order_id ON order_comments(order_id);

-- ================================================================
-- ФАЙЛЫ ЗАКАЗА
-- Фото, чертежи, акты
-- ================================================================

CREATE TABLE order_files (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage_id     UUID        REFERENCES order_stages(id) ON DELETE SET NULL,
    file_name    VARCHAR(255) NOT NULL,
    file_url     TEXT         NOT NULL,
    file_type    VARCHAR(50),
    file_size    BIGINT,
    file_category VARCHAR(30) DEFAULT 'other'
                     CHECK (file_category IN (
                         'photo',    -- фото объекта/работы
                         'design',   -- дизайн проекта
                         'drawing',  -- чертёж
                         'act',      -- акт выполненных работ
                         'receipt',  -- квитанция оплаты
                         'other'
                     )),
    uploaded_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_files_order_id ON order_files(order_id);

-- ================================================================
-- ЗАРПЛАТА МАСТЕРОВ
-- Сдельная оплата по выполненным этапам
-- ================================================================

CREATE TABLE master_wages (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id     UUID          REFERENCES orders(id) ON DELETE SET NULL,
    stage_id     UUID          REFERENCES order_stages(id) ON DELETE SET NULL,
    amount       NUMERIC(12,2) NOT NULL,
    wage_type    VARCHAR(30)   DEFAULT 'piece'
                     CHECK (wage_type IN ('piece','hourly','fixed')),
    description  TEXT,
    period_start DATE,
    period_end   DATE,
    is_paid      BOOLEAN       DEFAULT FALSE,
    paid_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_master_wages_user_id  ON master_wages(user_id);
CREATE INDEX idx_master_wages_order_id ON master_wages(order_id);
CREATE INDEX idx_master_wages_is_paid  ON master_wages(is_paid);
