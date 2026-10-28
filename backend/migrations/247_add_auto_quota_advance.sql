-- 订阅分组及用户订阅是否启用额度用尽自动提前重置周期（auto-quota-advance）。
-- groups 表增加 auto_quota_advance 字段，默认 false。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS auto_quota_advance BOOLEAN NOT NULL DEFAULT false;

-- user_subscriptions 表增加 auto_quota_advance 字段，默认 false。
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS auto_quota_advance BOOLEAN NOT NULL DEFAULT false;
