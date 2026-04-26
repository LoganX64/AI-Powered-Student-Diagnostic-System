-- ========================
-- SUPER ADMIN ONLY
-- ========================
-- credentials: super@system.com / admin123
INSERT INTO users (id, tenant_id, email, password, role)
VALUES (
    1,
    NULL,
    'super@system.com',
    '$2a$10$LDvRpRF7obN7hBBdIVfLZ.OVn69W/MPXIgosX0j/pIEokTqWtCAzC',
    'super_admin'
)
ON CONFLICT (id) DO NOTHING;

-- Sync sequences
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
SELECT setval('tenants_id_seq', 1, false);