-- ================================================================
-- ПРАЙСЛИСТ
-- Цены на услуги по типам заказов
-- ================================================================

CREATE TABLE price_list (
    id           SERIAL       PRIMARY KEY,
    order_type   VARCHAR(30)  NOT NULL
                     CHECK (order_type IN (
                         'workshop','cutting','painting','cnc',
                         'soft_fabric','soft_furniture'
                     )),
    service_type VARCHAR(100) NOT NULL,   -- тип услуги внутри заказа
    name         VARCHAR(200) NOT NULL,   -- название позиции
    unit         VARCHAR(30)  DEFAULT 'шт'
                     CHECK (unit IN ('шт','м','м²','м³','кг','л','час','лист')),
    price        NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency     VARCHAR(10)   DEFAULT 'сом',
    is_active    BOOLEAN       DEFAULT TRUE,
    notes        TEXT,
    created_at   TIMESTAMPTZ   DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   DEFAULT NOW()
);

-- Примеры: тип → service_type → name → price
-- cutting → sawing → Распил листа ЛДСП → 5000 сом
-- cutting → edging → Кромка ПВХ 1мм → 800 сом/м
-- painting → painting → Покраска эмаль → 25000 сом/м²
-- cnc → simple_cut → Простая резка → 15000 сом/м²
-- cnc → 2d → 2D фрезеровка → 35000 сом/м²
-- cnc → 3d → 3D фрезеровка → 70000 сом/м²
-- soft_furniture → sofa_2 → Диван 2-местный → 2500000 сом

INSERT INTO price_list (order_type, service_type, name, unit, price) VALUES
    -- Распил
    ('cutting', 'sawing',   'Распил листа ЛДСП 16мм',     'лист', 5000),
    ('cutting', 'sawing',   'Распил листа МДФ 16мм',       'лист', 6000),
    ('cutting', 'edging',   'Кромка ПВХ 0.4мм',            'м',    500),
    ('cutting', 'edging',   'Кромка ПВХ 1мм',              'м',    800),
    ('cutting', 'drilling', 'Присадка отверстий',           'шт',   200),
    -- Покраска
    ('painting', 'sanding',  'Шлифовка',                   'м²',   8000),
    ('painting', 'priming',  'Грунтовка',                  'м²',   10000),
    ('painting', 'painting', 'Покраска эмаль',             'м²',   25000),
    ('painting', 'painting', 'Покраска RAL',                'м²',   30000),
    -- ЧПУ
    ('cnc', 'simple_cut', 'Простая резка',                  'м²',   15000),
    ('cnc', '2d',         '2D фрезеровка',                  'м²',   35000),
    ('cnc', '3d',         '3D фрезеровка',                  'м²',   70000),
    -- Мягкая мебель обивка
    ('soft_fabric', 'upholstery', 'Обивка кровати',         'м²',   45000),
    ('soft_fabric', 'upholstery', 'Стеновые панели',        'м²',   40000),
    -- Мягкая мебель производство
    ('soft_furniture', 'sofa',    'Диван 2-местный',        'шт',   2500000),
    ('soft_furniture', 'sofa',    'Диван 3-местный',        'шт',   3500000),
    ('soft_furniture', 'bed',     'Кровать 160×200',        'шт',   3000000),
    ('soft_furniture', 'bed',     'Кровать 180×200',        'шт',   3500000);

CREATE INDEX idx_price_list_order_type ON price_list(order_type, service_type);

CREATE TRIGGER trg_price_list_updated_at
    BEFORE UPDATE ON price_list
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ================================================================
-- КАТАЛОГ МАТЕРИАЛОВ
-- Материалы которые используются в производстве
-- ================================================================

CREATE TABLE materials_catalog (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name         VARCHAR(200) NOT NULL,
    category     VARCHAR(50)  NOT NULL
                     CHECK (category IN (
                         'board',     -- плита (ЛДСП, МДФ, фанера)
                         'paint',     -- краска
                         'primer',    -- грунтовка
                         'edging',    -- кромка
                         'hardware',  -- фурнитура
                         'fabric',    -- ткань для обивки
                         'foam',      -- поролон
                         'other'
                     )),
    unit         VARCHAR(30)  DEFAULT 'шт'
                     CHECK (unit IN ('шт','м','м²','м³','кг','л','рулон','лист')),
    price        NUMERIC(12,2) DEFAULT 0,  -- закупочная цена
    stock_qty    NUMERIC(10,3) DEFAULT 0,  -- остаток на складе
    min_stock    NUMERIC(10,3) DEFAULT 0,  -- минимальный остаток
    supplier     VARCHAR(200),
    notes        TEXT,
    is_active    BOOLEAN       DEFAULT TRUE,
    created_at   TIMESTAMPTZ   DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_materials_catalog_category ON materials_catalog(category);

INSERT INTO materials_catalog (name, category, unit, price) VALUES
    ('ЛДСП 16мм белый',      'board',   'лист', 350000),
    ('МДФ 16мм',             'board',   'лист', 450000),
    ('Фанера 18мм',          'board',   'лист', 280000),
    ('Краска эмаль белая',   'paint',   'кг',   35000),
    ('Краска RAL',           'paint',   'кг',   55000),
    ('Грунтовка акриловая',  'primer',  'кг',   20000),
    ('Кромка ПВХ 0.4мм',    'edging',  'м',    300),
    ('Кромка ПВХ 1мм',      'edging',  'м',    500),
    ('Ткань велюр',          'fabric',  'м²',   120000),
    ('Ткань экокожа',        'fabric',  'м²',   180000),
    ('Поролон 50мм',         'foam',    'м²',   85000);

CREATE TRIGGER trg_materials_catalog_updated_at
    BEFORE UPDATE ON materials_catalog
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
