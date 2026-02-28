package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/services"
	chatws "github.com/saeid-a/CoachAppBack/internal/websocket"
)

type stubChatService struct {
	conversationsResult []models.ConversationSummary
	conversationsErr    error
	createResult        *models.Conversation
	createErr           error
	messagesResult      []models.ChatMessage
	messagesTotal       int
	messagesErr         error
	lastActorID         int64
	lastRole            string
	lastCoachID         int64
	lastConversationID  int64
	lastPage            int
	lastLimit           int
}

func (s *stubChatService) ListConversations(_ context.Context, actorID int64, role string) ([]models.ConversationSummary, error) {
	s.lastActorID = actorID
	s.lastRole = role
	return s.conversationsResult, s.conversationsErr
}

func (s *stubChatService) CreateConversation(_ context.Context, actorID int64, role string, coachID int64) (*models.Conversation, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastCoachID = coachID
	return s.createResult, s.createErr
}

func (s *stubChatService) ListMessages(_ context.Context, actorID int64, role string, conversationID int64, page int, limit int) ([]models.ChatMessage, int, error) {
	s.lastActorID = actorID
	s.lastRole = role
	s.lastConversationID = conversationID
	s.lastPage = page
	s.lastLimit = limit
	return s.messagesResult, s.messagesTotal, s.messagesErr
}

func (s *stubChatService) SendMessage(_ context.Context, _ int64, _ string, _ int64, _ string) (*services.ChatDelivery, error) {
	return nil, nil
}

func TestListConversationsReturnsConversationSummaries(t *testing.T) {
	service := &stubChatService{
		conversationsResult: []models.ConversationSummary{
			{
				Conversation: models.Conversation{ID: 17, UserID: 42, CoachID: 8},
				LastMessage: &models.ChatMessage{
					ID:             3,
					ConversationID: 17,
					SenderID:       8,
					Content:        "See you tomorrow",
					CreatedAt:      time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
				},
				UnreadCount: 2,
			},
		},
	}
	handler := NewChatHandler(service, chatws.NewHub(), "secret")

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Get("/api/v1/conversations", handler.ListConversations)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if service.lastActorID != 42 || service.lastRole != "user" {
		t.Fatalf("unexpected actor context: %d %q", service.lastActorID, service.lastRole)
	}

	var body struct {
		Conversations []models.ConversationSummary `json:"conversations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(body.Conversations) != 1 || body.Conversations[0].UnreadCount != 2 {
		t.Fatalf("unexpected response: %+v", body.Conversations)
	}
}

func TestCreateConversationReturnsCreatedConversation(t *testing.T) {
	service := &stubChatService{
		createResult: &models.Conversation{ID: 9, UserID: 42, CoachID: 7},
	}
	handler := NewChatHandler(service, chatws.NewHub(), "secret")

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		c.Locals("user_id", "42")
		return c.Next()
	})
	app.Post("/api/v1/conversations", handler.CreateConversation)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", strings.NewReader(`{"coach_id":7}`))
	req.Header.Set("Content-Type", "application/json")

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
}

func TestGetMessagesReturnsPagination(t *testing.T) {
	service := &stubChatService{
		messagesResult: []models.ChatMessage{
			{ID: 5, ConversationID: 11, SenderID: 7, Content: "Hi", CreatedAt: time.Now().UTC()},
		},
		messagesTotal: 12,
	}
	handler := NewChatHandler(service, chatws.NewHub(), "secret")

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Get("/api/v1/conversations/:id/messages", handler.GetMessages)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/11/messages?page=2&limit=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if service.lastConversationID != 11 || service.lastPage != 2 || service.lastLimit != 5 {
		t.Fatalf("unexpected forwarded pagination: conversation=%d page=%d limit=%d", service.lastConversationID, service.lastPage, service.lastLimit)
	}

	var body struct {
		Messages   []models.ChatMessage  `json:"messages"`
		Pagination models.PaginationMeta `json:"pagination"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(body.Messages) != 1 || body.Pagination.Total != 12 || body.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected response body: %+v %+v", body.Messages, body.Pagination)
	}
}

func TestGetMessagesReturnsNotFound(t *testing.T) {
	service := &stubChatService{messagesErr: pgx.ErrNoRows}
	handler := NewChatHandler(service, chatws.NewHub(), "secret")

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "coach")
		c.Locals("user_id", "7")
		return c.Next()
	})
	app.Get("/api/v1/conversations/:id/messages", handler.GetMessages)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/99/messages", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
