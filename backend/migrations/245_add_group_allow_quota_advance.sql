-- 订阅分组是否允许用户自助提前重置额度周期（advance-quota-cycle）。
-- 存量分组必须保持自助能力可用，因此默认 true；管理员手动重置额度不受该开关约束。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS allow_quota_advance BOOLEAN NOT NULL DEFAULT true;
