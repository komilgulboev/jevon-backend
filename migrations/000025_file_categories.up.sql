-- ══════════════════════════════════════════════════════════════
--  000024 — Категории файлов
-- ══════════════════════════════════════════════════════════════

ALTER TABLE stage_files
    ADD COLUMN IF NOT EXISTS category VARCHAR(100) DEFAULT 'other';

-- Индекс для фильтрации по категории
CREATE INDEX IF NOT EXISTS idx_stage_files_category ON stage_files(category);
