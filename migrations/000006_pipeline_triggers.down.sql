DROP TRIGGER IF EXISTS trg_advance_project_stage  ON project_stages;
DROP TRIGGER IF EXISTS trg_create_project_stages  ON projects;
DROP FUNCTION IF EXISTS advance_project_stage();
DROP FUNCTION IF EXISTS create_project_stages();
