-- Add upstream request payload snapshot fields.
-- 幂等执行：允许线上已手工补过部分结构时重复运行。

ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS upstream_request_headers TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS upstream_request_body TEXT;

UPDATE usage_log_details
SET upstream_request_headers = ''
WHERE upstream_request_headers IS NULL;

UPDATE usage_log_details
SET upstream_request_body = ''
WHERE upstream_request_body IS NULL;

ALTER TABLE usage_log_details ALTER COLUMN upstream_request_headers SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN upstream_request_headers SET NOT NULL;

ALTER TABLE usage_log_details ALTER COLUMN upstream_request_body SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN upstream_request_body SET NOT NULL;
