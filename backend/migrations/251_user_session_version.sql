-- Persistent browser/refresh/MCP revocation generation. Zero preserves existing tokens.
ALTER TABLE users ADD COLUMN IF NOT EXISTS session_version BIGINT NOT NULL DEFAULT 0;
