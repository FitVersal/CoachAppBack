package repository

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CreatePaymentInput struct {
	SessionID int64
	UserID    int64
	CoachID   int64
	Amount    float64
	Status    string
}

type PaymentRepository struct {
	db DBTX
}

func NewPaymentRepository(db DBTX) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, input CreatePaymentInput) (*models.Payment, error) {
	query := `
		INSERT INTO payments (booking_id, user_id, coach_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, booking_id, user_id, coach_id, amount, status, created_at
	`

	var payment models.Payment
	err := r.db.QueryRow(ctx, query, input.SessionID, input.UserID, input.CoachID, input.Amount, input.Status).Scan(
		&payment.ID,
		&payment.SessionID,
		&payment.UserID,
		&payment.CoachID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) GetBySessionID(ctx context.Context, sessionID int64) (*models.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, coach_id, amount, status, created_at
		FROM payments
		WHERE booking_id = $1
		ORDER BY id DESC
		LIMIT 1
	`

	var payment models.Payment
	err := r.db.QueryRow(ctx, query, sessionID).Scan(
		&payment.ID,
		&payment.SessionID,
		&payment.UserID,
		&payment.CoachID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) GetBySessionIDForUpdate(ctx context.Context, sessionID int64) (*models.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, coach_id, amount, status, created_at
		FROM payments
		WHERE booking_id = $1
		ORDER BY id DESC
		LIMIT 1
		FOR UPDATE
	`

	var payment models.Payment
	err := r.db.QueryRow(ctx, query, sessionID).Scan(
		&payment.ID,
		&payment.SessionID,
		&payment.UserID,
		&payment.CoachID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) ListBySessionIDs(ctx context.Context, sessionIDs []int64) (map[int64]models.Payment, error) {
	payments := make(map[int64]models.Payment, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return payments, nil
	}

	query := `
		SELECT DISTINCT ON (booking_id) id, booking_id, user_id, coach_id, amount, status, created_at
		FROM payments
		WHERE booking_id = ANY($1)
		ORDER BY booking_id, id DESC
	`

	rows, err := r.db.Query(ctx, query, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.SessionID,
			&payment.UserID,
			&payment.CoachID,
			&payment.Amount,
			&payment.Status,
			&payment.CreatedAt,
		); err != nil {
			return nil, err
		}
		payments[payment.SessionID] = payment
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, paymentID int64, status string) (*models.Payment, error) {
	query := `
		UPDATE payments
		SET status = $2
		WHERE id = $1
		RETURNING id, booking_id, user_id, coach_id, amount, status, created_at
	`

	var payment models.Payment
	err := r.db.QueryRow(ctx, query, paymentID, status).Scan(
		&payment.ID,
		&payment.SessionID,
		&payment.UserID,
		&payment.CoachID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) UpdateStatusIfCurrent(ctx context.Context, paymentID int64, currentStatus string, nextStatus string) (*models.Payment, error) {
	query := `
		UPDATE payments
		SET status = $3
		WHERE id = $1 AND status = $2
		RETURNING id, booking_id, user_id, coach_id, amount, status, created_at
	`

	var payment models.Payment
	err := r.db.QueryRow(ctx, query, paymentID, currentStatus, nextStatus).Scan(
		&payment.ID,
		&payment.SessionID,
		&payment.UserID,
		&payment.CoachID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}
