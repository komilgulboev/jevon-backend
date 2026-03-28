-- Default admin user
-- Password: Admin@1234
-- Hash generated with bcrypt cost=12
-- ВАЖНО: Смени пароль после первого входа!
INSERT INTO users (role_id, full_name, email, password_hash) VALUES (
    1,
    'Администратор',
    'admin@jevon.uz',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj0oQmqMqQm2'
) ON CONFLICT (email) DO NOTHING;
