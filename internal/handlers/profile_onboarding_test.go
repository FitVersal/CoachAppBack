package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type stubUserProfileRepo struct {
	profile             *models.UserProfile
	lastOnboardingInput repository.UserOnboardingInput
	lastUpdatePartial   repository.UpdateUserProfileInput
}

func (s *stubUserProfileRepo) GetByUserID(_ context.Context, _ int64) (*models.UserProfile, error) {
	return s.profile, nil
}

func (s *stubUserProfileRepo) UpdateOnboarding(_ context.Context, _ int64, req repository.UserOnboardingInput) (*models.UserProfile, error) {
	s.lastOnboardingInput = req
	if s.profile == nil {
		s.profile = &models.UserProfile{}
	}
	s.profile.FullName = &req.FullName
	s.profile.Age = &req.Age
	s.profile.Gender = &req.Gender
	s.profile.HeightCM = &req.HeightCM
	s.profile.WeightKG = &req.WeightKG
	s.profile.FitnessLevel = &req.FitnessLevel
	s.profile.Goals = &req.Goals
	s.profile.MaxHourlyRate = req.MaxHourlyRate
	s.profile.MedicalConditions = &req.MedicalConditions
	s.profile.OnboardingComplete = true
	return s.profile, nil
}

func (s *stubUserProfileRepo) UpdatePartial(_ context.Context, _ int64, req repository.UpdateUserProfileInput) (*models.UserProfile, error) {
	s.lastUpdatePartial = req
	if s.profile == nil {
		s.profile = &models.UserProfile{}
	}
	if req.AvatarURL != nil {
		s.profile.AvatarURL = req.AvatarURL
	}
	if req.MaxHourlyRate != nil {
		s.profile.MaxHourlyRate = req.MaxHourlyRate
	}
	if req.MedicalConditions != nil {
		s.profile.MedicalConditions = req.MedicalConditions
	}
	return s.profile, nil
}

type stubCoachProfileRepo struct {
	profile             *models.CoachProfile
	lastOnboardingInput repository.CoachOnboardingInput
	lastUpdatePartial   repository.UpdateCoachProfileInput
}

func (s *stubCoachProfileRepo) GetByUserID(_ context.Context, _ int64) (*models.CoachProfile, error) {
	return s.profile, nil
}

func (s *stubCoachProfileRepo) UpdateOnboarding(_ context.Context, _ int64, req repository.CoachOnboardingInput) (*models.CoachProfile, error) {
	s.lastOnboardingInput = req
	if s.profile == nil {
		s.profile = &models.CoachProfile{}
	}
	s.profile.FullName = &req.FullName
	s.profile.Bio = &req.Bio
	s.profile.Specializations = &req.Specializations
	s.profile.Certifications = &req.Certifications
	s.profile.ExperienceYears = &req.ExperienceYears
	s.profile.HourlyRate = &req.HourlyRate
	s.profile.OnboardingComplete = true
	return s.profile, nil
}

func (s *stubCoachProfileRepo) UpdatePartial(_ context.Context, _ int64, req repository.UpdateCoachProfileInput) (*models.CoachProfile, error) {
	s.lastUpdatePartial = req
	if s.profile == nil {
		s.profile = &models.CoachProfile{}
	}
	if req.AvatarURL != nil {
		s.profile.AvatarURL = req.AvatarURL
	}
	if req.Certifications != nil {
		s.profile.Certifications = req.Certifications
	}
	return s.profile, nil
}

type stubStorageService struct {
	uploadedFolder   string
	uploadedFilename string
	uploadedContent  []byte
	uploadedURL      string
	deletedURL       string
}

func (s *stubStorageService) UploadFile(_ context.Context, file multipart.File, filename string, folder string) (string, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	s.uploadedFilename = filename
	s.uploadedFolder = folder
	s.uploadedContent = content
	if s.uploadedURL == "" {
		s.uploadedURL = "https://storage.example/avatar.png"
	}
	return s.uploadedURL, nil
}

