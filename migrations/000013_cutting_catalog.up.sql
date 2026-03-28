-- ================================================================
-- КАТАЛОГ УСЛУГ РАСПИЛА
-- Группы: sawing, edging, drilling, milling, other
-- ================================================================

CREATE TABLE cutting_service_catalog (
    id           SERIAL       PRIMARY KEY,
    group_name   VARCHAR(50)  NOT NULL
                     CHECK (group_name IN (
                         'sawing',    -- Распил
                         'edging',    -- Кромкование
                         'drilling',  -- Присадка
                         'milling',   -- Фрезеровка
                         'gluing',    -- Склейка
                         'packing',   -- Упаковка
                         'design',    -- Чертёж
                         'other'      -- Другое
                     )),
    name         VARCHAR(200) NOT NULL,   -- Распил ДСП 16мм
    unit         VARCHAR(30)  DEFAULT 'шт'
                     CHECK (unit IN ('шт','м','м²','лист','пара')),
    unit_spec    VARCHAR(50),             -- спецификация: 1x2, 1x3, 0.8x19 и т.д.
    price        NUMERIC(10,2) DEFAULT 0, -- цена по умолчанию
    is_active    BOOLEAN       DEFAULT TRUE,
    sort_order   INT           DEFAULT 0,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_cutting_service_catalog_group ON cutting_service_catalog(group_name);

-- Заполняем базовым каталогом
INSERT INTO cutting_service_catalog (group_name, name, unit, unit_spec, price, sort_order) VALUES
    -- Чертёж
    ('design',   'Чертёж в Базисе',           'м²',   NULL,   0,    1),
    -- Распил
    ('sawing',   'Распил ДСП 16мм',            'лист', NULL,   35,   10),
    ('sawing',   'Распил ДСП 8мм',             'лист', NULL,   35,   11),
    ('sawing',   'Распил ДСП 6мм',             'лист', NULL,   35,   12),
    ('sawing',   'Распил МДФ 16мм',            'лист', NULL,   40,   13),
    ('sawing',   'Распил МДФ 6мм',             'лист', NULL,   40,   14),
    ('sawing',   'Распил ХДФ',                 'лист', NULL,   15,   15),
    -- Кромкование
    ('edging',   'Кромкование ПВХ 0.8×19',     'м',    '0.8×19', 2,  20),
    ('edging',   'Кромкование ПВХ 0.8×35',     'м',    '0.8×35', 4,  21),
    ('edging',   'Кромкование ПВХ 2×19',       'м',    '2×19',   5,  22),
    ('edging',   'Кромкование ПВХ 2×35',       'м',    '2×35',   6,  23),
    -- Присадка
    ('drilling', 'Присадка под Евровинт',       'шт',   '1×2',   0.8, 30),
    ('drilling', 'Присадка под Эксцентрики',    'шт',   '1×3',   0.8, 31),
    ('drilling', 'Присадка под Шканты',         'шт',   '1×2',   0.8, 32),
    ('drilling', 'Присадка под Петли',          'шт',   '1×1',   1.2, 33),
    ('drilling', 'Метки под Шурупы',            'шт',   NULL,    0.5, 34),
    -- Фрезеровка
    ('milling',  'Фрезеровка ровных деталей',   'м',    NULL,    5,   40),
    ('milling',  'Фрезеровка не ровных деталей','м',    NULL,    5,   41),
    -- Склейка
    ('gluing',   'Склейка деталей',             'шт',   NULL,    5,   50),
    -- Упаковка
    ('packing',  'Упаковка стрейч-плёнкой',    'шт',   NULL,    3,   60);

-- ================================================================
-- СМЕТА УСЛУГ ЗАКАЗА
-- Строки сметы привязаны к заказу
-- ================================================================

CREATE TABLE order_estimate_services (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    catalog_id   INT           REFERENCES cutting_service_catalog(id) ON DELETE SET NULL,
    -- Данные строки (могут отличаться от каталога)
    name         VARCHAR(200)  NOT NULL,
    color        VARCHAR(100),            -- цвет/артикул (Сафед, Диамант, Дерево...)
    article      VARCHAR(100),            -- артикул (вводится текстом)
    quantity     NUMERIC(10,3) DEFAULT 0,
    unit         VARCHAR(30)   DEFAULT 'шт',
    unit_spec    VARCHAR(50),             -- 1x2, 0.8x19 и т.д.
    unit_price   NUMERIC(10,2) DEFAULT 0,
    total_price  NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    sort_order   INT           DEFAULT 0,
    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_estimate_services_order_id ON order_estimate_services(order_id);

-- ================================================================
-- СМЕТА МАТЕРИАЛОВ ЗАКАЗА
-- Отдельная таблица для материалов в смете (Эксцентрики, Шканты...)
-- ================================================================

CREATE TABLE order_estimate_materials (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    name         VARCHAR(200)  NOT NULL,
    quantity     NUMERIC(10,3) DEFAULT 0,
    unit         VARCHAR(30)   DEFAULT 'шт',
    unit_price   NUMERIC(10,2) DEFAULT 0,
    total_price  NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    sort_order   INT           DEFAULT 0,
    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_order_estimate_materials_order_id ON order_estimate_materials(order_id);

-- Фиксированный список цветов (можно расширять)
CREATE TABLE color_catalog (
    id       SERIAL      PRIMARY KEY,
    name     VARCHAR(100) NOT NULL UNIQUE,
    sort_order INT DEFAULT 0
);

INSERT INTO color_catalog (name, sort_order) VALUES
    ('Сафед',       1),
    ('Диамант',     2),
    ('Дерево',      3),
    ('Дуб Стоунд',  4),
    ('Венге',       5),
    ('Белый',       6),
    ('Чёрный',      7),
    ('Орех',        8),
    ('Другой',      99);
