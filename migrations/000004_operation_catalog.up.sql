-- ── Каталог операций ─────────────────────────────────────
-- Справочник работ которые выбирают мастера
CREATE TABLE operation_catalog (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(150) NOT NULL,
    description TEXT,
    category    VARCHAR(50)  NOT NULL
                    CHECK (category IN (
                        'production',  -- производство
                        'assembly',    -- сборка
                        'finishing',   -- отделка
                        'other'
                    )),
    is_active   BOOLEAN      DEFAULT TRUE,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_operation_catalog_category ON operation_catalog(category);

-- Базовые операции
INSERT INTO operation_catalog (name, category, description) VALUES
    -- Производство
    ('Распыл',          'production', 'Распиловка материала по размерам'),
    ('Шкурка',          'production', 'Шлифовка поверхности'),
    ('Покраска',        'production', 'Покраска изделия'),
    ('Лакировка',       'production', 'Покрытие лаком'),
    ('Фрезеровка',      'production', 'Фрезерные работы'),
    ('Кромкование',     'production', 'Наклейка кромки'),
    ('Сверление',       'production', 'Сверление отверстий'),
    ('Фурнитура',       'production', 'Установка фурнитуры'),
    -- Отделка
    ('Грунтовка',       'finishing',  'Нанесение грунтовки'),
    ('Патина',          'finishing',  'Нанесение патины'),
    ('Полировка',       'finishing',  'Полировка поверхности'),
    -- Сборка
    ('Сборка каркаса',  'assembly',   'Сборка основного каркаса'),
    ('Сборка дверей',   'assembly',   'Сборка и навеска дверей'),
    ('Сборка ящиков',   'assembly',   'Сборка выдвижных ящиков'),
    ('Финальная сборка','assembly',   'Полная сборка изделия на объекте');
