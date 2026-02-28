package repository

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CoachProfileRepository struct {
	db DBTX
}

func NewCoachProfileRepository(db DBTX) *CoachProfileRepository {
	return &CoachProfileRepository{db: db}
}

func (r *CoachProfileRepository) CreateEmpty(ctx context.Context, userID int64) error {
	query := `INSERT INTO coach_profiles (user_id) VALUES ($1)`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *CoachProfileRepository) GetByUserID(ctx context.Context, userID int64) (*models.CoachProfile, error) {
	query := `
		SELECT id, user_id, full_name, avatar_url, bio, specializations, certifications,
			   experience_years, hourly_rate, rating, total_clients, is_verified,
			   onboarding_complete, created_at, updated_at
		FROM coach_profiles
		WHERE user_id = $1
	`
	var profile models.CoachProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Specializations,
		&profile.Certifications,
		&profile.ExperienceYears,
		&profile.HourlyRate,
		&profile.Rating,
		&profile.TotalClients,
		&profile.IsVerified,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *CoachProfileRepository) UpdateOnboarding(ctx context.Context, userID int64, req CoachOnboardingInput) (*models.CoachProfile, error) {
	query := `
		UPDATE coach_profiles
		SET full_name = $1,
			bio = $2,
			specializations = $3,
			certifications = $4,
			experience_years = $5,
			hourly_rate = $6,
			onboarding_complete = TRUE,
			updated_at = NOW()
		WHERE user_id = $7
		RETURNING id, user_id, full_name, avatar_url, bio, specializations, certifications,
				  experience_years, hourly_rate, rating, total_clients, is_verified,
				  onboarding_complete, created_at, updated_at
	`
	var profile models.CoachProfile
	err := r.db.QueryRow(ctx, query,
		req.FullName,
		req.Bio,
		req.Specializations,
		req.Certifications,
		req.ExperienceYears,
		req.HourlyRate,
		userID,
	).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Specializations,
		&profile.Certifications,
		&profile.ExperienceYears,
		&profile.HourlyRate,
		&profile.Rating,
		&profile.TotalClients,
		&profile.IsVerified,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *CoachProfileRepository) UpdatePartial(ctx context.Context, userID int64, req UpdateCoachProfileInput) (*models.CoachProfile, error) {
	query := `
		UPDATE coach_profiles
		SET full_name = COALESCE($1, full_name),
			avatar_url = COALESCE($2, avatar_url),
			bio = COALESCE($3, bio),
			specializations = COALESCE($4, specializations),
			certifications = COALESCE($5, certifications),
			experience_years = COALESCE($6, experience_years),
			hourly_rate = COALESCE($7, hourly_rate),
			updated_at = NOW()
		WHERE user_id = $8
		RETURNING id, user_id, full_name, avatar_url, bio, specializations, certifications,
				  experience_years, hourly_rate, rating, total_clients, is_verified,
				  onboarding_complete, created_at, updated_at
	`
	var profile models.CoachProfile
	err := r.db.QueryRow(ctx, query,
		req.FullName,
		req.AvatarURL,
		req.Bio,
		req.Specializations,
		req.Certifications,
		req.ExperienceYears,
		req.HourlyRate,
		userID,
	).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Specializations,
		&profile.Certifications,
		&profile.ExperienceYears,
		&profile.HourlyRate,
		&profile.Rating,
		&profile.TotalClients,
		&profile.IsVerified,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

type CoachOnboardingInput struct {
	FullName        string
	Bio             string
	Specializations []string
	Certifications  []string
	ExperienceYears int
	HourlyRate      float64
}

type UpdateCoachProfileInput struct {
	FullName        *string
	AvatarURL       *string
	Bio             *string
	Specializations *[]string
	Certifications  *[]string
	ExperienceYears *int
	HourlyRate      *float64
}
