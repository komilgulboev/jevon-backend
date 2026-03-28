-- ══════════════════════════════════════════════════════════════
--  000021 — Мессенджеры поставщиков
-- ══════════════════════════════════════════════════════════════

ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS whatsapp VARCHAR(50),
    ADD COLUMN IF NOT EXISTS telegram  VARCHAR(100);
