package repository

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CreateWorkoutProgramInput struct {
	CoachID     int64
	UserID      int64
	SessionID   int64
	Title       string
	Description *string
	FileURL     string
}

type WorkoutProgramRepository struct {
	db DBTX
}

func NewWorkoutProgramRepository(db DBTX) *WorkoutProgramRepository {
	return &WorkoutProgramRepository{db: db}
}

func (r *WorkoutProgramRepository) Create(
	ctx context.Context,
	input CreateWorkoutProgramInput,
) (*models.WorkoutProgram, error) {
	query := `
		INSERT INTO workout_programs (coach_id, user_id, booking_id, title, description, file_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, coach_id, user_id, booking_id, title, description, file_url, created_at
	`

	var program models.WorkoutProgram
	err := r.db.QueryRow(
		ctx,
		query,
		input.CoachID,
		input.UserID,
		input.SessionID,
		input.Title,
		input.Description,
		input.FileURL,
	).Scan(
		&program.ID,
		&program.CoachID,
		&program.UserID,
		&program.SessionID,
		&program.Title,
		&program.Description,
		&program.FileURL,
		&program.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &program, nil
}

func (r *WorkoutProgramRepository) ListByCoachID(ctx context.Context, coachID int64) ([]models.WorkoutProgram, error) {
	query := `
		SELECT id, coach_id, user_id, booking_id, title, description, file_url, created_at
		FROM workout_programs
		WHERE coach_id = $1
		ORDER BY created_at DESC, id DESC
	`
	return r.list(ctx, query, coachID)
}

func (r *WorkoutProgramRepository) ListByUserID(ctx context.Context, userID int64) ([]models.WorkoutProgram, error) {
	query := `
		SELECT id, coach_id, user_id, booking_id, title, description, file_url, created_at
		FROM workout_programs
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`
	return r.list(ctx, query, userID)
}

func (r *WorkoutProgramRepository) GetByID(ctx context.Context, programID int64) (*models.WorkoutProgram, error) {
	query := `
		SELECT id, coach_id, user_id, booking_id, title, description, file_url, created_at
		FROM workout_programs
		WHERE id = $1
	`

	var program models.WorkoutProgram
	err := r.db.QueryRow(ctx, query, programID).Scan(
		&program.ID,
		&program.CoachID,
		&program.UserID,
		&program.SessionID,
		&program.Title,
		&program.Description,
		&program.FileURL,
		&program.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &program, nil
}

func (r *WorkoutProgramRepository) list(
	ctx context.Context,
	query string,
	actorID int64,
) ([]models.WorkoutProgram, error) {
	rows, err := r.db.Query(ctx, query, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	programs := make([]models.WorkoutProgram, 0)
	for rows.Next() {
		var program models.WorkoutProgram
		if err := rows.Scan(
			&program.ID,
			&program.CoachID,
			&program.UserID,
			&program.SessionID,
			&program.Title,
			&program.Description,
			&program.FileURL,
			&program.CreatedAt,
		); err != nil {
			return nil, err
		}
		programs = append(programs, program)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return programs, nil
}
