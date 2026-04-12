DROP INDEX IF EXISTS idx_users_phone_unique;
ALTER TABLE users
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS whatsapp,
    DROP COLUMN IF EXISTS telegram,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS salary,
    DROP COLUMN IF EXISTS hourly_rate,
    DROP COLUMN IF EXISTS contract_type;
