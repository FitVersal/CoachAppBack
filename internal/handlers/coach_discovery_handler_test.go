package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

type stubCoachDiscoveryRepo struct {
	coaches       []models.CoachProfile
	total         int
	listFilter    repository.CoachListFilter
	detailCoach   *models.CoachProfile
	detailCoachID int64
	detailErr     error
	slots         []string
}

func (s *stubCoachDiscoveryRepo) List(_ context.Context, filter repository.CoachListFilter) ([]models.CoachProfile, int, error) {
	s.listFilter = filter
	return s.coaches, s.total, nil
}

func (s *stubCoachDiscoveryRepo) GetByCoachID(_ context.Context, coachID int64) (*models.CoachProfile, error) {
	s.detailCoachID = coachID
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	return s.detailCoach, nil
}

func (s *stubCoachDiscoveryRepo) GetAvailableSlotsPreview(_ context.Context, _ int64, _ int) ([]string, error) {
	return s.slots, nil
}

type stubUserDiscoveryRepo struct {
	profile *models.UserProfile
	err     error
}

func (s *stubUserDiscoveryRepo) GetByUserID(_ context.Context, _ int64) (*models.UserProfile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.profile, nil
}

type stubCoachMatchmaker struct {
	coaches []models.CoachWithScore
	limit   int
}

func (s *stubCoachMatchmaker) GetMatchedCoaches(_ context.Context, _ *models.UserProfile, limit int) ([]models.CoachWithScore, error) {
	s.limit = limit
	return s.coaches, nil
}

func TestListCoachesReturnsPaginationAndFilters(t *testing.T) {
	fullName := "Coach Ana"
	specializations := []string{"weight_loss"}
	rating := 4.7
	experience := 6
	hourlyRate := 55.0

	coachRepo := &stubCoachDiscoveryRepo{
		coaches: []models.CoachProfile{{
			UserID:             91,
			FullName:           &fullName,
			Specializations:    &specializations,
			Rating:             &rating,
			TotalReviews:       12,
			ExperienceYears:    &experience,
			HourlyRate:         &hourlyRate,
			OnboardingComplete: true,
		}},
		total: 11,
	}
	handler := NewCoachDiscoveryHandler(coachRepo, &stubUserDiscoveryRepo{}, &stubCoachMatchmaker{})

	app := fiber.New()
	app.Get("/api/v1/coaches", handler.ListCoaches)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/coaches?specialization=weight_loss&min_rating=4.5&max_price=60&experience=3&page=2&limit=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Coaches    []models.CoachListResponse `json:"coaches"`
		Pagination models.PaginationMeta      `json:"pagination"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if coachRepo.listFilter.Specialization != "weight_loss" || coachRepo.listFilter.Offset != 5 || coachRepo.listFilter.Limit != 5 {
		t.Fatalf("unexpected filter: %+v", coachRepo.listFilter)
	}
	if len(body.Coaches) != 1 || body.Coaches[0].ID != "91" {
		t.Fatalf("unexpected coaches response: %+v", body.Coaches)
	}
	if body.Coaches[0].TotalReviews != 12 {
		t.Fatalf("expected total_reviews 12, got %d", body.Coaches[0].TotalReviews)
	}
	if body.Pagination.Total != 11 || body.Pagination.TotalPages != 3 || body.Pagination.Page != 2 {
		t.Fatalf("unexpected pagination: %+v", body.Pagination)
	}
}

func TestGetRecommendedCoachesReturnsMatchScores(t *testing.T) {
	goals := []string{"weight_loss"}
	userRepo := &stubUserDiscoveryRepo{profile: &models.UserProfile{Goals: &goals}}
	matchmaker := &stubCoachMatchmaker{
		coaches: []models.CoachWithScore{
			{
				CoachProfile: models.CoachProfile{
					UserID:             44,
					Specializations:    &goals,
					OnboardingComplete: true,
				},
				MatchScore: 85,
			},
		},
	}
	handler := NewCoachDiscoveryHandler(&stubCoachDiscoveryRepo{}, userRepo, matchmaker)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Get("/api/v1/coaches/recommended", handler.GetRecommendedCoaches)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/coaches/recommended?limit=3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Coaches []models.CoachListResponse `json:"coaches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if matchmaker.limit != 3 {
		t.Fatalf("expected limit 3, got %d", matchmaker.limit)
	}
	if len(body.Coaches) != 1 || body.Coaches[0].MatchScore != 85 {
		t.Fatalf("unexpected recommended coaches: %+v", body.Coaches)
	}
}

func TestGetCoachDetailReturnsCoachProfile(t *testing.T) {
	fullName := "Coach Detail"
	bio := "Precision nutrition coach"
	certs := []string{"NASM"}
	verified := true

	coachRepo := &stubCoachDiscoveryRepo{
		slots: []string{"2026-03-01T10:00:00Z", "2026-03-01T11:00:00Z"},
		detailCoach: &models.CoachProfile{
			UserID:             55,
			FullName:           &fullName,
			Bio:                &bio,
			Certifications:     &certs,
			IsVerified:         &verified,
			TotalReviews:       3,
			OnboardingComplete: true,
		},
	}
	handler := NewCoachDiscoveryHandler(coachRepo, &stubUserDiscoveryRepo{}, &stubCoachMatchmaker{})

	app := fiber.New()
	app.Get("/api/v1/coaches/:id", handler.GetCoachDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/coaches/55", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Coach models.CoachDetailResponse `json:"coach"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if coachRepo.detailCoachID != 55 {
		t.Fatalf("expected detail lookup for coach 55, got %d", coachRepo.detailCoachID)
	}
	if body.Coach.ID != "55" || body.Coach.Bio != bio || !body.Coach.IsVerified {
		t.Fatalf("unexpected coach detail: %+v", body.Coach)
	}
	if len(body.Coach.AvailableSlots) != 2 || body.Coach.TotalReviews != 3 {
		t.Fatalf("unexpected coach detail preview fields: %+v", body.Coach)
	}
}

func TestGetCoachDetailReturnsNotFound(t *testing.T) {
	handler := NewCoachDiscoveryHandler(&stubCoachDiscoveryRepo{detailErr: pgx.ErrNoRows}, &stubUserDiscoveryRepo{}, &stubCoachMatchmaker{})

	app := fiber.New()
	app.Get("/api/v1/coaches/:id", handler.GetCoachDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/coaches/99", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
