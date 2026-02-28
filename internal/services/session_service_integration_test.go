package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

var (
	testDBOnce sync.Once
	testDBPool *pgxpool.Pool
	testDBErr  error
)

func TestSessionServiceBookAndPayFlow(t *testing.T) {
	ctx := context.Background()
	pool := integrationTestPool(t)
	service := newIntegrationSessionService(pool)

	userID := createTestAccount(t, ctx, pool, "user", 0)
	coachID := createTestAccount(t, ctx, pool, "coach", 120)
	t.Cleanup(func() { cleanupTestUsers(t, ctx, pool, userID, coachID) })

	scheduledAt := time.Date(2030, 3, 15, 9, 0, 0, 0, time.UTC)
	detail, err := service.BookSession(ctx, userID, BookSessionInput{
		CoachID:         coachID,
		ScheduledAt:     scheduledAt,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("BookSession: %v", err)
	}

	if detail.Status != "pending" {
		t.Fatalf("expected pending session, got %q", detail.Status)
	}
	if detail.Payment == nil || detail.Payment.Status != "placeholder" {
		t.Fatalf("expected placeholder payment, got %+v", detail.Payment)
	}
	if detail.Payment.Amount != 180 {
		t.Fatalf("expected amount 180, got %.2f", detail.Payment.Amount)
	}

	paidDetail, err := service.PayForSession(ctx, userID, "user", detail.ID)
	if err != nil {
		t.Fatalf("PayForSession: %v", err)
	}

	if paidDetail.Status != "confirmed" {
		t.Fatalf("expected confirmed session after payment, got %q", paidDetail.Status)
	}
	if paidDetail.Payment == nil || paidDetail.Payment.Status != "paid" {
		t.Fatalf("expected paid payment, got %+v", paidDetail.Payment)
	}
}

func TestSessionServiceRejectsOverlappingBookings(t *testing.T) {
	ctx := context.Background()
	pool := integrationTestPool(t)
	service := newIntegrationSessionService(pool)

	firstUserID := createTestAccount(t, ctx, pool, "user", 0)
	secondUserID := createTestAccount(t, ctx, pool, "user", 0)
	coachID := createTestAccount(t, ctx, pool, "coach", 80)
	t.Cleanup(func() { cleanupTestUsers(t, ctx, pool, firstUserID, secondUserID, coachID) })

	scheduledAt := time.Date(2030, 4, 1, 12, 0, 0, 0, time.UTC)
	if _, err := service.BookSession(ctx, firstUserID, BookSessionInput{
		CoachID:         coachID,
		ScheduledAt:     scheduledAt,
		DurationMinutes: 60,
	}); err != nil {
		t.Fatalf("first BookSession: %v", err)
	}

	_, err := service.BookSession(ctx, secondUserID, BookSessionInput{
		CoachID:         coachID,
		ScheduledAt:     scheduledAt.Add(30 * time.Minute),
		DurationMinutes: 45,
	})
	if err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestSessionServiceListsSessionsForBothSides(t *testing.T) {
	ctx := context.Background()
	pool := integrationTestPool(t)
	service := newIntegrationSessionService(pool)

	userID := createTestAccount(t, ctx, pool, "user", 0)
	coachID := createTestAccount(t, ctx, pool, "coach", 95)
	t.Cleanup(func() { cleanupTestUsers(t, ctx, pool, userID, coachID) })

	upcoming := time.Date(2030, 5, 10, 8, 0, 0, 0, time.UTC)
	booked, err := service.BookSession(ctx, userID, BookSessionInput{
		CoachID:         coachID,
		ScheduledAt:     upcoming,
		DurationMinutes: 60,
	})
	if err != nil {
		t.Fatalf("BookSession: %v", err)
	}

	userSessions, err := service.ListSessions(ctx, userID, "user", repository.SessionListFilter{
		Status:    "pending",
		Timeframe: "upcoming",
	})
	if err != nil {
		t.Fatalf("ListSessions user: %v", err)
	}
	if len(userSessions) != 1 || userSessions[0].ID != booked.ID {
		t.Fatalf("expected user to see session %d, got %+v", booked.ID, userSessions)
	}
	if userSessions[0].Payment == nil || userSessions[0].Payment.Status != "placeholder" {
		t.Fatalf("expected placeholder payment in list, got %+v", userSessions[0].Payment)
	}

	coachSessions, err := service.ListSessions(ctx, coachID, "coach", repository.SessionListFilter{
		Timeframe: "upcoming",
	})
	if err != nil {
		t.Fatalf("ListSessions coach: %v", err)
	}
	if len(coachSessions) != 1 || coachSessions[0].ID != booked.ID {
		t.Fatalf("expected coach to see session %d, got %+v", booked.ID, coachSessions)
	}
}

func integrationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	testDBOnce.Do(func() {
		_ = godotenv.Load(".env")
		_ = godotenv.Load(filepath.Join("..", "..", ".env"))

		dbURL := os.Getenv("DB_URL")
		if dbURL == "" {
			testDBErr = fmt.Errorf("DB_URL is not set")
			return
		}

		cfg, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			testDBErr = err
			return
		}

		testDBPool, testDBErr = pgxpool.NewWithConfig(context.Background(), cfg)
		if testDBErr != nil {
			return
		}
		testDBErr = testDBPool.Ping(context.Background())
	})

	if testDBErr != nil {
		t.Skipf("skipping integration test: %v", testDBErr)
	}
	return testDBPool
}

