DROP TRIGGER IF EXISTS trg_tasks_updated_at    ON tasks;
DROP TRIGGER IF EXISTS trg_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS trg_users_updated_at    ON users;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;

DROP EXTENSION IF EXISTS "uuid-ossp";
