package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type coachDiscoveryRepository interface {
	List(ctx context.Context, filter repository.CoachListFilter) ([]models.CoachProfile, int, error)
	GetByCoachID(ctx context.Context, coachID int64) (*models.CoachProfile, error)
	GetAvailableSlotsPreview(ctx context.Context, coachID int64, limit int) ([]string, error)
}

type userDiscoveryRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*models.UserProfile, error)
}

type coachMatchmaker interface {
	GetMatchedCoaches(ctx context.Context, userProfile *models.UserProfile, limit int) ([]models.CoachWithScore, error)
}

type CoachDiscoveryHandler struct {
	coachRepo          coachDiscoveryRepository
	userProfileRepo    userDiscoveryRepository
	matchmakingService coachMatchmaker
}

func NewCoachDiscoveryHandler(
	coachRepo coachDiscoveryRepository,
	userProfileRepo userDiscoveryRepository,
	matchmakingService coachMatchmaker,
) *CoachDiscoveryHandler {
	return &CoachDiscoveryHandler{
		coachRepo:          coachRepo,
		userProfileRepo:    userProfileRepo,
		matchmakingService: matchmakingService,
	}
}

func (h *CoachDiscoveryHandler) ListCoaches(c *fiber.Ctx) error {
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	minRating, err := parseNonNegativeFloat(c.Query("min_rating"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "min_rating must be a valid non-negative number"})
	}
	maxPrice, err := parseNonNegativeFloat(c.Query("max_price"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "max_price must be a valid non-negative number"})
	}
	experience, err := parseNonNegativeInt(c.Query("experience"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "experience must be a valid non-negative integer"})
	}

	coaches, total, err := h.coachRepo.List(c.Context(), repository.CoachListFilter{
		Specialization: strings.TrimSpace(c.Query("specialization")),
		MinRating:      minRating,
		MaxPrice:       maxPrice,
		Experience:     experience,
		Offset:         (page - 1) * limit,
		Limit:          limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch coaches"})
	}

	response := make([]models.CoachListResponse, 0, len(coaches))
	for _, coach := range coaches {
		response = append(response, buildCoachListResponse(coach, 0))
	}

	return c.JSON(fiber.Map{
		"coaches":    response,
		"pagination": buildPaginationMeta(page, limit, total),
	})
}

func (h *CoachDiscoveryHandler) GetRecommendedCoaches(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	limit := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	userProfile, err := h.userProfileRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user profile"})
	}

	coaches, err := h.matchmakingService.GetMatchedCoaches(c.Context(), userProfile, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch recommended coaches"})
	}

	response := make([]models.CoachListResponse, 0, len(coaches))
	for _, coach := range coaches {
		response = append(response, buildCoachListResponse(coach.CoachProfile, coach.MatchScore))
	}

	return c.JSON(fiber.Map{"coaches": response})
}

func (h *CoachDiscoveryHandler) GetCoachDetail(c *fiber.Ctx) error {
	coachID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || coachID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid coach id"})
	}

	coach, err := h.coachRepo.GetByCoachID(c.Context(), coachID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Coach not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch coach"})
	}

	slots, err := h.coachRepo.GetAvailableSlotsPreview(c.Context(), coachID, 3)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch coach availability"})
	}

	return c.JSON(fiber.Map{
		"coach": buildCoachDetailResponse(*coach, slots),
	})
}

func buildCoachListResponse(coach models.CoachProfile, matchScore int) models.CoachListResponse {
	response := models.CoachListResponse{
		ID:              strconv.FormatInt(coach.UserID, 10),
		FullName:        stringValue(coach.FullName),
		AvatarURL:       stringValue(coach.AvatarURL),
		Specializations: stringSliceValue(coach.Specializations),
		ExperienceYears: intValueResponse(coach.ExperienceYears),
		HourlyRate:      floatValueResponse(coach.HourlyRate),
		Rating:          floatValueResponse(coach.Rating),
		TotalReviews:    coach.TotalReviews,
	}
	if matchScore > 0 {
		response.MatchScore = matchScore
	}
	return response
}

func buildCoachDetailResponse(coach models.CoachProfile, slots []string) models.CoachDetailResponse {
	return models.CoachDetailResponse{
		CoachListResponse:  buildCoachListResponse(coach, 0),
		Bio:                stringValue(coach.Bio),
		Certifications:     stringSliceValue(coach.Certifications),
		IsVerified:         boolValue(coach.IsVerified),
		AvailableSlots:     slots,
		OnboardingComplete: coach.OnboardingComplete,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseNonNegativeInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, errInvalidNumber
	}
	return value, nil
}

func parseNonNegativeFloat(raw string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return 0, errInvalidNumber
	}
	return value, nil
}

var errInvalidNumber = errors.New("invalid number")

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringSliceValue(value *[]string) []string {
	if value == nil {
		return []string{}
	}
	return *value
}

func floatValueResponse(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func intValueResponse(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

var _ services.CoachMatcher = (*repository.CoachProfileRepository)(nil)
