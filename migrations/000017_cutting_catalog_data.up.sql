-- Добавляем уникальный индекс если его нет
CREATE UNIQUE INDEX IF NOT EXISTS idx_cutting_service_catalog_name
  ON cutting_service_catalog(name);

-- Деактивируем старые записи
UPDATE cutting_service_catalog SET is_active = false;

-- Вставляем/обновляем каталог
INSERT INTO cutting_service_catalog (group_name, name, unit, unit_spec, price, sort_order, is_active) VALUES
('design',   'Чертёж Базис Мебельщик',                    'м²',   'за 1м² Листового материала', 15.00,  1,  true),
('sawing',   'Распил ДСП 5,03м²',                         'лист',  NULL,   35.00, 10, true),
('sawing',   'Распил МДФ 3,41м²',                         'лист',  NULL,   35.00, 11, true),
('sawing',   'Распил ДСП 5,79м²',                         'лист',  NULL,   40.00, 12, true),
('sawing',   'Распил МДФ 5,79м²',                         'лист',  NULL,   40.00, 13, true),
('sawing',   'Распил ХДФ',                                'лист',  NULL,   15.00, 14, true),
('sawing',   'Распил столешница 3000×600',                'лист',  NULL,   20.00, 15, true),
('sawing',   'Распил столешница 4000×600',                'лист',  NULL,   30.00, 16, true),
('sawing',   'Распил столешница 3000/4000×900',           'лист',  NULL,   40.00, 17, true),
('edging',   'Кромкование 0,8×19',                        'м',    '0.8×19',    2.00, 20, true),
('edging',   'Кромкование 0,8×22',                        'м',    '0.8×22',    2.50, 21, true),
('edging',   'Кромкование 0,8×35',                        'м',    '0.8×35',    4.00, 22, true),
('edging',   'Кромкование овальных деталей 0,8×19/22',    'м',    '0.8×19/22', 7.00, 23, true),
('edging',   'Кромкование овальных деталей 0,8×35',       'м',    '0.8×35',    9.00, 24, true),
('drilling', 'Присадка под петли',                        'шт',    NULL,    1.20, 30, true),
('drilling', 'Присадка под евровинты',                    'шт',    '1×2',   0.80, 31, true),
('drilling', 'Присадка под эксцентрики',                  'шт',    '1×3',   0.80, 32, true),
('drilling', 'Присадка под шканты',                       'шт',    '1×2',   0.80, 33, true),
('drilling', 'Зенковка отверстий',                        'шт',    NULL,    0.50, 34, true),
('drilling', 'Присадка под полкадержатели',               'шт',    NULL,    0.50, 35, true),
('drilling', 'Метки под шурупы',                          'шт',    NULL,    0.50, 36, true),
('milling',  'Паз под профиль подсветка',                 'м',    'за 1п.м.', 5.00, 40, true),
('milling',  'Паз под ЛХДФ',                              'м',    'за 1п.м.', 3.00, 41, true),
('milling',  'Фрезеровка овальных деталей',               'шт',   'за 1 угл', 5.00, 42, true),
('milling',  'Фрезеровкаи под Gola профиль',              'шт',   'за 1шт',   5.00, 43, true),
('milling',  'Запил ЛДСП и МДФ под 45 градусов',         'м',    'за 1п.м.', 5.00, 44, true),
('gluing',   'Склейка ровных деталей',                    'м',    'за 1п.м.',  5.00, 50, true),
('gluing',   'Склейка деталей под 45 градусов',           'м',    'за 1п.м.', 10.00, 51, true),
('other',    'Доставка материала ДСП, МДФ до 10л',        'шт',    NULL,   50.00, 60, true),
('packing',  'Упаковка стрейчплёнкой',                    'шт',   'за 1 упаковку', 3.00, 70, true)
ON CONFLICT (name) DO UPDATE SET
  group_name = EXCLUDED.group_name,
  unit       = EXCLUDED.unit,
  unit_spec  = EXCLUDED.unit_spec,
  price      = EXCLUDED.price,
  sort_order = EXCLUDED.sort_order,
  is_active  = EXCLUDED.is_active;