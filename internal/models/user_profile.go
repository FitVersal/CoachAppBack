package models

import "time"

type UserProfile struct {
	ID                 int64     `json:"id"`
	UserID             int64     `json:"user_id"`
	FullName           *string   `json:"full_name"`
	AvatarURL          *string   `json:"avatar_url"`
	Age                *int      `json:"age"`
	Gender             *string   `json:"gender"`
	HeightCM           *float64  `json:"height_cm"`
	WeightKG           *float64  `json:"weight_kg"`
	FitnessLevel       *string   `json:"fitness_level"`
	Goals              *[]string `json:"goals"`
	MaxHourlyRate      *float64  `json:"max_hourly_rate"`
	MedicalConditions  *string   `json:"medical_conditions"`
	OnboardingComplete bool      `json:"onboarding_complete"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
