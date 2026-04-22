-- Привязка заказа к проекту
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_orders_project_id ON orders(project_id);
