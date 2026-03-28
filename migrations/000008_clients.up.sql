-- ================================================================
-- КЛИЕНТЫ
-- Единая база клиентов для всех типов заказов
-- ================================================================

CREATE TABLE clients (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    full_name    VARCHAR(200) NOT NULL,
    phone        VARCHAR(30)  NOT NULL,
    phone2       VARCHAR(30),
    company      VARCHAR(200),            -- если юрлицо
    address      TEXT,
    notes        TEXT,
    is_active    BOOLEAN      DEFAULT TRUE,
    created_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_clients_phone    ON clients(phone);
CREATE INDEX idx_clients_fullname ON clients(full_name);

CREATE TRIGGER trg_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
