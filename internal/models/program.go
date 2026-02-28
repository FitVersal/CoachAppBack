package models

import "time"

type WorkoutProgram struct {
	ID          int64     `json:"id"`
	CoachID     int64     `json:"coach_id"`
	UserID      int64     `json:"user_id"`
	SessionID   int64     `json:"session_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	FileURL     string    `json:"file_url"`
	CreatedAt   time.Time `json:"created_at"`
}
