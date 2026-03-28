DROP INDEX IF EXISTS idx_stage_files_category;
ALTER TABLE stage_files DROP COLUMN IF EXISTS category;
