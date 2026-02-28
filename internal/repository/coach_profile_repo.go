package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CoachProfileRepository struct {
	db DBTX
}

type CoachListFilter struct {
	Specialization string
	MinRating      float64
	MaxPrice       float64
	Experience     int
	Offset         int
	Limit          int
}

const coachProfileSelectWithReviewStats = `
	SELECT cp.id, cp.user_id, cp.full_name, cp.avatar_url, cp.bio, cp.specializations, cp.certifications,
	       cp.experience_years, cp.hourly_rate, COALESCE(review_stats.avg_rating, cp.rating, 0) AS rating,
	       COALESCE(review_stats.total_reviews, 0) AS total_reviews, cp.total_clients, cp.is_verified,
	       cp.onboarding_complete, cp.created_at, cp.updated_at
	FROM coach_profiles cp
	LEFT JOIN (
		SELECT coach_id, AVG(rating)::DECIMAL(3,2) AS avg_rating, COUNT(*)::INT AS total_reviews
		FROM coach_reviews
		GROUP BY coach_id
	) AS review_stats ON review_stats.coach_id = cp.user_id
`

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
	` + coachProfileSelectWithReviewStats + `
		WHERE cp.user_id = $1
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
		&profile.TotalReviews,
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
		WITH updated AS (
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
			RETURNING user_id
		)
	` + coachProfileSelectWithReviewStats + `
		JOIN updated ON updated.user_id = cp.user_id
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
		&profile.TotalReviews,
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
		WITH updated AS (
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
			RETURNING user_id
		)
	` + coachProfileSelectWithReviewStats + `
		JOIN updated ON updated.user_id = cp.user_id
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
		&profile.TotalReviews,
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

func (r *CoachProfileRepository) List(ctx context.Context, filter CoachListFilter) ([]models.CoachProfile, int, error) {
	whereParts := []string{"onboarding_complete = TRUE"}
	args := make([]any, 0, 4)

	if specialization := strings.TrimSpace(filter.Specialization); specialization != "" {
		args = append(args, strings.ToLower(specialization))
		whereParts = append(whereParts, fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(specializations) AS specialization WHERE LOWER(specialization) = $%d)", len(args)))
	}
	if filter.MinRating > 0 {
		args = append(args, filter.MinRating)
		whereParts = append(whereParts, fmt.Sprintf("rating >= $%d", len(args)))
	}
	if filter.MaxPrice > 0 {
		args = append(args, filter.MaxPrice)
		whereParts = append(whereParts, fmt.Sprintf("hourly_rate <= $%d", len(args)))
	}
	if filter.Experience > 0 {
		args = append(args, filter.Experience)
		whereParts = append(whereParts, fmt.Sprintf("experience_years >= $%d", len(args)))
	}

	whereClause := strings.Join(whereParts, " AND ")
	baseQuery := strings.TrimSpace(coachProfileSelectWithReviewStats)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS coaches WHERE %s", baseQuery, whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset)
	query := fmt.Sprintf(`
		SELECT *
		FROM (%s) AS coaches
		WHERE %s
		ORDER BY rating DESC, experience_years DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseQuery, whereClause, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	coaches := make([]models.CoachProfile, 0, filter.Limit)
	for rows.Next() {
		var profile models.CoachProfile
		if err := rows.Scan(
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
			&profile.TotalReviews,
			&profile.TotalClients,
			&profile.IsVerified,
			&profile.OnboardingComplete,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		coaches = append(coaches, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return coaches, total, nil
}

func (r *CoachProfileRepository) ListAll(ctx context.Context) ([]models.CoachProfile, error) {
	query := `
	` + coachProfileSelectWithReviewStats + `
		WHERE cp.onboarding_complete = TRUE
		ORDER BY rating DESC, experience_years DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coaches []models.CoachProfile
	for rows.Next() {
		var profile models.CoachProfile
		if err := rows.Scan(
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
			&profile.TotalReviews,
			&profile.TotalClients,
			&profile.IsVerified,
			&profile.OnboardingComplete,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		); err != nil {
			return nil, err
		}
		coaches = append(coaches, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return coaches, nil
}

func (r *CoachProfileRepository) GetByCoachID(ctx context.Context, coachID int64) (*models.CoachProfile, error) {
	query := `
	` + coachProfileSelectWithReviewStats + `
		WHERE cp.user_id = $1 AND cp.onboarding_complete = TRUE
	`
	var profile models.CoachProfile
	err := r.db.QueryRow(ctx, query, coachID).Scan(
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
		&profile.TotalReviews,
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

func (r *CoachProfileRepository) GetAvailableSlotsPreview(ctx context.Context, coachID int64, limit int) ([]string, error) {
	query := `
		SELECT starts_at
		FROM coach_availability_slots
		WHERE coach_id = $1
		  AND is_booked = FALSE
		  AND starts_at >= NOW()
		ORDER BY starts_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, coachID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]string, 0, limit)
	for rows.Next() {
		var startsAt time.Time
		if err := rows.Scan(&startsAt); err != nil {
			return nil, err
		}
		slots = append(slots, startsAt.UTC().Format(time.RFC3339))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return slots, nil
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
