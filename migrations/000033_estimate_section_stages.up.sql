-- Этапы внутри секций сметы
CREATE TABLE IF NOT EXISTS estimate_section_stages (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id    UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    service_type VARCHAR(50) NOT NULL,  -- cutting, painting, cnc, soft
    stage       VARCHAR(50) NOT NULL,
    stage_order INT         NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
                            -- pending, in_progress, done, skipped
    assigned_to UUID        REFERENCES users(id) ON DELETE SET NULL,
    notes       TEXT,
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (order_id, service_type, stage)
);

CREATE INDEX IF NOT EXISTS idx_ess_order_id     ON estimate_section_stages(order_id);
CREATE INDEX IF NOT EXISTS idx_ess_service_type ON estimate_section_stages(order_id, service_type);

CREATE TRIGGER trg_ess_updated_at
    BEFORE UPDATE ON estimate_section_stages
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
