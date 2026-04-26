-- ========================
-- TENANT & USERS
-- ========================
INSERT INTO tenants (id, name) VALUES (1, 'Default Organization') ON CONFLICT DO NOTHING;

-- Super Admin
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

-- Admin
-- credentials: admin@system.com / admin123
INSERT INTO users (id, tenant_id, email, password, role)
VALUES (
    2,
    1,
    'admin@system.com',
    '$2a$10$LDvRpRF7obN7hBBdIVfLZ.OVn69W/MPXIgosX0j/pIEokTqWtCAzC',
    'admin'
)
ON CONFLICT (id) DO NOTHING;