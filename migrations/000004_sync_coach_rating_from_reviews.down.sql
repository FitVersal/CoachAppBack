DROP TRIGGER IF EXISTS trg_sync_coach_profile_rating_from_reviews ON coach_reviews;

DROP FUNCTION IF EXISTS sync_coach_profile_rating_from_reviews();
DROP FUNCTION IF EXISTS refresh_coach_profile_rating(BIGINT);
