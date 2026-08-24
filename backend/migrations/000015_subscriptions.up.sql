BEGIN;

CREATE TABLE IF NOT EXISTS subscription_plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    student_limit INT NOT NULL DEFAULT 0,
    coach_limit INT NOT NULL DEFAULT 0,
    storage_limit_bytes BIGINT NOT NULL DEFAULT 0,
    test_limit INT NOT NULL DEFAULT 0,
    sqi_access BOOLEAN NOT NULL DEFAULT FALSE,
    video_proctoring_included BOOLEAN NOT NULL DEFAULT FALSE,
    video_proctoring_limit INT NOT NULL DEFAULT 0,
    -- to avoid silent rounding loss. Convert to Rs only at the display layer.
    video_proctoring_price_per_student BIGINT NOT NULL DEFAULT 0,
    price_monthly BIGINT NOT NULL DEFAULT 0,
    features JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id INT UNIQUE NOT NULL,
    plan_id INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    razorpay_subscription_id VARCHAR(100),
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS storage_usage (
    id SERIAL PRIMARY KEY,
    tenant_id INT UNIQUE NOT NULL,
    used_bytes BIGINT NOT NULL DEFAULT 0,
    last_calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Seed plans
INSERT INTO subscription_plans (name, slug, student_limit, coach_limit, storage_limit_bytes, test_limit, sqi_access, video_proctoring_included, video_proctoring_limit, video_proctoring_price_per_student, price_monthly, features) VALUES
-- Seed prices expressed in PAISE (Rs999 -> 99900, Rs4999 -> 499900)
-- video_proctoring_price_per_student is only meaningful when proctoring is NOT bundled
-- (included plans set it to 0; see Professional/Enterprise where it is included).
('Free', 'free', 50, 3, 1073741824, 10, FALSE, FALSE, 0, 0, 0, '["Basic exam management","Student code login"]'),
('Starter', 'starter', 200, 10, 5368709120, 50, TRUE, FALSE, 0, 0, 99900, '["Basic SQI analytics","Email support","Priority queue"]'),
('Professional', 'professional', 1000, 50, 53687091200, -1, TRUE, TRUE, 100, 0, 499900, '["Advanced SQI analytics","Video proctoring (100 students)","Priority support","Custom branding"]'),
('Enterprise', 'enterprise', -1, -1, 536870912000, -1, TRUE, TRUE, 500, 0, 0, '["Unlimited everything","Dedicated support","Custom integrations","SLA guarantee"]');

-- The plan of record lives in `tenant_subscriptions.plan_id` 
-- do NOT denormalize it onto `tenants`.
--  Give every existing tenant a free-plan subscription instead.
INSERT INTO tenant_subscriptions (tenant_id, plan_id, status)
SELECT id, (SELECT id FROM subscription_plans WHERE slug = 'free'), 'active'
FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;

-- Initialize storage usage for existing tenants
INSERT INTO storage_usage (tenant_id, used_bytes)
SELECT id, 0 FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;

COMMIT;
