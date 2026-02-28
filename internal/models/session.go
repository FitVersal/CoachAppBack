package models

import "time"

type Session struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CoachID         int64     `json:"coach_id"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
	Notes           *string   `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Payment struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"session_id"`
	UserID    int64     `json:"user_id"`
	CoachID   int64     `json:"coach_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionDetail struct {
	Session
	Payment *Payment `json:"payment,omitempty"`
}
