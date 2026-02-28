package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

const maxProgramSizeBytes = 25 * 1024 * 1024

type programApplicationService interface {
	CreateProgram(
		ctx context.Context,
		coachID int64,
		input services.CreateProgramInput,
	) (*models.WorkoutProgram, error)
	ListPrograms(ctx context.Context, actorID int64, role string) ([]models.WorkoutProgram, error)
	GetProgram(
		ctx context.Context,
		actorID int64,
		role string,
		programID int64,
	) (*models.WorkoutProgram, error)
	GetDownloadURL(ctx context.Context, actorID int64, role string, programID int64) (string, error)
}

type workoutProgramResponse struct {
	ID          int64     `json:"id"`
	CoachID     int64     `json:"coach_id"`
	UserID      int64     `json:"user_id"`
	SessionID   int64     `json:"session_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProgramHandler struct {
	service programApplicationService
}

func NewProgramHandler(service programApplicationService) *ProgramHandler {
	return &ProgramHandler{service: service}
}

func (h *ProgramHandler) CreateProgram(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "coach" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	coachID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	userID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "user_id must be a positive integer"})
	}

	sessionID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("session_id")), 10, 64)
	if err != nil || sessionID <= 0 {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "session_id must be a positive integer"})
	}

	title := strings.TrimSpace(c.FormValue("title"))
	if title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	var description *string
	if rawDescription := c.FormValue("description"); rawDescription != "" {
		description = &rawDescription
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	if fileHeader.Size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is empty"})
	}
	if fileHeader.Size > maxProgramSizeBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file exceeds 25MB limit"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer file.Close()

	program, err := h.service.CreateProgram(c.Context(), coachID, services.CreateProgramInput{
		UserID:      userID,
		SessionID:   sessionID,
		Title:       title,
		Description: description,
		File:        file,
		Filename:    fileHeader.Filename,
	})
	if err != nil {
		return mapProgramError(c, err)
	}

	return c.Status(fiber.StatusCreated).
		JSON(fiber.Map{"program": newWorkoutProgramResponse(program)})
}

func (h *ProgramHandler) ListPrograms(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	actorID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	programs, err := h.service.ListPrograms(c.Context(), actorID, role)
	if err != nil {
		return mapProgramError(c, err)
	}

	return c.JSON(fiber.Map{"programs": newWorkoutProgramResponses(programs)})
}

func (h *ProgramHandler) GetProgram(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	actorID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	programID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || programID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid program id"})
	}

	program, err := h.service.GetProgram(c.Context(), actorID, role, programID)
	if err != nil {
		return mapProgramError(c, err)
	}

	return c.JSON(fiber.Map{"program": newWorkoutProgramResponse(program)})
}

func (h *ProgramHandler) DownloadProgram(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	actorID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	programID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || programID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid program id"})
	}

	signedURL, err := h.service.GetDownloadURL(c.Context(), actorID, role, programID)
	if err != nil {
		return mapProgramError(c, err)
	}

	return c.JSON(fiber.Map{"download_url": signedURL, "expires_in_seconds": 3600})
}

func mapProgramError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	case errors.Is(err, services.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	case errors.Is(err, services.ErrStorageUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).
			JSON(fiber.Map{"error": "Storage service is not configured"})
	case errors.Is(err, pgx.ErrNoRows):
		return c.Status(fiber.StatusNotFound).
			JSON(fiber.Map{"error": "Program or related resource not found"})
	default:
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to process program request"})
	}
}

func newWorkoutProgramResponse(program *models.WorkoutProgram) *workoutProgramResponse {
	if program == nil {
		return nil
	}
	return &workoutProgramResponse{
		ID:          program.ID,
		CoachID:     program.CoachID,
		UserID:      program.UserID,
		SessionID:   program.SessionID,
		Title:       program.Title,
		Description: program.Description,
		CreatedAt:   program.CreatedAt,
	}
}

func newWorkoutProgramResponses(programs []models.WorkoutProgram) []workoutProgramResponse {
	if len(programs) == 0 {
		return []workoutProgramResponse{}
	}
	responses := make([]workoutProgramResponse, 0, len(programs))
	for i := range programs {
		program := programs[i]
		responses = append(responses, workoutProgramResponse{
			ID:          program.ID,
			CoachID:     program.CoachID,
			UserID:      program.UserID,
			SessionID:   program.SessionID,
			Title:       program.Title,
			Description: program.Description,
			CreatedAt:   program.CreatedAt,
		})
	}
	return responses
}
