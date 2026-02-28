ALTER TABLE user_profiles
    RENAME COLUMN injuries TO medical_conditions;

ALTER TABLE coach_profiles
    RENAME COLUMN credentials TO certifications;

ALTER TABLE coach_profiles
    ALTER COLUMN certifications TYPE TEXT[]
    USING CASE
        WHEN certifications IS NULL OR btrim(certifications) = '' THEN NULL
        ELSE ARRAY[certifications]
    END;
