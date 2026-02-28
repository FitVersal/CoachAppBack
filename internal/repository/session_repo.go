package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CreateSessionInput struct {
	UserID          int64
	CoachID         int64
	ScheduledAt     time.Time
	DurationMinutes int
	Notes           *string
}

type SessionListFilter struct {
	ActorID   int64
	Role      string
	Status    string
	Timeframe string
}

type SessionRepository struct {
	db DBTX
}

func NewSessionRepository(db DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(
	ctx context.Context,
	input CreateSessionInput,
) (*models.Session, error) {
	query := `
		INSERT INTO bookings (user_id, coach_id, scheduled_at, duration_min, status, notes)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
	`

	var session models.Session
	err := r.db.QueryRow(
		ctx,
		query,
		input.UserID,
		input.CoachID,
		input.ScheduledAt,
		input.DurationMinutes,
		input.Notes,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.CoachID,
		&session.ScheduledAt,
		&session.DurationMinutes,
		&session.Status,
		&session.Notes,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) GetByID(ctx context.Context, sessionID int64) (*models.Session, error) {
	query := `
		SELECT id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	var session models.Session
	err := r.db.QueryRow(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.CoachID,
		&session.ScheduledAt,
		&session.DurationMinutes,
		&session.Status,
		&session.Notes,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) GetByIDForUpdate(
	ctx context.Context,
	sessionID int64,
) (*models.Session, error) {
	query := `
		SELECT id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
		FROM bookings
		WHERE id = $1
		FOR UPDATE
	`
	var session models.Session
	err := r.db.QueryRow(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.CoachID,
		&session.ScheduledAt,
		&session.DurationMinutes,
		&session.Status,
		&session.Notes,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) List(
	ctx context.Context,
	filter SessionListFilter,
) ([]models.Session, error) {
	actorColumn := "user_id"
	if filter.Role == "coach" {
		actorColumn = "coach_id"
	}

	args := []any{filter.ActorID}
	whereParts := []string{fmt.Sprintf("%s = $1", actorColumn)}

	if status := strings.TrimSpace(filter.Status); status != "" {
		args = append(args, status)
		whereParts = append(whereParts, fmt.Sprintf("status = $%d", len(args)))
	}

	switch strings.TrimSpace(filter.Timeframe) {
	case "upcoming":
		whereParts = append(
			whereParts,
			"(scheduled_at + (duration_min * INTERVAL '1 minute')) > NOW()",
		)
	case "past":
		whereParts = append(
			whereParts,
			"(scheduled_at + (duration_min * INTERVAL '1 minute')) <= NOW()",
		)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
		FROM bookings
		WHERE %s
		ORDER BY scheduled_at ASC, id ASC
	`, strings.Join(whereParts, " AND "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]models.Session, 0)
	for rows.Next() {
		var session models.Session
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.CoachID,
			&session.ScheduledAt,
			&session.DurationMinutes,
			&session.Status,
			&session.Notes,
			&session.CreatedAt,
			&session.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *SessionRepository) UpdateStatus(
	ctx context.Context,
	sessionID int64,
	status string,
) (*models.Session, error) {
	query := `
		UPDATE bookings
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
	`
	var session models.Session
	err := r.db.QueryRow(ctx, query, sessionID, status).Scan(
		&session.ID,
		&session.UserID,
		&session.CoachID,
		&session.ScheduledAt,
		&session.DurationMinutes,
		&session.Status,
		&session.Notes,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) UpdateStatusIfCurrent(
	ctx context.Context,
	sessionID int64,
	currentStatus string,
	nextStatus string,
) (*models.Session, error) {
	query := `
		UPDATE bookings
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND status = $2
		RETURNING id, user_id, coach_id, scheduled_at, duration_min, status, notes, created_at, updated_at
	`
	var session models.Session
	err := r.db.QueryRow(ctx, query, sessionID, currentStatus, nextStatus).Scan(
		&session.ID,
		&session.UserID,
		&session.CoachID,
		&session.ScheduledAt,
		&session.DurationMinutes,
		&session.Status,
		&session.Notes,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) HasConflict(
	ctx context.Context,
	coachID int64,
	requestedTime time.Time,
	durationMinutes int,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM bookings
			WHERE coach_id = $1
			  AND status <> 'cancelled'
			  AND scheduled_at < ($2::timestamp + ($3::int * INTERVAL '1 minute'))
			  AND (scheduled_at + (duration_min * INTERVAL '1 minute')) > $2::timestamp
		)
	`
	var hasConflict bool
	if err := r.db.QueryRow(ctx, query, coachID, requestedTime, durationMinutes).Scan(&hasConflict); err != nil {
		return false, err
	}
	return hasConflict, nil
}

func (r *SessionRepository) HasConflictExcludingSession(
	ctx context.Context,
	coachID int64,
	requestedTime time.Time,
	durationMinutes int,
	excludedSessionID int64,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM bookings
			WHERE coach_id = $1
			  AND id <> $4
			  AND status <> 'cancelled'
			  AND scheduled_at < ($2::timestamp + ($3::int * INTERVAL '1 minute'))
			  AND (scheduled_at + (duration_min * INTERVAL '1 minute')) > $2::timestamp
		)
	`
	var hasConflict bool
	if err := r.db.QueryRow(ctx, query, coachID, requestedTime, durationMinutes, excludedSessionID).Scan(&hasConflict); err != nil {
		return false, err
	}
	return hasConflict, nil
}
