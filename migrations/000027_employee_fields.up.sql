-- ══════════════════════════════════════════════════════════════
--  000027 — Поля сотрудников
-- ══════════════════════════════════════════════════════════════

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_name      VARCHAR(150),
    ADD COLUMN IF NOT EXISTS whatsapp       VARCHAR(50),
    ADD COLUMN IF NOT EXISTS telegram       VARCHAR(100),
    ADD COLUMN IF NOT EXISTS address        TEXT,
    ADD COLUMN IF NOT EXISTS salary         NUMERIC(12,2),
    ADD COLUMN IF NOT EXISTS hourly_rate    NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS contract_type  VARCHAR(50)
        CHECK (contract_type IN ('employment', 'gph', 'ip', 'none'));

-- Уникальный индекс на телефон для логина
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique
    ON users(phone) WHERE phone IS NOT NULL AND phone != '';

COMMENT ON COLUMN users.contract_type IS
    'employment=Трудовой договор, gph=ГПХ/подряд, ip=ИП, none=Без договора';
COMMENT ON COLUMN users.salary IS 'Фиксированная ставка в месяц (сом)';
COMMENT ON COLUMN users.hourly_rate IS 'Почасовая ставка (сом/час)';