func newIntegrationSessionService(pool *pgxpool.Pool) *SessionService {
	return NewSessionService(
		pool,
		repository.NewSessionRepository(pool),
		repository.NewPaymentRepository(pool),
		repository.NewUserRepository(pool),
		repository.NewCoachProfileRepository(pool),
	)
}

func createTestAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role string, hourlyRate float64) int64 {
	t.Helper()

	userRepo := repository.NewUserRepository(pool)
	user := &models.User{
		Email:        fmt.Sprintf("session-test-%s-%d@example.com", role, time.Now().UnixNano()),
		PasswordHash: "test-hash",
		Role:         role,
	}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser(%s): %v", role, err)
	}

	if role == "user" {
		userProfileRepo := repository.NewUserProfileRepository(pool)
		if err := userProfileRepo.CreateEmpty(ctx, user.ID); err != nil {
			t.Fatalf("CreateEmpty user profile: %v", err)
		}
		return user.ID
	}

	coachProfileRepo := repository.NewCoachProfileRepository(pool)
	if err := coachProfileRepo.CreateEmpty(ctx, user.ID); err != nil {
		t.Fatalf("CreateEmpty coach profile: %v", err)
	}
	if _, err := coachProfileRepo.UpdateOnboarding(ctx, user.ID, repository.CoachOnboardingInput{
		FullName:        "Test Coach",
		Bio:             "Test Bio",
		Specializations: []string{"fitness"},
		Certifications:  []string{"cert"},
		ExperienceYears: 1,
		HourlyRate:      hourlyRate,
	}); err != nil {
		t.Fatalf("UpdateOnboarding coach profile: %v", err)
	}

	return user.ID
}

func cleanupTestUsers(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userIDs ...int64) {
	t.Helper()

	if len(userIDs) == 0 {
		return
	}

	if _, err := pool.Exec(ctx, "DELETE FROM payments WHERE user_id = ANY($1) OR coach_id = ANY($1)", userIDs); err != nil {
		t.Fatalf("cleanup payments: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM workout_programs WHERE user_id = ANY($1) OR coach_id = ANY($1) OR booking_id IN (SELECT id FROM bookings WHERE user_id = ANY($1) OR coach_id = ANY($1))", userIDs); err != nil {
		t.Fatalf("cleanup workout programs: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM bookings WHERE user_id = ANY($1) OR coach_id = ANY($1)", userIDs); err != nil {
		t.Fatalf("cleanup bookings: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)", userIDs); err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}
