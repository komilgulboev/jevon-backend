DROP FUNCTION IF EXISTS distribute_client_payment(UUID, NUMERIC, VARCHAR, DATE, TEXT, UUID);
DROP VIEW  IF EXISTS client_debt;
DROP TABLE IF EXISTS client_balance;
ALTER TABLE order_payments
    DROP COLUMN IF EXISTS client_id,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS is_client_payment;
ALTER TABLE order_payments
    ALTER COLUMN order_id SET NOT NULL;
DROP INDEX IF EXISTS idx_order_payments_client;
