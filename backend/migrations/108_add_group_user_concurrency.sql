-- 108_add_group_user_concurrency.sql
-- Add per-group user concurrency limit fields
ALTER TABLE groups ADD COLUMN user_concurrency_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN user_concurrency_limit INTEGER NOT NULL DEFAULT 0;
