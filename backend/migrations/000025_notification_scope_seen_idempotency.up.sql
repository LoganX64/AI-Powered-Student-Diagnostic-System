BEGIN;

-- Legacy org-wide rows have no owner and cannot hold per-user seen/read state.
DELETE FROM notifications WHERE user_id IS NULL;

-- These event types have no producers anymore.
DELETE FROM notification_preferences
WHERE event_type IN ('system_alert', 'storage_warning');

ALTER TABLE notifications
    ALTER COLUMN tenant_id DROP NOT NULL,
    ALTER COLUMN user_id SET NOT NULL,
    ADD COLUMN scope VARCHAR(20) NOT NULL DEFAULT 'tenant',
    ADD COLUMN seen_at TIMESTAMP NULL,
    ADD COLUMN event_key VARCHAR(200) NULL;

-- Existing read rows are also considered seen.
UPDATE notifications
SET seen_at = read_at
WHERE read_at IS NOT NULL AND seen_at IS NULL;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_scope_check
        CHECK (scope IN ('tenant', 'platform')),
    ADD CONSTRAINT notifications_scope_tenant_check
        CHECK (
            (scope = 'tenant' AND tenant_id IS NOT NULL)
            OR (scope = 'platform' AND tenant_id IS NULL)
        );

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unseen
    ON notifications (user_id)
    WHERE seen_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_user_event_key
    ON notifications (user_id, event_key)
    WHERE event_key IS NOT NULL;

COMMIT;
