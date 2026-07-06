CREATE TABLE IF NOT EXISTS user_resource_overrides (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT NOT NULL,
    effect VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_resource_overrides_unique UNIQUE (user_id, resource_type, resource_id, effect)
);

CREATE INDEX IF NOT EXISTS idx_user_resource_overrides_user_resource_effect
    ON user_resource_overrides (user_id, resource_type, effect);
