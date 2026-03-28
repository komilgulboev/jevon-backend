-- Восстанавливаем FK (только если нужен откат)
ALTER TABLE stage_files
  ADD CONSTRAINT stage_files_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  ADD CONSTRAINT stage_files_stage_id_fkey
    FOREIGN KEY (stage_id) REFERENCES project_stages(id) ON DELETE CASCADE;
