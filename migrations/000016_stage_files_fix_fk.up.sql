-- Убираем FK на project_stages.id чтобы файлы можно было
-- прикреплять к этапам заказов (order_stages)

ALTER TABLE stage_files
  DROP CONSTRAINT IF EXISTS stage_files_project_id_fkey,
  DROP CONSTRAINT IF EXISTS stage_files_stage_id_fkey;
