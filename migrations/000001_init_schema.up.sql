CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── Roles ─────────────────────────────────────────────────
CREATE TABLE roles (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50)  UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin',      'Полный доступ'),
    ('supervisor', 'Управление проектами и командой'),
    ('master',     'Ведение своих проектов'),
    ('assistant',  'Просмотр и помощь');

-- ── Users ─────────────────────────────────────────────────
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id       INT          NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    full_name     VARCHAR(150) NOT NULL,
    email         VARCHAR(150) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone         VARCHAR(30),
    is_active     BOOLEAN      DEFAULT TRUE,
    avatar_url    TEXT,
    created_at    TIMESTAMPTZ  DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_users_email   ON users(email);
CREATE INDEX idx_users_role_id ON users(role_id);

-- ── Refresh tokens ────────────────────────────────────────
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- ── Projects ──────────────────────────────────────────────
CREATE TABLE projects (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    title        VARCHAR(200) NOT NULL,
    description  TEXT,
    client_name  VARCHAR(150),
    client_phone VARCHAR(30),
    status       VARCHAR(30)  DEFAULT 'new'
                     CHECK (status IN ('new','in_progress','on_hold','done','cancelled')),
    priority     VARCHAR(20)  DEFAULT 'medium'
                     CHECK (priority IN ('low','medium','high')),
    deadline     DATE,
    created_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_projects_status ON projects(status);

-- ── Project members ───────────────────────────────────────
CREATE TABLE project_members (
    project_id  UUID REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id)    ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

-- ── Tasks ─────────────────────────────────────────────────
CREATE TABLE tasks (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assigned_to UUID        REFERENCES users(id) ON DELETE SET NULL,
    title       VARCHAR(300) NOT NULL,
    description TEXT,
    status      VARCHAR(30)  DEFAULT 'todo'
                    CHECK (status IN ('todo','in_progress','review','done')),
    priority    VARCHAR(20)  DEFAULT 'medium'
                    CHECK (priority IN ('low','medium','high')),
    due_date    DATE,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_tasks_project_id  ON tasks(project_id);
CREATE INDEX idx_tasks_assigned_to ON tasks(assigned_to);
CREATE INDEX idx_tasks_status      ON tasks(status);

-- ── Auto-update updated_at ────────────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_projects_updated_at
    BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION set_updated_at();
