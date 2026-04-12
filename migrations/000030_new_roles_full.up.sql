-- ══════════════════════════════════════════════════════════════
--  000030 — Новые роли пользователей
-- ══════════════════════════════════════════════════════════════

-- Добавляем колонку display_name если её нет
ALTER TABLE roles ADD COLUMN IF NOT EXISTS display_name VARCHAR(100);

-- Обновляем display_name для существующих ролей
UPDATE roles SET display_name = 'Администратор'  WHERE name = 'admin';
UPDATE roles SET display_name = 'Супервайзер'    WHERE name = 'supervisor';
UPDATE roles SET display_name = 'Мастер'         WHERE name = 'master';
UPDATE roles SET display_name = 'Ассистент'      WHERE name = 'assistant';
UPDATE roles SET display_name = 'Менеджер'       WHERE name = 'manager';
UPDATE roles SET display_name = 'Дизайнер'       WHERE name = 'designer';
UPDATE roles SET display_name = 'Раскройщик'     WHERE name = 'cutter';
UPDATE roles SET display_name = 'Склад'          WHERE name = 'warehouse';
UPDATE roles SET display_name = 'Водитель'       WHERE name = 'driver';
UPDATE roles SET display_name = 'Сборщик'        WHERE name = 'assembler';

-- Добавляем новые роли
INSERT INTO roles (name, display_name) VALUES
    ('director',             'Директор'),
    ('accountant',           'Бухгалтер'),
    ('assistant_manager',    'Помощник менеджера'),
    ('project_manager',      'Менеджер проектов'),
    ('workshop_manager',     'Управляющий цеха'),
    ('design_lead',          'Руководитель конструкторов'),
    ('cutting_lead',         'Руководитель услуги распил'),
    ('painting_lead',        'Руководитель покраски'),
    ('soft_furniture_lead',  'Руководитель мягкой мебели'),
    ('procurement',          'Снабженец/Закупщик'),
    ('seller',               'Продавец'),
    ('constructor',          'Конструктор'),
    ('carpenter',            'Столяр'),
    ('installer',            'Сборщик-установщик'),
    ('cnc_operator',         'Оператор ЧПУ станка'),
    ('soft_furniture_master','Мастер мягкой мебели'),
    ('sander',               'Шлифовщик'),
    ('painter_worker',       'Маляр'),
    ('cutter_worker',        'Распиловщик'),
    ('driller',              'Присадчик'),
    ('edger',                'Кромщик'),
    ('cook',                 'Повар')
ON CONFLICT (name) DO UPDATE SET display_name = EXCLUDED.display_name;
