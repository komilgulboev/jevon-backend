DROP TRIGGER IF EXISTS trg_update_payment_status ON order_payments;
DROP FUNCTION IF EXISTS update_order_payment_status();
DROP TABLE IF EXISTS order_history;
DROP TRIGGER IF EXISTS trg_advance_order_stage ON order_stages;
DROP FUNCTION IF EXISTS advance_order_stage();
DROP TRIGGER IF EXISTS trg_create_order_stages ON orders;
DROP FUNCTION IF EXISTS create_order_stages();
