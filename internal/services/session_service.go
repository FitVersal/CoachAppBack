package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

var (
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrInvalidInput           = errors.New("invalid input")
	ErrCoachNotFound          = errors.New("coach not found")
)

type coachProfileReader interface {
	GetByUserID(ctx context.Context, userID int64) (*models.CoachProfile, error)
}

type userReader interface {
	GetByID(ctx context.Context, id int64) (*models.User, error)
}

type SessionService struct {
	db               *pgxpool.Pool
	sessionRepo      *repository.SessionRepository
	paymentRepo      *repository.PaymentRepository
	userRepo         userReader
	coachProfileRepo coachProfileReader
}

func NewSessionService(
	db *pgxpool.Pool,
	sessionRepo *repository.SessionRepository,
	paymentRepo *repository.PaymentRepository,
	userRepo userReader,
	coachProfileRepo coachProfileReader,
) *SessionService {
	return &SessionService{
		db:               db,
		sessionRepo:      sessionRepo,
		paymentRepo:      paymentRepo,
		userRepo:         userRepo,
		coachProfileRepo: coachProfileRepo,
	}
}

type BookSessionInput struct {
	CoachID         int64
	ScheduledAt     time.Time
	DurationMinutes int
	Notes           *string
}

func (s *SessionService) BookSession(
	ctx context.Context,
	userID int64,
	input BookSessionInput,
) (*models.SessionDetail, error) {
	if input.CoachID <= 0 || input.DurationMinutes <= 0 {
		return nil, ErrInvalidInput
	}
	if input.ScheduledAt.Before(time.Now().Add(-1 * time.Minute)) {
		return nil, ErrInvalidInput
	}
	if userID == input.CoachID {
		return nil, ErrInvalidInput
	}

	coach, err := s.userRepo.GetByID(ctx, input.CoachID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoachNotFound
		}
		return nil, err
	}
	if coach.Role != "coach" {
		return nil, ErrInvalidInput
	}

	coachProfile, err := s.coachProfileRepo.GetByUserID(ctx, input.CoachID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoachNotFound
		}
		return nil, err
	}
	if !coachProfile.OnboardingComplete || coachProfile.HourlyRate == nil ||
		*coachProfile.HourlyRate <= 0 {
		return nil, ErrInvalidInput
	}

	amount := 0.0
	if coachProfile.HourlyRate != nil {
		amount = *coachProfile.HourlyRate * float64(input.DurationMinutes) / 60
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txSessionRepo := repository.NewSessionRepository(tx)
	txPaymentRepo := repository.NewPaymentRepository(tx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", input.CoachID); err != nil {
		return nil, err
	}

	hasConflict, err := txSessionRepo.HasConflict(
		ctx,
		input.CoachID,
		input.ScheduledAt.UTC(),
		input.DurationMinutes,
	)
	if err != nil {
		return nil, err
	}
	if hasConflict {
		return nil, ErrConflict
	}

	session, err := txSessionRepo.Create(ctx, repository.CreateSessionInput{
		UserID:          userID,
		CoachID:         input.CoachID,
		ScheduledAt:     input.ScheduledAt.UTC(),
		DurationMinutes: input.DurationMinutes,
		Notes:           input.Notes,
	})
	if err != nil {
		return nil, err
	}

	payment, err := txPaymentRepo.Create(ctx, repository.CreatePaymentInput{
		SessionID: session.ID,
		UserID:    userID,
		CoachID:   input.CoachID,
		Amount:    amount,
		Status:    "placeholder",
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &models.SessionDetail{
		Session: *session,
		Payment: payment,
	}, nil
}

func (s *SessionService) CheckAvailability(
	ctx context.Context,
	coachID int64,
	requestedTime time.Time,
	durationMins int,
) (bool, error) {
	hasConflict, err := s.sessionRepo.HasConflict(ctx, coachID, requestedTime.UTC(), durationMins)
	if err != nil {
		return false, err
	}
	return !hasConflict, nil
}

func (s *SessionService) ListSessions(
	ctx context.Context,
	actorID int64,
	role string,
	filter repository.SessionListFilter,
) ([]models.SessionDetail, error) {
	sessions, err := s.sessionRepo.List(ctx, repository.SessionListFilter{
		ActorID:   actorID,
		Role:      role,
		Status:    filter.Status,
		Timeframe: filter.Timeframe,
	})
	if err != nil {
		return nil, err
	}

	sessionIDs := make([]int64, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
	}

	paymentsBySession, err := s.paymentRepo.ListBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}

	details := make([]models.SessionDetail, 0, len(sessions))
	for _, session := range sessions {
		detail := models.SessionDetail{Session: session}
		if payment, ok := paymentsBySession[session.ID]; ok {
			paymentCopy := payment
			detail.Payment = &paymentCopy
		}
		details = append(details, detail)
	}

	return details, nil
}

func (s *SessionService) GetSession(
	ctx context.Context,
	actorID int64,
	role string,
	sessionID int64,
) (*models.SessionDetail, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !canAccessSession(role, actorID, session) {
		return nil, ErrForbidden
	}

	detail := &models.SessionDetail{Session: *session}
	payment, err := s.paymentRepo.GetBySessionID(ctx, sessionID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		detail.Payment = payment
	}
	return detail, nil
}

func (s *SessionService) UpdateStatus(
	ctx context.Context,
	actorID int64,
	role string,
	sessionID int64,
	requestedStatus string,
) (*models.SessionDetail, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !canAccessSession(role, actorID, session) {
		return nil, ErrForbidden
	}

	nextStatus, err := normalizeRequestedStatus(requestedStatus)
	if err != nil {
		return nil, err
	}
	if err := validateStatusTransition(role, actorID, session, nextStatus); err != nil {
		return nil, err
	}
	if nextStatus == "confirmed" {
		payment, err := s.paymentRepo.GetBySessionID(ctx, sessionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrInvalidStateTransition
			}
			return nil, err
		}
		if payment.Status != "paid" {
			return nil, ErrInvalidStateTransition
		}
	}

	updated, err := s.sessionRepo.UpdateStatusIfCurrent(ctx, sessionID, session.Status, nextStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidStateTransition
		}
		return nil, err
	}
	return s.GetSession(ctx, actorID, role, updated.ID)
}

