CREATE OR REPLACE FUNCTION refresh_coach_profile_rating(target_coach_id BIGINT)
RETURNS VOID
LANGUAGE SQL
AS $$
    UPDATE coach_profiles
    SET rating = COALESCE((
            SELECT ROUND(AVG(rating)::numeric, 2)
            FROM coach_reviews
            WHERE coach_id = target_coach_id
        ), 0.00),
        updated_at = NOW()
    WHERE user_id = target_coach_id;
$$;

CREATE OR REPLACE FUNCTION sync_coach_profile_rating_from_reviews()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_coach_profile_rating(OLD.coach_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_coach_profile_rating(NEW.coach_id);

    IF TG_OP = 'UPDATE' AND OLD.coach_id <> NEW.coach_id THEN
        PERFORM refresh_coach_profile_rating(OLD.coach_id);
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_sync_coach_profile_rating_from_reviews ON coach_reviews;

CREATE TRIGGER trg_sync_coach_profile_rating_from_reviews
AFTER INSERT OR UPDATE OR DELETE ON coach_reviews
FOR EACH ROW
EXECUTE FUNCTION sync_coach_profile_rating_from_reviews();

UPDATE coach_profiles cp
SET rating = COALESCE(review_stats.avg_rating, 0.00),
    updated_at = NOW()
FROM (
    SELECT coach_id, ROUND(AVG(rating)::numeric, 2) AS avg_rating
    FROM coach_reviews
    GROUP BY coach_id
) AS review_stats
WHERE cp.user_id = review_stats.coach_id;

UPDATE coach_profiles
SET rating = 0.00,
    updated_at = NOW()
WHERE user_id NOT IN (SELECT DISTINCT coach_id FROM coach_reviews);
