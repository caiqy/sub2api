ALTER TABLE users ADD COLUMN IF NOT EXISTS allow_large_requests boolean NOT NULL DEFAULT false;
ALTER TABLE usage_logs
 ADD COLUMN IF NOT EXISTS traffic_cost numeric(20,10) NOT NULL DEFAULT 0,
 ADD COLUMN IF NOT EXISTS traffic_input_bytes bigint,
 ADD COLUMN IF NOT EXISTS traffic_threshold_bytes bigint,
 ADD COLUMN IF NOT EXISTS traffic_price_per_mb double precision,
 ADD COLUMN IF NOT EXISTS traffic_billing_reason text NOT NULL DEFAULT '';
