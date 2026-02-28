package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"

	websocket "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/services"
	chatws "github.com/saeid-a/CoachAppBack/internal/websocket"
	"github.com/saeid-a/CoachAppBack/pkg/utils"
)

type chatApplicationService interface {
	ListConversations(ctx context.Context, actorID int64, role string) ([]models.ConversationSummary, error)
	CreateConversation(ctx context.Context, actorID int64, role string, coachID int64) (*models.Conversation, error)
	ListMessages(ctx context.Context, actorID int64, role string, conversationID int64, page int, limit int) ([]models.ChatMessage, int, error)
	SendMessage(ctx context.Context, actorID int64, role string, conversationID int64, content string) (*services.ChatDelivery, error)
}

type ChatHandler struct {
	service   chatApplicationService
	hub       *chatws.Hub
	jwtSecret string
}

type createConversationRequest struct {
	CoachID int64 `json:"coach_id"`
}

func NewChatHandler(service chatApplicationService, hub *chatws.Hub, jwtSecret string) *ChatHandler {
	return &ChatHandler{
		service:   service,
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

func (h *ChatHandler) ListConversations(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	conversations, err := h.service.ListConversations(c.Context(), userID, role)
	if err != nil {
		return mapChatError(c, err)
	}

	return c.JSON(fiber.Map{"conversations": conversations})
}

func (h *ChatHandler) CreateConversation(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	var req createConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	conversation, err := h.service.CreateConversation(c.Context(), userID, role, req.CoachID)
	if err != nil {
		return mapChatError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"conversation": conversation})
}

func (h *ChatHandler) GetMessages(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != "user" && role != "coach") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	userID, err := parseProfileUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid conversation id"})
	}

	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	messages, total, err := h.service.ListMessages(c.Context(), userID, role, conversationID, page, limit)
	if err != nil {
		return mapChatError(c, err)
	}

	return c.JSON(fiber.Map{
		"messages":   messages,
		"pagination": buildPaginationMeta(page, limit, total),
	})
}

func (h *ChatHandler) WebSocketAuth(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{"error": "WebSocket upgrade required"})
	}

	claims, err := h.parseWSClaims(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("role", claims.Role)
	return c.Next()
}

func (h *ChatHandler) HandleWebSocket(conn *websocket.Conn) {
	userID, _ := conn.Locals("user_id").(string)
	role, _ := conn.Locals("role").(string)
	client := chatws.NewClient(h.hub, conn, userID)

	h.hub.Register(client)
	go client.WritePump()
	client.ReadPump(h.service, role)
}

func (h *ChatHandler) parseWSClaims(c *fiber.Ctx) (*utils.Claims, error) {
	tokenString := strings.TrimSpace(c.Query("token"))
	if tokenString == "" {
		authHeader := strings.TrimSpace(c.Get("Authorization"))
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
	}

	if tokenString == "" {
		return nil, errors.New("missing token")
	}

	return utils.ValidateToken(tokenString, h.jwtSecret)
}

func mapChatError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	case errors.Is(err, services.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	case errors.Is(err, services.ErrCoachNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Coach not found"})
	case errors.Is(err, pgx.ErrNoRows):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process chat request"})
	}
}
