-- Расходы по заказу цеха
CREATE TABLE order_expenses (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    name         VARCHAR(300) NOT NULL,
    amount       NUMERIC(12,2) NOT NULL DEFAULT 0,
    expense_date DATE,
    description  TEXT,
    method       VARCHAR(30)  DEFAULT 'cash'
                     CHECK (method IN ('cash','card','transfer','other')),
    created_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_order_expenses_order_id ON order_expenses(order_id);
