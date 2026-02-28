ALTER TABLE coach_profiles
    ALTER COLUMN certifications TYPE TEXT
    USING CASE
        WHEN certifications IS NULL OR array_length(certifications, 1) IS NULL THEN NULL
        ELSE array_to_string(certifications, ', ')
    END;

ALTER TABLE coach_profiles
    RENAME COLUMN certifications TO credentials;

ALTER TABLE user_profiles
    RENAME COLUMN medical_conditions TO injuries;
