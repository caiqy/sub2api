ALTER TABLE groups ADD COLUMN IF NOT EXISTS exclusive_rate_models JSONB NOT NULL DEFAULT '{"enabled":false,"models":[]}'::jsonb;

-- Durable invalidation covers direct SQL updates and the post-commit crash window.
CREATE OR REPLACE FUNCTION enqueue_group_exclusive_rate_models_invalidation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.group_id = NEW.id AND k.deleted_at IS NULL AND k.key <> '';
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg_group_exclusive_rate_models_invalidation ON groups;
CREATE TRIGGER trg_group_exclusive_rate_models_invalidation
AFTER UPDATE OF exclusive_rate_models ON groups
FOR EACH ROW WHEN (OLD.exclusive_rate_models IS DISTINCT FROM NEW.exclusive_rate_models)
EXECUTE FUNCTION enqueue_group_exclusive_rate_models_invalidation();
