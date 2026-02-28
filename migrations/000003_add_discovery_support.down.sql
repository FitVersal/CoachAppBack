DROP INDEX IF EXISTS idx_coach_availability_slots_coach_start;
DROP TABLE IF EXISTS coach_availability_slots;

DROP INDEX IF EXISTS idx_coach_reviews_coach_id;
DROP TABLE IF EXISTS coach_reviews;

ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS max_hourly_rate;
