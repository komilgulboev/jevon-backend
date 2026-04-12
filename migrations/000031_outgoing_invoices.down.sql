DROP SEQUENCE IF EXISTS outgoing_invoice_ext_seq;
DROP TABLE IF EXISTS outgoing_invoice_items;
DROP TABLE IF EXISTS outgoing_invoices;
ALTER TABLE warehouse_items DROP COLUMN IF EXISTS sale_price;
