-- Default new users to accepting large requests and enable existing users once.
ALTER TABLE users ALTER COLUMN allow_large_requests SET DEFAULT true;
UPDATE users SET allow_large_requests = true WHERE allow_large_requests = false;
