package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type stubProgramService struct {
	createResult    *models.WorkoutProgram
	createErr       error
	listResult      []models.WorkoutProgram
	listErr         error
	getResult       *models.WorkoutProgram
	getErr          error
	downloadURL     string
	downloadErr     error
	lastCoachID     int64
	lastActorID     int64
	lastRole        string
	lastProgramID   int64
	lastCreateInput services.CreateProgramInput
}

func (s *stubProgramService) CreateProgram(
	_ context.Context,
	coachID int64,
	input services.CreateProgramInput,
) (*models.WorkoutProgram, error) {
	s.lastCoachID = coachID
	s.lastCreateInput = input
	return s.createResult, s.createErr
}

func (s *stubProgramService) ListPrograms(
	_ context.Context,
	actorID int64,
	role string,
) ([]models.WorkoutProgram, error) {
	s.lastActorID = actorID
	s.lastRole = role
	return s.listResult, s.listErr
}

func (s *stubProgramService) GetProgram(
	_ context.Context,
	actorID int64,
	role string,
	programID int64,
) (*models.WorkoutProgram, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastProgramID = programID
	return s.getResult, s.getErr
}

func (s *stubProgramService) GetDownloadURL(
	_ context.Context,
	actorID int64,
	role string,
	programID int64,
) (string, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastProgramID = programID
	return s.downloadURL, s.downloadErr
}

func TestCreateProgramParsesMultipartRequest(t *testing.T) {
	service := &stubProgramService{
		createResult: &models.WorkoutProgram{
			ID:        17,
			CoachID:   7,
			UserID:    42,
			SessionID: 99,
			Title:     "Week 1",
			FileURL:   "https://public.storage/program.pdf",
		},
	}
	handler := NewProgramHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Post("/api/v1/programs", handler.CreateProgram)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("user_id", "42"); err != nil {
		t.Fatalf("WriteField user_id: %v", err)
	}
	if err := writer.WriteField("session_id", "99"); err != nil {
		t.Fatalf("WriteField session_id: %v", err)
	}
	if err := writer.WriteField("title", "Week 1"); err != nil {
		t.Fatalf("WriteField title: %v", err)
	}
	if err := writer.WriteField("description", "Strength block"); err != nil {
		t.Fatalf("WriteField description: %v", err)
	}
	part, err := writer.CreateFormFile("file", "program.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("pdf-content")); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/programs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if service.lastCoachID != 7 {
		t.Fatalf("expected coach id 7, got %d", service.lastCoachID)
	}
	if service.lastCreateInput.UserID != 42 || service.lastCreateInput.SessionID != 99 {
		t.Fatalf("unexpected ids: %+v", service.lastCreateInput)
	}
	if service.lastCreateInput.Title != "Week 1" {
		t.Fatalf("expected title Week 1, got %q", service.lastCreateInput.Title)
	}
	if service.lastCreateInput.Description == nil ||
		*service.lastCreateInput.Description != "Strength block" {
		t.Fatalf("unexpected description: %+v", service.lastCreateInput.Description)
	}
	if service.lastCreateInput.File == nil {
		t.Fatalf("expected uploaded file to be forwarded")
	}

	var payload struct {
		Program map[string]any `json:"program"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, ok := payload.Program["file_url"]; ok {
		t.Fatalf("expected file_url to be omitted")
	}
}

func TestListProgramsReturnsProgramsForRole(t *testing.T) {
	service := &stubProgramService{
		listResult: []models.WorkoutProgram{
			{ID: 3, Title: "Mobility", FileURL: "https://public.storage/program.pdf"},
		},
	}
	handler := NewProgramHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Get("/api/v1/programs", handler.ListPrograms)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if service.lastActorID != 42 || service.lastRole != "user" {
		t.Fatalf("unexpected actor forwarding: %d %q", service.lastActorID, service.lastRole)
	}

	var payload struct {
		Programs []map[string]any `json:"programs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(payload.Programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(payload.Programs))
	}
	if _, ok := payload.Programs[0]["file_url"]; ok {
		t.Fatalf("expected file_url to be omitted")
	}
}

func TestGetProgramReturnsNotFound(t *testing.T) {
	service := &stubProgramService{getErr: pgx.ErrNoRows}
	handler := NewProgramHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Get("/api/v1/programs/:id", handler.GetProgram)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetProgramOmitsFileURL(t *testing.T) {
	service := &stubProgramService{
		getResult: &models.WorkoutProgram{
			ID:        12,
			CoachID:   7,
			UserID:    42,
			SessionID: 99,
			Title:     "Week 1",
			FileURL:   "https://public.storage/program.pdf",
		},
	}
	handler := NewProgramHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Get("/api/v1/programs/:id", handler.GetProgram)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/12", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Program map[string]any `json:"program"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, ok := payload.Program["file_url"]; ok {
		t.Fatalf("expected file_url to be omitted")
	}
}

func TestDownloadProgramReturnsSignedURL(t *testing.T) {
	service := &stubProgramService{downloadURL: "https://signed.example/file"}
	handler := NewProgramHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Get("/api/v1/programs/:id/download", handler.DownloadProgram)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/12/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if payload.DownloadURL != "https://signed.example/file" {
		t.Fatalf("unexpected download url: %q", payload.DownloadURL)
	}
}
