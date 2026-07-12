-- 013_user_email: Add email column to user_settings for briefing delivery

ALTER TABLE superread.user_settings ADD COLUMN IF NOT EXISTS email VARCHAR(256) DEFAULT '';