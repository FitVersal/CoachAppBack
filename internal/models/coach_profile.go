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
	TotalReviews       int       `json:"total_reviews"`
	TotalClients       *int      `json:"total_clients"`
	IsVerified         *bool     `json:"is_verified"`
	OnboardingComplete bool      `json:"onboarding_complete"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CoachWithScore struct {
	CoachProfile
	MatchScore int `json:"match_score"`
}

type CoachListResponse struct {
	ID              string   `json:"id"`
	FullName        string   `json:"full_name"`
	AvatarURL       string   `json:"avatar_url"`
	Specializations []string `json:"specializations"`
	ExperienceYears int      `json:"experience_years"`
	HourlyRate      float64  `json:"hourly_rate"`
	Rating          float64  `json:"rating"`
	TotalReviews    int      `json:"total_reviews"`
	MatchScore      int      `json:"match_score,omitempty"`
}

type CoachDetailResponse struct {
	CoachListResponse
	Bio                string   `json:"bio"`
	Certifications     []string `json:"certifications"`
	IsVerified         bool     `json:"is_verified"`
	AvailableSlots     []string `json:"available_slots_preview"`
	OnboardingComplete bool     `json:"onboarding_complete"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
