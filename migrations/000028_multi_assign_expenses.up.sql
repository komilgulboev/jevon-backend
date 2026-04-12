-- ══════════════════════════════════════════════════════════════
--  000028 — Мультиназначение этапов, расходы цеха, табель
-- ══════════════════════════════════════════════════════════════

-- ── 1. Мультиназначение на этапы проектов ─────────────────────
CREATE TABLE IF NOT EXISTS project_stage_assignees (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id   UUID NOT NULL REFERENCES project_stages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    UNIQUE (stage_id, user_id)
);

-- ── 2. Мультиназначение на этапы заказов ──────────────────────
CREATE TABLE IF NOT EXISTS order_stage_assignees (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id   UUID NOT NULL REFERENCES order_stages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    UNIQUE (stage_id, user_id)
);

-- ── 3. Расходы цеха (общие, не привязанные к заказу) ──────────
CREATE TABLE IF NOT EXISTS workshop_expenses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    category     VARCHAR(100) NOT NULL,
    description  TEXT,
    amount       NUMERIC(12,2) NOT NULL,
    method       VARCHAR(50) NOT NULL DEFAULT 'cash'
        CHECK (method IN ('cash','card','transfer','other')),
    created_by   UUID REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для фильтрации по дате и категории
CREATE INDEX IF NOT EXISTS idx_workshop_expenses_date     ON workshop_expenses(expense_date DESC);
CREATE INDEX IF NOT EXISTS idx_workshop_expenses_category ON workshop_expenses(category);

-- ── 4. Табель — записи отработанного времени ──────────────────
CREATE TABLE IF NOT EXISTS timesheets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_date   DATE NOT NULL,
    hours       NUMERIC(5,2) NOT NULL DEFAULT 8,
    -- Привязка к источнику (необязательно)
    source_type VARCHAR(50),  -- 'task', 'order_stage', 'project_stage', 'manual'
    source_id   UUID,
    notes       TEXT,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, work_date, source_type, source_id)
);

CREATE INDEX IF NOT EXISTS idx_timesheets_user_date ON timesheets(user_id, work_date DESC);
CREATE INDEX IF NOT EXISTS idx_timesheets_date      ON timesheets(work_date DESC);
