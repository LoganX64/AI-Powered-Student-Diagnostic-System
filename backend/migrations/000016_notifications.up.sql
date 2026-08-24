BEGIN;

CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL,
    user_id INT NULL,
    event_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'info',
    read_at TIMESTAMP NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_tenant_created ON notifications (tenant_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id, read_at) WHERE read_at IS NULL;

-- Seed default notification preferences for existing admin/coach users.
-- NOTE: notification_preferences is created in 000014; 
INSERT INTO notification_preferences (user_id, event_type, enabled)
SELECT u.id, e.event_type, TRUE
FROM users u
CROSS JOIN (VALUES
    ('exam_submitted'),
    ('coach_activity'),
    ('system_alert'),
    ('sqi_complete'),
    ('storage_warning'),
    ('student_exam_logout')
) AS e(event_type)
WHERE u.role IN ('admin', 'coach')
ON CONFLICT (user_id, event_type) DO NOTHING;

COMMIT;
