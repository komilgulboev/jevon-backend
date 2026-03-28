-- ================================================================
-- ПРОИЗВОДСТВЕННЫЙ КОНВЕЙЕР ПРОЕКТА
-- ================================================================

-- ── Расширяем таблицу projects ───────────────────────────
-- Добавляем поля для приёма заказа (менеджер)
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS current_stage  VARCHAR(50)  DEFAULT 'intake'
        CHECK (current_stage IN (
            'intake',       -- 1. Приём заказа
            'design',       -- 2. Дизайн
            'cutting',      -- 3. Раскрой
            'production',   -- 4. Производство
            'warehouse',    -- 5. Склад
            'delivery',     -- 6. Доставка
            'assembly',     -- 7. Сборка
            'handover',     -- 8. Сдача
            'done',         -- Завершён
            'cancelled'     -- Отменён
        )),
    ADD COLUMN IF NOT EXISTS manager_id     UUID         REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS address        TEXT,
    ADD COLUMN IF NOT EXISTS notes          TEXT;

CREATE INDEX IF NOT EXISTS idx_projects_current_stage ON projects(current_stage);
CREATE INDEX IF NOT EXISTS idx_projects_manager_id    ON projects(manager_id);

-- ── Этапы проекта ────────────────────────────────────────
-- Каждый проект проходит этапы последовательно
-- Здесь хранится история и текущее состояние каждого этапа
CREATE TABLE project_stages (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stage       VARCHAR(50) NOT NULL
                    CHECK (stage IN (
                        'intake','design','cutting','production',
                        'warehouse','delivery','assembly','handover'
                    )),
    status      VARCHAR(30) DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','done','skipped')),
    assigned_to UUID        REFERENCES users(id) ON DELETE SET NULL,
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    notes       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (project_id, stage)
);

CREATE INDEX idx_project_stages_project_id ON project_stages(project_id);
CREATE INDEX idx_project_stages_assigned_to ON project_stages(assigned_to);

CREATE TRIGGER trg_project_stages_updated_at
    BEFORE UPDATE ON project_stages
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Операции этапа ───────────────────────────────────────
-- Мастер выбирает проект → выбирает операцию из каталога → фиксирует работу
CREATE TABLE stage_operations (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID        NOT NULL REFERENCES projects(id)          ON DELETE CASCADE,
    stage_id        UUID        NOT NULL REFERENCES project_stages(id)    ON DELETE CASCADE,
    catalog_id      INT         REFERENCES operation_catalog(id)          ON DELETE SET NULL,
    custom_name     VARCHAR(150),           -- если операции нет в каталоге
    assigned_to     UUID        REFERENCES users(id) ON DELETE SET NULL,
    status          VARCHAR(30) DEFAULT 'todo'
                        CHECK (status IN ('todo','in_progress','done')),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stage_operations_project_id  ON stage_operations(project_id);
CREATE INDEX idx_stage_operations_stage_id    ON stage_operations(stage_id);
CREATE INDEX idx_stage_operations_assigned_to ON stage_operations(assigned_to);

CREATE TRIGGER trg_stage_operations_updated_at
    BEFORE UPDATE ON stage_operations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Материалы операции ───────────────────────────────────
-- На каждой операции можно добавить материалы с ценой и поставщиком
CREATE TABLE operation_materials (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation_id    UUID          NOT NULL REFERENCES stage_operations(id) ON DELETE CASCADE,
    project_id      UUID          NOT NULL REFERENCES projects(id)         ON DELETE CASCADE,
    name            VARCHAR(200)  NOT NULL,         -- название материала
    quantity        NUMERIC(10,3) NOT NULL DEFAULT 1,
    unit            VARCHAR(30)   DEFAULT 'шт'
                        CHECK (unit IN ('шт','м','м²','м³','кг','л','упак')),
    unit_price      NUMERIC(12,2) DEFAULT 0,        -- цена за единицу
    total_price     NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    supplier        VARCHAR(200),                   -- откуда покупали
    supplier_phone  VARCHAR(30),
    purchased_at    DATE,
    notes           TEXT,
    created_by      UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX idx_operation_materials_operation_id ON operation_materials(operation_id);
CREATE INDEX idx_operation_materials_project_id   ON operation_materials(project_id);

-- ── Файлы этапа ──────────────────────────────────────────
-- Дизайнер, раскройщик и другие прикрепляют файлы к этапу
CREATE TABLE stage_files (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID        NOT NULL REFERENCES projects(id)       ON DELETE CASCADE,
    stage_id    UUID        NOT NULL REFERENCES project_stages(id) ON DELETE CASCADE,
    file_name   VARCHAR(255) NOT NULL,
    file_url    TEXT         NOT NULL,      -- URL в хранилище (S3, MinIO и т.д.)
    file_type   VARCHAR(50),               -- image/jpeg, application/pdf и т.д.
    file_size   BIGINT,                    -- размер в байтах
    uploaded_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stage_files_project_id ON stage_files(project_id);
CREATE INDEX idx_stage_files_stage_id   ON stage_files(stage_id);

-- ── История изменений статуса проекта ───────────────────
CREATE TABLE project_history (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    from_stage  VARCHAR(50),
    to_stage    VARCHAR(50),
    changed_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    comment     TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_project_history_project_id ON project_history(project_id);
