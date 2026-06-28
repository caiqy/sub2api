ALTER TABLE usage_log_details
    ADD COLUMN IF NOT EXISTS detail_type VARCHAR(20) NOT NULL DEFAULT 'normal';

UPDATE usage_log_details AS d
SET detail_type = 'image'
FROM usage_logs AS l
WHERE d.usage_log_id = l.id
  AND d.detail_type = 'normal'
  AND (
      COALESCE(l.inbound_endpoint, '') ILIKE '%/v1/images/generations%'
      OR COALESCE(l.inbound_endpoint, '') ILIKE '%/v1/images/edits%'
      OR COALESCE(l.upstream_endpoint, '') ILIKE '%/v1/images/generations%'
      OR COALESCE(l.upstream_endpoint, '') ILIKE '%/v1/images/edits%'
      OR LOWER(BTRIM(COALESCE(l.billing_mode, ''))) = 'image'
      OR COALESCE(l.image_count, 0) > 0
  );

-- The hand-written PostgreSQL index includes created_at DESC, id DESC to match
-- usage-log detail pruning order. Ent keeps the logical index in schema; this
-- migration provides the more specific PostgreSQL index shape.
CREATE INDEX IF NOT EXISTS idx_usage_log_details_detail_type_created_at
    ON usage_log_details (detail_type, created_at DESC, id DESC);
