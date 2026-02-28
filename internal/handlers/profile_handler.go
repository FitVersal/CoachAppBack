package handlers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

const maxAvatarSizeBytes = 5 * 1024 * 1024

type ProfileHandler struct {
	profileService   *services.ProfileService
	userProfileRepo  userProfileStore
	coachProfileRepo coachProfileStore
	storageService   services.StorageService
}

type userProfileStore interface {
	GetByUserID(ctx context.Context, userID int64) (*models.UserProfile, error)
}

type coachProfileStore interface {
	GetByUserID(ctx context.Context, userID int64) (*models.CoachProfile, error)
}

func NewProfileHandler(
	profileService *services.ProfileService,
	userProfileRepo userProfileStore,
	coachProfileRepo coachProfileStore,
	storageService services.StorageService,
) *ProfileHandler {
	return &ProfileHandler{
		profileService:   profileService,
		userProfileRepo:  userProfileRepo,
		coachProfileRepo: coachProfileRepo,
		storageService:   storageService,
	}
}

type updateUserProfileRequest struct {
	FullName          *string   `json:"full_name"`
	Age               *int      `json:"age"`
	Gender            *string   `json:"gender"`
	HeightCM          *float64  `json:"height_cm"`
	WeightKG          *float64  `json:"weight_kg"`
	FitnessLevel      *string   `json:"fitness_level"`
	Goals             *[]string `json:"goals"`
	MaxHourlyRate     *float64  `json:"max_hourly_rate"`
	MedicalConditions *string   `json:"medical_conditions"`
}

type updateCoachProfileRequest struct {
	FullName        *string   `json:"full_name"`
	Bio             *string   `json:"bio"`
	Specializations *[]string `json:"specializations"`
	Certifications  *[]string `json:"certifications"`
	ExperienceYears *int      `json:"experience_years"`
	HourlyRate      *float64  `json:"hourly_rate"`
}

func (h *ProfileHandler) UpdateUserProfile(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req updateUserProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if validationErr := validateUserProfileUpdateRequest(req); validationErr != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validationErr})
	}

	profile, err := h.profileService.UpdateUserProfile(c.Context(), userID, repository.UpdateUserProfileInput{
		FullName:          req.FullName,
		Age:               req.Age,
		Gender:            req.Gender,
		HeightCM:          req.HeightCM,
		WeightKG:          req.WeightKG,
		FitnessLevel:      req.FitnessLevel,
		Goals:             req.Goals,
		MaxHourlyRate:     req.MaxHourlyRate,
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

func (h *ProfileHandler) UpdateCoachProfile(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "coach" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req updateCoachProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if validationErr := validateCoachProfileUpdateRequest(req); validationErr != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validationErr})
	}

	profile, err := h.profileService.UpdateCoachProfile(c.Context(), userID, repository.UpdateCoachProfileInput{
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

func (h *ProfileHandler) GetUserProfile(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	profile, err := h.userProfileRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
	}

	return c.JSON(fiber.Map{
		"profile":             profile,
		"onboarding_complete": profile.OnboardingComplete,
	})
}

func (h *ProfileHandler) GetCoachProfile(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "coach" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	profile, err := h.coachProfileRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
	}

	return c.JSON(fiber.Map{
		"profile":             profile,
		"onboarding_complete": profile.OnboardingComplete,
	})
}

func (h *ProfileHandler) UploadUserAvatar(c *fiber.Ctx) error {
	return h.uploadAvatar(c, "user")
}

func (h *ProfileHandler) UploadCoachAvatar(c *fiber.Ctx) error {
	return h.uploadAvatar(c, "coach")
}

func (h *ProfileHandler) uploadAvatar(c *fiber.Ctx, expectedRole string) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != expectedRole {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	if h.storageService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Storage service is not configured"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file is required"})
	}
	if fileHeader.Size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file is empty"})
	}
	if fileHeader.Size > maxAvatarSizeBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file exceeds 5MB limit"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open avatar file"})
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar must be a jpg, jpeg, png, or webp file"})
	}

	filename := fmt.Sprintf("%d-%d%s", userID, time.Now().UnixNano(), ext)
	folder := expectedRole + "s/avatars"
	avatarURL, err := h.storageService.UploadFile(c.Context(), file, filename, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to upload avatar"})
	}

	var profile any
	if expectedRole == "user" {
		currentProfile, err := h.userProfileRepo.GetByUserID(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
		}
		if currentProfile.AvatarURL != nil && *currentProfile.AvatarURL != "" && *currentProfile.AvatarURL != avatarURL {
			_ = h.storageService.DeleteFile(c.Context(), *currentProfile.AvatarURL)
		}
		profile, err = h.profileService.UpdateUserProfile(c.Context(), userID, repository.UpdateUserProfileInput{
			AvatarURL: &avatarURL,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
		}
	} else {
		currentProfile, err := h.coachProfileRepo.GetByUserID(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
		}
		if currentProfile.AvatarURL != nil && *currentProfile.AvatarURL != "" && *currentProfile.AvatarURL != avatarURL {
			_ = h.storageService.DeleteFile(c.Context(), *currentProfile.AvatarURL)
		}
		profile, err = h.profileService.UpdateCoachProfile(c.Context(), userID, repository.UpdateCoachProfileInput{
			AvatarURL: &avatarURL,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
		}
	}

	return c.JSON(fiber.Map{
		"avatar_url": avatarURL,
		"profile":    profile,
	})
}

func parseProfileUserID(c *fiber.Ctx) (int64, error) {
	userIDValue := c.Locals("user_id")
	userIDStr, ok := userIDValue.(string)
	if !ok {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(userIDStr, 10, 64)
}
