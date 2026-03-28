-- Делаем project_id nullable в stage_files
-- чтобы файлы можно было прикреплять к этапам заказов

ALTER TABLE stage_files
  ALTER COLUMN project_id DROP NOT NULL;
