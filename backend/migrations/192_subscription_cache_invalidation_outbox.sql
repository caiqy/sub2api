-- Durable, transactionally-enqueued subscription authorization cache invalidation.

CREATE TABLE IF NOT EXISTS subscription_cache_invalidation_outbox (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL,
    group_id       BIGINT NOT NULL,
    version        BIGINT NOT NULL CHECK (version > 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivery_stage SMALLINT NOT NULL DEFAULT 0 CHECK (delivery_stage IN (0, 1)),
    attempts       INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error     TEXT,
    claimed_at     TIMESTAMPTZ,
    claimed_by     TEXT
);

CREATE INDEX IF NOT EXISTS idx_subscription_cache_invalidation_outbox_available
    ON subscription_cache_invalidation_outbox (available_at, id)
    WHERE claimed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_cache_invalidation_outbox_lease
    ON subscription_cache_invalidation_outbox (claimed_at)
    WHERE claimed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_cache_invalidation_outbox_subscription
    ON subscription_cache_invalidation_outbox (user_id, group_id, version);

CREATE TABLE IF NOT EXISTS subscription_cache_version_watermarks (
    user_id      BIGINT NOT NULL,
    group_id     BIGINT NOT NULL,
    watermark_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, group_id)
);

ALTER TABLE subscription_cache_version_watermarks
    ADD COLUMN IF NOT EXISTS watermark_at TIMESTAMPTZ;
UPDATE subscription_cache_version_watermarks
SET watermark_at = clock_timestamp()
WHERE watermark_at IS NULL;
ALTER TABLE subscription_cache_version_watermarks
    ALTER COLUMN watermark_at SET NOT NULL;

-- Backfill without regressing any watermark already allocated by a prior rerun.
INSERT INTO subscription_cache_version_watermarks (user_id, group_id, watermark_at)
SELECT user_id, group_id, MAX(updated_at)
FROM user_subscriptions
GROUP BY user_id, group_id
ON CONFLICT (user_id, group_id) DO UPDATE
SET watermark_at = GREATEST(
    subscription_cache_version_watermarks.watermark_at,
    EXCLUDED.watermark_at
);

CREATE OR REPLACE FUNCTION next_subscription_cache_version(
    target_user_id BIGINT,
    target_group_id BIGINT
)
RETURNS TIMESTAMPTZ
LANGUAGE plpgsql
AS $$
DECLARE
    next_version TIMESTAMPTZ;
BEGIN
    INSERT INTO subscription_cache_version_watermarks (user_id, group_id, watermark_at)
    VALUES (target_user_id, target_group_id, clock_timestamp())
    ON CONFLICT (user_id, group_id) DO UPDATE
    SET watermark_at = GREATEST(
        clock_timestamp(),
        subscription_cache_version_watermarks.watermark_at + INTERVAL '1 microsecond'
    )
    RETURNING watermark_at INTO next_version;
    RETURN next_version;
END;
$$;

CREATE OR REPLACE FUNCTION subscription_cache_version_nanos(version_at TIMESTAMPTZ)
RETURNS BIGINT
LANGUAGE SQL
IMMUTABLE
STRICT
AS $$
    SELECT (EXTRACT(EPOCH FROM version_at) * 1000000)::BIGINT * 1000
$$;

CREATE OR REPLACE FUNCTION assign_user_subscription_cache_version()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    deletion_version TIMESTAMPTZ;
BEGIN
    IF TG_OP = 'DELETE' THEN
        deletion_version := next_subscription_cache_version(OLD.user_id, OLD.group_id);
        INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
        VALUES (OLD.user_id, OLD.group_id, subscription_cache_version_nanos(deletion_version));
        RETURN OLD;
    END IF;

    NEW.updated_at := next_subscription_cache_version(NEW.user_id, NEW.group_id);
    IF TG_OP = 'UPDATE' AND (OLD.user_id IS DISTINCT FROM NEW.user_id OR OLD.group_id IS DISTINCT FROM NEW.group_id) THEN
        deletion_version := next_subscription_cache_version(OLD.user_id, OLD.group_id);
        INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
        VALUES (OLD.user_id, OLD.group_id, subscription_cache_version_nanos(deletion_version));
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION enqueue_user_subscription_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    changed BOOLEAN;
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
        VALUES (NEW.user_id, NEW.group_id, subscription_cache_version_nanos(NEW.updated_at));
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    changed := OLD.expires_at IS DISTINCT FROM NEW.expires_at
        OR OLD.starts_at IS DISTINCT FROM NEW.starts_at
        OR OLD.status IS DISTINCT FROM NEW.status
        OR OLD.daily_window_start IS DISTINCT FROM NEW.daily_window_start
        OR OLD.weekly_window_start IS DISTINCT FROM NEW.weekly_window_start
        OR OLD.monthly_window_start IS DISTINCT FROM NEW.monthly_window_start
        OR NEW.daily_usage_usd < OLD.daily_usage_usd
        OR NEW.weekly_usage_usd < OLD.weekly_usage_usd
        OR NEW.monthly_usage_usd < OLD.monthly_usage_usd
        OR OLD.deleted_at IS DISTINCT FROM NEW.deleted_at
        OR OLD.user_id IS DISTINCT FROM NEW.user_id
        OR OLD.group_id IS DISTINCT FROM NEW.group_id;
    IF NOT changed THEN
        RETURN NEW;
    END IF;

    INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
    VALUES (NEW.user_id, NEW.group_id, subscription_cache_version_nanos(NEW.updated_at));
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_user_subscriptions_cache_version ON user_subscriptions;
CREATE TRIGGER trg_user_subscriptions_cache_version
BEFORE INSERT OR UPDATE OR DELETE ON user_subscriptions
FOR EACH ROW EXECUTE FUNCTION assign_user_subscription_cache_version();

DROP TRIGGER IF EXISTS trg_user_subscriptions_cache_invalidation ON user_subscriptions;
CREATE TRIGGER trg_user_subscriptions_cache_invalidation
AFTER INSERT OR UPDATE OR DELETE ON user_subscriptions
FOR EACH ROW EXECUTE FUNCTION enqueue_user_subscription_cache_invalidation();

COMMENT ON TABLE subscription_cache_invalidation_outbox IS
    'Durable versioned invalidations for subscription authorization caches';
