-- Добавляем序列 для номеров проектов
CREATE SEQUENCE IF NOT EXISTS project_number_seq START 1;

-- Добавляем поле project_number
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS project_number INT UNIQUE DEFAULT nextval('project_number_seq');

-- Заполняем номера для существующих проектов
UPDATE projects
SET project_number = nextval('project_number_seq')
WHERE project_number IS NULL;

-- Делаем NOT NULL после заполнения
ALTER TABLE projects
    ALTER COLUMN project_number SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_projects_number ON projects(project_number);
