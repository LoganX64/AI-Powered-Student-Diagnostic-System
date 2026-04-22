-- ========================
-- ADMIN USER
-- ========================
-- credentials: admin@system.com / admin123
INSERT INTO users (id, email, password, role)
VALUES (
    1,
    'admin@system.com',
    '$2a$10$LDvRpRF7obN7hBBdIVfLZ.OVn69W/MPXIgosX0j/pIEokTqWtCAzC',
    'admin'
)
ON CONFLICT (id) DO NOTHING;