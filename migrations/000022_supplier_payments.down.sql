DROP VIEW  IF EXISTS supplier_debt;
ALTER TABLE warehouse_payments
    DROP COLUMN IF EXISTS supplier_id,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS is_supplier_payment;
ALTER TABLE warehouse_payments
    ALTER COLUMN receipt_id SET NOT NULL;
DROP INDEX IF EXISTS idx_warehouse_payments_supplier;
