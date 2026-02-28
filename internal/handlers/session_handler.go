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
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type SessionHandler struct {
	service sessionApplicationService
}

type sessionApplicationService interface {
	BookSession(ctx context.Context, userID int64, input services.BookSessionInput) (*models.SessionDetail, error)
	ListSessions(ctx context.Context, actorID int64, role string, filter repository.SessionListFilter) ([]models.SessionDetail, error)
	GetSession(ctx context.Context, actorID int64, role string, sessionID int64) (*models.SessionDetail, error)
	UpdateStatus(ctx context.Context, actorID int64, role string, sessionID int64, requestedStatus string) (*models.SessionDetail, error)
	PayForSession(ctx context.Context, actorID int64, role string, sessionID int64) (*models.SessionDetail, error)
}

func NewSessionHandler(service *services.SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

type bookSessionRequest struct {
	CoachID         int64   `json:"coach_id"`
	ScheduledAt     string  `json:"scheduled_at"`
	DurationMinutes int     `json:"duration_minutes"`
	Notes           *string `json:"notes"`
}

type updateSessionStatusRequest struct {
	Status string `json:"status"`
}

func (h *SessionHandler) BookSession(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req bookSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	scheduledAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ScheduledAt))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "scheduled_at must be a valid RFC3339 timestamp"})
	}
	if req.DurationMinutes <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "duration_minutes must be greater than 0"})
	}
	if req.Notes != nil && strings.TrimSpace(*req.Notes) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "notes must not be empty"})
	}

	detail, err := h.service.BookSession(c.Context(), userID, services.BookSessionInput{
		CoachID:         req.CoachID,
		ScheduledAt:     scheduledAt,
		DurationMinutes: req.DurationMinutes,
		Notes:           req.Notes,
	})
	if err != nil {
		return mapSessionError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"session": detail})
}

func (h *SessionHandler) ListSessions(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	timeframe := strings.TrimSpace(c.Query("timeframe"))
	if timeframe != "" && timeframe != "upcoming" && timeframe != "past" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "timeframe must be upcoming or past"})
	}

	sessions, err := h.service.ListSessions(c.Context(), userID, role, repository.SessionListFilter{
		Status:    strings.TrimSpace(c.Query("status")),
		Timeframe: timeframe,
	})
	if err != nil {
		return mapSessionError(c, err)
	}

	return c.JSON(fiber.Map{"sessions": sessions})
}

func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	sessionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid session id"})
	}

	session, err := h.service.GetSession(c.Context(), userID, role, sessionID)
	if err != nil {
		return mapSessionError(c, err)
	}

	return c.JSON(fiber.Map{"session": session})
}

func (h *SessionHandler) UpdateStatus(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	sessionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid session id"})
	}

	var req updateSessionStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	session, err := h.service.UpdateStatus(c.Context(), userID, role, sessionID, req.Status)
	if err != nil {
		return mapSessionError(c, err)
	}

	return c.JSON(fiber.Map{"session": session})
}

func (h *SessionHandler) PayForSession(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	sessionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid session id"})
	}

	session, err := h.service.PayForSession(c.Context(), userID, role, sessionID)
	if err != nil {
		return mapSessionError(c, err)
	}

	return c.JSON(fiber.Map{"session": session})
}

func mapSessionError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidInput), errors.Is(err, services.ErrInvalidStatus):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, services.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	case errors.Is(err, services.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Requested time conflicts with another session"})
	case errors.Is(err, services.ErrInvalidStateTransition):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, services.ErrCoachNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Coach not found"})
	case errors.Is(err, pgx.ErrNoRows):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Session not found"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process session request"})
	}
}
