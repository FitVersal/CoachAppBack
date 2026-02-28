package models

import "time"

type CoachProfile struct {
	ID                 int64     `json:"id"`
	UserID             int64     `json:"user_id"`
	FullName           *string   `json:"full_name"`
	AvatarURL          *string   `json:"avatar_url"`
	Bio                *string   `json:"bio"`
	Specializations    *[]string `json:"specializations"`
	Certifications     *[]string `json:"certifications"`
	ExperienceYears    *int      `json:"experience_years"`
	HourlyRate         *float64  `json:"hourly_rate"`
	Rating             *float64  `json:"rating"`
	TotalClients       *int      `json:"total_clients"`
	IsVerified         *bool     `json:"is_verified"`
	OnboardingComplete bool      `json:"onboarding_complete"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
