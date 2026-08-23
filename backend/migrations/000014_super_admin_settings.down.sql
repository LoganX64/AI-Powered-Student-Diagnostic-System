BEGIN;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS tenant_settings;
DROP TABLE IF EXISTS user_profiles;
ALTER TABLE tenants DROP COLUMN IF EXISTS plan_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspended_at;
COMMIT;
