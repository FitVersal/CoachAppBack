package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

type userOnboardingProfileStore interface {
	UpdateOnboarding(ctx context.Context, userID int64, req repository.UserOnboardingInput) (*models.UserProfile, error)
}

type coachOnboardingProfileStore interface {
	UpdateOnboarding(ctx context.Context, userID int64, req repository.CoachOnboardingInput) (*models.CoachProfile, error)
}

type OnboardingHandler struct {
	userProfileRepo  userOnboardingProfileStore
	coachProfileRepo coachOnboardingProfileStore
}

func NewOnboardingHandler(userProfileRepo userOnboardingProfileStore, coachProfileRepo coachOnboardingProfileStore) *OnboardingHandler {
	return &OnboardingHandler{
		userProfileRepo:  userProfileRepo,
		coachProfileRepo: coachProfileRepo,
	}
}

type userOnboardingRequest struct {
	FullName          string   `json:"full_name"`
	Age               int      `json:"age"`
	Gender            string   `json:"gender"`
	HeightCM          float64  `json:"height_cm"`
	WeightKG          float64  `json:"weight_kg"`
	FitnessLevel      string   `json:"fitness_level"`
	Goals             []string `json:"goals"`
	MedicalConditions string   `json:"medical_conditions"`
}

type coachOnboardingRequest struct {
	FullName        string   `json:"full_name"`
	Bio             string   `json:"bio"`
	Specializations []string `json:"specializations"`
	Certifications  []string `json:"certifications"`
	ExperienceYears int      `json:"experience_years"`
	HourlyRate      float64  `json:"hourly_rate"`
}

func (h *OnboardingHandler) UserOnboarding(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req userOnboardingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if validationErr := validateUserOnboardingRequest(req); validationErr != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validationErr})
	}

	profile, err := h.userProfileRepo.UpdateOnboarding(c.Context(), userID, repository.UserOnboardingInput{
		FullName:          req.FullName,
		Age:               req.Age,
		Gender:            req.Gender,
		HeightCM:          req.HeightCM,
		WeightKG:          req.WeightKG,
		FitnessLevel:      req.FitnessLevel,
		Goals:             req.Goals,
		MedicalConditions: req.MedicalConditions,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	return c.JSON(fiber.Map{
		"profile":             profile,
		"onboarding_complete": profile.OnboardingComplete,
	})
}

func (h *OnboardingHandler) CoachOnboarding(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "coach" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req coachOnboardingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if validationErr := validateCoachOnboardingRequest(req); validationErr != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validationErr})
	}

	profile, err := h.coachProfileRepo.UpdateOnboarding(c.Context(), userID, repository.CoachOnboardingInput{
		FullName:        req.FullName,
		Bio:             req.Bio,
		Specializations: req.Specializations,
		Certifications:  req.Certifications,
		ExperienceYears: req.ExperienceYears,
		HourlyRate:      req.HourlyRate,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	return c.JSON(fiber.Map{
		"profile":             profile,
		"onboarding_complete": profile.OnboardingComplete,
	})
}

func parseUserID(c *fiber.Ctx) (int64, error) {
	userIDValue := c.Locals("user_id")
	userIDStr, ok := userIDValue.(string)
	if !ok {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(userIDStr, 10, 64)
}
