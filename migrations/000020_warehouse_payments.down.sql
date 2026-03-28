DROP TRIGGER  IF EXISTS trg_warehouse_payment_insert ON warehouse_payments;
DROP FUNCTION IF EXISTS recalc_receipt_payment();
DROP TABLE    IF EXISTS warehouse_payments;
ALTER TABLE warehouse_receipts
    DROP COLUMN IF EXISTS paid_amount,
    DROP COLUMN IF EXISTS payment_status,
    DROP COLUMN IF EXISTS payment_notes;
