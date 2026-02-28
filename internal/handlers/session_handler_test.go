package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type stubSessionService struct {
	bookResult         *models.SessionDetail
	bookErr            error
	listResult         []models.SessionDetail
	listErr            error
	getResult          *models.SessionDetail
	getErr             error
	updateStatusResult *models.SessionDetail
	updateStatusErr    error
	payResult          *models.SessionDetail
	payErr             error
	lastBookInput      services.BookSessionInput
	lastActorID        int64
	lastRole           string
	lastSessionID      int64
	lastStatus         string
	lastListFilter     repository.SessionListFilter
}

func (s *stubSessionService) BookSession(_ context.Context, userID int64, input services.BookSessionInput) (*models.SessionDetail, error) {
	s.lastActorID = userID
	s.lastBookInput = input
	return s.bookResult, s.bookErr
}

func (s *stubSessionService) ListSessions(_ context.Context, actorID int64, role string, filter repository.SessionListFilter) ([]models.SessionDetail, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastListFilter = filter
	return s.listResult, s.listErr
}

func (s *stubSessionService) GetSession(_ context.Context, actorID int64, role string, sessionID int64) (*models.SessionDetail, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastSessionID = sessionID
	return s.getResult, s.getErr
}

func (s *stubSessionService) UpdateStatus(_ context.Context, actorID int64, role string, sessionID int64, requestedStatus string) (*models.SessionDetail, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastSessionID = sessionID
	s.lastStatus = requestedStatus
	return s.updateStatusResult, s.updateStatusErr
}

func (s *stubSessionService) PayForSession(_ context.Context, actorID int64, role string, sessionID int64) (*models.SessionDetail, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastSessionID = sessionID
	return s.payResult, s.payErr
}

func TestBookSessionReturnsCreatedSession(t *testing.T) {
	service := &stubSessionService{
		bookResult: &models.SessionDetail{
			Session: models.Session{
				ID:              91,
				UserID:          42,
				CoachID:         7,
				Status:          "pending",
				DurationMinutes: 60,
			},
			Payment: &models.Payment{Status: "placeholder"},
		},
	}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Post("/api/v1/sessions/book", handler.BookSession)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/book", strings.NewReader(`{
		"coach_id": 7,
		"scheduled_at": "2026-03-15T09:00:00Z",
		"duration_minutes": 60,
		"notes": "focus on mobility"
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if service.lastActorID != 42 {
		t.Fatalf("expected actor id 42, got %d", service.lastActorID)
	}
	if service.lastBookInput.CoachID != 7 {
		t.Fatalf("expected coach id 7, got %d", service.lastBookInput.CoachID)
	}
	if service.lastBookInput.DurationMinutes != 60 {
		t.Fatalf("expected 60 minutes, got %d", service.lastBookInput.DurationMinutes)
	}
}

func TestBookSessionReturnsConflictForAvailabilityIssue(t *testing.T) {
	service := &stubSessionService{bookErr: services.ErrConflict}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Post("/api/v1/sessions/book", handler.BookSession)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/book", strings.NewReader(`{
		"coach_id": 7,
		"scheduled_at": "2026-03-15T09:00:00Z",
		"duration_minutes": 60
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestListSessionsPassesStatusAndTimeframe(t *testing.T) {
	service := &stubSessionService{
		listResult: []models.SessionDetail{{Session: models.Session{ID: 5, Status: "confirmed"}}},
	}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "9")
		return c.Next()
	})
	app.Get("/api/v1/sessions", handler.ListSessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?status=confirmed&timeframe=upcoming", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if service.lastRole != "coach" {
		t.Fatalf("expected coach role, got %q", service.lastRole)
	}
	if service.lastListFilter.Status != "confirmed" || service.lastListFilter.Timeframe != "upcoming" {
		t.Fatalf("unexpected filter: %+v", service.lastListFilter)
	}
}

func TestGetSessionReturnsNotFound(t *testing.T) {
	service := &stubSessionService{getErr: pgx.ErrNoRows}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Get("/api/v1/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateStatusReturnsUnprocessableForInvalidTransition(t *testing.T) {
	service := &stubSessionService{updateStatusErr: services.ErrInvalidStateTransition}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Put("/api/v1/sessions/:id/status", handler.UpdateStatus)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/sessions/55/status", strings.NewReader(`{"status":"complete"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	if service.lastStatus != "complete" {
		t.Fatalf("expected forwarded status, got %q", service.lastStatus)
	}
}

func TestPayForSessionReturnsConfirmedSession(t *testing.T) {
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	service := &stubSessionService{
		payResult: &models.SessionDetail{
			Session: models.Session{
				ID:              88,
				UserID:          42,
				CoachID:         7,
				ScheduledAt:     now,
				DurationMinutes: 45,
				Status:          "confirmed",
			},
			Payment: &models.Payment{
				ID:        11,
				SessionID: 88,
				Status:    "paid",
			},
		},
	}
	handler := &SessionHandler{service: service}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Post("/api/v1/sessions/:id/pay", handler.PayForSession)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/88/pay", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Session models.SessionDetail `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body.Session.Status != "confirmed" {
		t.Fatalf("expected confirmed status, got %q", body.Session.Status)
	}
	if body.Session.Payment == nil || body.Session.Payment.Status != "paid" {
		t.Fatalf("expected paid payment, got %+v", body.Session.Payment)
	}
}

func TestMapSessionErrorDefaultsToInternalServerError(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return mapSessionError(c, errors.New("boom"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestMapSessionErrorReturnsCoachNotFound(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return mapSessionError(c, services.ErrCoachNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