func (s *stubStorageService) DeleteFile(_ context.Context, fileURL string) error {
	s.deletedURL = fileURL
	return nil
}

func (s *stubStorageService) GetSignedURL(_ context.Context, fileURL string) (string, error) {
	return fileURL, nil
}

func TestUserOnboardingUsesMedicalConditionsContract(t *testing.T) {
	userRepo := &stubUserProfileRepo{profile: &models.UserProfile{}}
	coachRepo := &stubCoachProfileRepo{}
	handler := NewOnboardingHandler(userRepo, coachRepo)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Post("/api/v1/users/onboarding", handler.UserOnboarding)

	body := `{"full_name":"Sam User","age":29,"gender":"male","height_cm":180,"weight_kg":78,"fitness_level":"beginner","goals":["weight_loss"],"medical_conditions":"asthma"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/onboarding", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := userRepo.lastOnboardingInput.MedicalConditions; got != "asthma" {
		t.Fatalf("expected medical_conditions to be forwarded, got %q", got)
	}
}

func TestUserProfileUpdateUsesMaxHourlyRatePreference(t *testing.T) {
	userRepo := &stubUserProfileRepo{profile: &models.UserProfile{}}
	coachRepo := &stubCoachProfileRepo{}
	profileService := services.NewProfileService(userRepo, coachRepo)
	handler := NewProfileHandler(profileService, userRepo, coachRepo, nil)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Put("/api/v1/users/profile", handler.UpdateUserProfile)

	body := `{"max_hourly_rate":65}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if userRepo.lastUpdatePartial.MaxHourlyRate == nil || *userRepo.lastUpdatePartial.MaxHourlyRate != 65 {
		t.Fatalf("expected max_hourly_rate 65, got %+v", userRepo.lastUpdatePartial.MaxHourlyRate)
	}
}

func TestCoachProfileUpdateUsesCertificationsArray(t *testing.T) {
	userRepo := &stubUserProfileRepo{}
	coachRepo := &stubCoachProfileRepo{profile: &models.CoachProfile{}}
	profileService := services.NewProfileService(userRepo, coachRepo)
	handler := NewProfileHandler(profileService, userRepo, coachRepo, nil)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "77")
		return c.Next()
	})
	app.Put("/api/v1/coaches/profile", handler.UpdateCoachProfile)

	body := `{"certifications":["NASM","ACE"],"hourly_rate":55}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/coaches/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if coachRepo.lastUpdatePartial.Certifications == nil {
		t.Fatal("expected certifications to be forwarded")
	}
	if got := len(*coachRepo.lastUpdatePartial.Certifications); got != 2 {
		t.Fatalf("expected 2 certifications, got %d", got)
	}
}

func TestUserAvatarUploadUpdatesAvatarURL(t *testing.T) {
	oldURL := "https://storage.example/old.png"
	userRepo := &stubUserProfileRepo{
		profile: &models.UserProfile{
			AvatarURL: &oldURL,
		},
	}
	coachRepo := &stubCoachProfileRepo{}
	storage := &stubStorageService{
		uploadedURL: "https://storage.example/new.png",
	}
	profileService := services.NewProfileService(userRepo, coachRepo)
	handler := NewProfileHandler(profileService, userRepo, coachRepo, storage)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "15")
		return c.Next()
	})
	app.Post("/api/v1/users/profile/avatar", handler.UploadUserAvatar)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("png-bytes")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/profile/avatar", &requestBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if storage.uploadedFolder != "users/avatars" {
		t.Fatalf("expected users/avatars folder, got %q", storage.uploadedFolder)
	}
	if storage.deletedURL != oldURL {
		t.Fatalf("expected previous avatar to be deleted, got %q", storage.deletedURL)
	}
	if userRepo.lastUpdatePartial.AvatarURL == nil || *userRepo.lastUpdatePartial.AvatarURL != storage.uploadedURL {
		t.Fatal("expected avatar_url update to be persisted")
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["avatar_url"] != storage.uploadedURL {
		t.Fatalf("expected avatar_url %q, got %#v", storage.uploadedURL, payload["avatar_url"])
	}
}
