-- Add upstream response payload snapshot fields.
-- 幂等执行：允许线上已手工补过部分结构时重复运行。

ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS upstream_response_headers TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS upstream_response_body TEXT;

UPDATE usage_log_details
SET upstream_response_headers = ''
WHERE upstream_response_headers IS NULL;

UPDATE usage_log_details
SET upstream_response_body = ''
WHERE upstream_response_body IS NULL;

ALTER TABLE usage_log_details ALTER COLUMN upstream_response_headers SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN upstream_response_headers SET NOT NULL;

ALTER TABLE usage_log_details ALTER COLUMN upstream_response_body SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN upstream_response_body SET NOT NULL;
