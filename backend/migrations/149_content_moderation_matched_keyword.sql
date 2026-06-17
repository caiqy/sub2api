-- Store the keyword that triggered a content moderation keyword block.

ALTER TABLE content_moderation_logs ADD COLUMN IF NOT EXISTS matched_keyword TEXT NOT NULL DEFAULT '';
