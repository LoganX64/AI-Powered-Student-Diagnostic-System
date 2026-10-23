BEGIN;

-- Platform rows cannot exist under the original tenant_id NOT NULL schema.
DELETE FROM notifications WHERE scope = 'platform';

DROP INDEX IF EXISTS idx_notifications_user_event_key;
DROP INDEX IF EXISTS idx_notifications_user_unseen;
DROP INDEX IF EXISTS idx_notifications_user_created;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_scope_tenant_check,
    DROP CONSTRAINT IF EXISTS notifications_scope_check,
    DROP COLUMN IF EXISTS event_key,
    DROP COLUMN IF EXISTS seen_at,
    DROP COLUMN IF EXISTS scope,
    ALTER COLUMN tenant_id SET NOT NULL,
    ALTER COLUMN user_id DROP NOT NULL;

COMMIT;