func (s *SessionService) PayForSession(
	ctx context.Context,
	actorID int64,
	role string,
	sessionID int64,
) (*models.SessionDetail, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txPaymentRepo := repository.NewPaymentRepository(tx)
	txSessionRepo := repository.NewSessionRepository(tx)

	session, err := txSessionRepo.GetByIDForUpdate(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if role != "user" || session.UserID != actorID {
		return nil, ErrForbidden
	}
	payment, err := txPaymentRepo.GetBySessionIDForUpdate(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if payment.Status == "paid" {
		return s.GetSession(ctx, actorID, role, sessionID)
	}
	if session.Status != "pending" {
		return nil, ErrInvalidStateTransition
	}
	if !session.ScheduledAt.After(time.Now().UTC()) {
		return nil, ErrInvalidStateTransition
	}

	if _, err := txPaymentRepo.UpdateStatusIfCurrent(ctx, payment.ID, "placeholder", "paid"); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidStateTransition
		}
		return nil, err
	}
	if _, err := txSessionRepo.UpdateStatusIfCurrent(ctx, sessionID, "pending", "confirmed"); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidStateTransition
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetSession(ctx, actorID, role, sessionID)
}

func canAccessSession(role string, actorID int64, session *models.Session) bool {
	if role == "user" {
		return session.UserID == actorID
	}
	if role == "coach" {
		return session.CoachID == actorID
	}
	return false
}

func normalizeRequestedStatus(status string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "confirm", "confirmed":
		return "confirmed", nil
	case "complete", "completed":
		return "completed", nil
	case "cancel", "cancelled", "canceled":
		return "cancelled", nil
	default:
		return "", ErrInvalidStatus
	}
}

func validateStatusTransition(
	role string,
	actorID int64,
	session *models.Session,
	nextStatus string,
) error {
	switch role {
	case "user":
		if session.UserID != actorID || nextStatus != "cancelled" {
			return ErrForbidden
		}
		if session.Status == "completed" || session.Status == "cancelled" {
			return ErrInvalidStateTransition
		}
		return nil
	case "coach":
		if session.CoachID != actorID {
			return ErrForbidden
		}
		switch nextStatus {
		case "confirmed":
			if session.Status != "pending" {
				return ErrInvalidStateTransition
			}
		case "completed":
			if session.Status != "confirmed" {
				return ErrInvalidStateTransition
			}
			sessionEnd := session.ScheduledAt.UTC().Add(time.Duration(session.DurationMinutes) * time.Minute)
			if sessionEnd.After(time.Now().UTC()) {
				return ErrInvalidStateTransition
			}
		case "cancelled":
			if session.Status == "completed" || session.Status == "cancelled" {
				return ErrInvalidStateTransition
			}
		default:
			return ErrInvalidStatus
		}
		return nil
	default:
		return ErrForbidden
	}
}
