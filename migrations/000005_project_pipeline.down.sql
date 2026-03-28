DROP TABLE IF EXISTS project_history;
DROP TABLE IF EXISTS stage_files;
DROP TABLE IF EXISTS operation_materials;
DROP TABLE IF EXISTS stage_operations;
DROP TABLE IF EXISTS project_stages;

ALTER TABLE projects
    DROP COLUMN IF EXISTS current_stage,
    DROP COLUMN IF EXISTS manager_id,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS notes;
