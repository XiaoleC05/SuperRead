-- 015_briefing_range: Add briefing_range column to user_settings

ALTER TABLE superread.user_settings ADD COLUMN IF NOT EXISTS briefing_range VARCHAR(8) DEFAULT '24h';