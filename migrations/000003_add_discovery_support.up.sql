ALTER TABLE user_profiles
    ADD COLUMN max_hourly_rate DECIMAL(10,2);

CREATE TABLE coach_reviews (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    coach_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating     INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, coach_id)
);

CREATE INDEX idx_coach_reviews_coach_id ON coach_reviews(coach_id);

CREATE TABLE coach_availability_slots (
    id         BIGSERIAL PRIMARY KEY,
    coach_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    starts_at  TIMESTAMP NOT NULL,
    ends_at    TIMESTAMP NOT NULL,
    is_booked  BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

CREATE INDEX idx_coach_availability_slots_coach_start
    ON coach_availability_slots(coach_id, starts_at)
    WHERE is_booked = FALSE;
