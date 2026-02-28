package repository

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type UserProfileRepository struct {
	db DBTX
}

func NewUserProfileRepository(db DBTX) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

func (r *UserProfileRepository) CreateEmpty(ctx context.Context, userID int64) error {
	query := `INSERT INTO user_profiles (user_id) VALUES ($1)`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID int64) (*models.UserProfile, error) {
	query := `
		SELECT id, user_id, full_name, avatar_url, age, gender, height_cm, weight_kg,
			   fitness_level, goals, max_hourly_rate, medical_conditions, onboarding_complete, created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`
	var profile models.UserProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Age,
		&profile.Gender,
		&profile.HeightCM,
		&profile.WeightKG,
		&profile.FitnessLevel,
		&profile.Goals,
		&profile.MaxHourlyRate,
		&profile.MedicalConditions,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *UserProfileRepository) UpdateOnboarding(ctx context.Context, userID int64, req UserOnboardingInput) (*models.UserProfile, error) {
	query := `
		UPDATE user_profiles
		SET full_name = $1,
			age = $2,
			gender = $3,
			height_cm = $4,
			weight_kg = $5,
			fitness_level = $6,
			goals = $7,
			max_hourly_rate = $8,
			medical_conditions = $9,
			onboarding_complete = TRUE,
			updated_at = NOW()
		WHERE user_id = $10
		RETURNING id, user_id, full_name, avatar_url, age, gender, height_cm, weight_kg,
				  fitness_level, goals, max_hourly_rate, medical_conditions, onboarding_complete, created_at, updated_at
	`
	var profile models.UserProfile
	err := r.db.QueryRow(ctx, query,
		req.FullName,
		req.Age,
		req.Gender,
		req.HeightCM,
		req.WeightKG,
		req.FitnessLevel,
		req.Goals,
		req.MaxHourlyRate,
		req.MedicalConditions,
		userID,
	).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Age,
		&profile.Gender,
		&profile.HeightCM,
		&profile.WeightKG,
		&profile.FitnessLevel,
		&profile.Goals,
		&profile.MaxHourlyRate,
		&profile.MedicalConditions,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *UserProfileRepository) UpdatePartial(ctx context.Context, userID int64, req UpdateUserProfileInput) (*models.UserProfile, error) {
	query := `
		UPDATE user_profiles
		SET full_name = COALESCE($1, full_name),
			avatar_url = COALESCE($2, avatar_url),
			age = COALESCE($3, age),
			gender = COALESCE($4, gender),
			height_cm = COALESCE($5, height_cm),
			weight_kg = COALESCE($6, weight_kg),
			fitness_level = COALESCE($7, fitness_level),
			goals = COALESCE($8, goals),
			max_hourly_rate = COALESCE($9, max_hourly_rate),
			medical_conditions = COALESCE($10, medical_conditions),
			updated_at = NOW()
		WHERE user_id = $11
		RETURNING id, user_id, full_name, avatar_url, age, gender, height_cm, weight_kg,
				  fitness_level, goals, max_hourly_rate, medical_conditions, onboarding_complete, created_at, updated_at
	`
	var profile models.UserProfile
	err := r.db.QueryRow(ctx, query,
		req.FullName,
		req.AvatarURL,
		req.Age,
		req.Gender,
		req.HeightCM,
		req.WeightKG,
		req.FitnessLevel,
		req.Goals,
		req.MaxHourlyRate,
		req.MedicalConditions,
		userID,
	).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.AvatarURL,
		&profile.Age,
		&profile.Gender,
		&profile.HeightCM,
		&profile.WeightKG,
		&profile.FitnessLevel,
		&profile.Goals,
		&profile.MaxHourlyRate,
		&profile.MedicalConditions,
		&profile.OnboardingComplete,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

type UserOnboardingInput struct {
	FullName          string
	Age               int
	Gender            string
	HeightCM          float64
	WeightKG          float64
	FitnessLevel      string
	Goals             []string
	MaxHourlyRate     *float64
	MedicalConditions string
}

type UpdateUserProfileInput struct {
	FullName          *string
	AvatarURL         *string
	Age               *int
	Gender            *string
	HeightCM          *float64
	WeightKG          *float64
	FitnessLevel      *string
	Goals             *[]string
	MaxHourlyRate     *float64
	MedicalConditions *string
}
