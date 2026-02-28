-- 001_create_users.sql
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL CHECK (role IN ('user', 'coach')),
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW()
);

-- 002_create_user_profiles.sql
CREATE TABLE user_profiles (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name      VARCHAR(100),
    avatar_url     VARCHAR(500),
    age            INT,
    gender         VARCHAR(20),
    height_cm      DECIMAL(5,1),
    weight_kg      DECIMAL(5,1),
    fitness_level  VARCHAR(30),  -- 'beginner', 'intermediate', 'advanced'
    goals          TEXT[],       -- '{weight_loss, muscle_gain, flexibility}'
    medical_conditions TEXT,
    onboarding_complete BOOLEAN DEFAULT FALSE,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW()
);

-- 003_create_coach_profiles.sql
CREATE TABLE coach_profiles (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name        VARCHAR(100),
    avatar_url       VARCHAR(500),
    bio              TEXT,
    specializations  TEXT[],    -- '{weight_loss, strength, yoga, hiit}'
    certifications   TEXT[],
    experience_years INT,
    hourly_rate      DECIMAL(10,2),
    rating           DECIMAL(3,2) DEFAULT 0.00,
    total_clients    INT DEFAULT 0,
    is_verified      BOOLEAN DEFAULT FALSE,
    onboarding_complete BOOLEAN DEFAULT FALSE,
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);

-- 004_create_conversations.sql
CREATE TABLE conversations (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id),
    coach_id   BIGINT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, coach_id)
);

-- 005_create_messages.sql
CREATE TABLE messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       BIGINT REFERENCES users(id),
    content         TEXT NOT NULL,
    is_read         BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);

-- 006_create_bookings.sql
CREATE TABLE bookings (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id),
    coach_id    BIGINT REFERENCES users(id),
    scheduled_at TIMESTAMP NOT NULL,
    duration_min INT DEFAULT 60,
    status       VARCHAR(20) DEFAULT 'pending'
                 CHECK (status IN ('pending','confirmed','completed','cancelled')),
    notes        TEXT,
    created_at   TIMESTAMP DEFAULT NOW(),
    updated_at   TIMESTAMP DEFAULT NOW()
);

-- 007_create_workout_programs.sql
CREATE TABLE workout_programs (
    id          BIGSERIAL PRIMARY KEY,
    coach_id    BIGINT REFERENCES users(id),
    user_id     BIGINT REFERENCES users(id),
    booking_id  BIGINT REFERENCES bookings(id),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    file_url    VARCHAR(500),
    created_at  TIMESTAMP DEFAULT NOW()
);

-- 008_create_payments.sql
CREATE TABLE payments (
    id         BIGSERIAL PRIMARY KEY,
    booking_id BIGINT REFERENCES bookings(id),
    user_id    BIGINT REFERENCES users(id),
    coach_id   BIGINT REFERENCES users(id),
    amount     DECIMAL(10,2) NOT NULL,
    status     VARCHAR(20) DEFAULT 'placeholder'
               CHECK (status IN ('placeholder','completed','refunded')),
    created_at TIMESTAMP DEFAULT NOW()
);
