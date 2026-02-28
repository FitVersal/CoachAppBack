package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

type ChatService struct {
	db               *pgxpool.Pool
	conversationRepo *repository.ConversationRepository
	messageRepo      *repository.MessageRepository
	userRepo         userReader
}

type ChatDelivery struct {
	Conversation *models.Conversation
	Message      *models.ChatMessage
	RecipientID  int64
}

func NewChatService(
	db *pgxpool.Pool,
	conversationRepo *repository.ConversationRepository,
	messageRepo *repository.MessageRepository,
	userRepo userReader,
) *ChatService {
	return &ChatService{
		db:               db,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		userRepo:         userRepo,
	}
}

func (s *ChatService) ListConversations(
	ctx context.Context,
	actorID int64,
	role string,
) ([]models.ConversationSummary, error) {
	if role != "user" && role != "coach" {
		return nil, ErrForbidden
	}

	return s.conversationRepo.ListForParticipant(ctx, actorID)
}

func (s *ChatService) CreateConversation(
	ctx context.Context,
	actorID int64,
	role string,
	coachID int64,
) (*models.Conversation, error) {
	if role != "user" {
		return nil, ErrForbidden
	}
	if coachID <= 0 || coachID == actorID {
		return nil, ErrInvalidInput
	}

	coach, err := s.userRepo.GetByID(ctx, coachID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoachNotFound
		}
		return nil, err
	}
	if coach.Role != "coach" {
		return nil, ErrInvalidInput
	}

	return s.conversationRepo.CreateOrGet(ctx, actorID, coachID)
}

func (s *ChatService) ListMessages(
	ctx context.Context,
	actorID int64,
	role string,
	conversationID int64,
	page int,
	limit int,
) ([]models.ChatMessage, int, error) {
	if role != "user" && role != "coach" {
		return nil, 0, ErrForbidden
	}
	if conversationID <= 0 || page <= 0 || limit <= 0 {
		return nil, 0, ErrInvalidInput
	}

	_, err := s.conversationRepo.GetByIDForParticipant(ctx, conversationID, actorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, pgx.ErrNoRows
		}
		return nil, 0, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txMessageRepo := repository.NewMessageRepository(tx)

	messages, total, err := txMessageRepo.ListByConversation(
		ctx,
		conversationID,
		limit,
		(page-1)*limit,
	)
	if err != nil {
		return nil, 0, err
	}

	messageIDs := make([]int64, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
	}

	if err := txMessageRepo.MarkMessagesRead(ctx, messageIDs, actorID); err != nil {
		return nil, 0, err
	}

	for i := range messages {
		if messages[i].SenderID != actorID {
			messages[i].IsRead = true
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (s *ChatService) SendMessage(
	ctx context.Context,
	actorID int64,
	role string,
	conversationID int64,
	content string,
) (*ChatDelivery, error) {
	if role != "user" && role != "coach" {
		return nil, ErrForbidden
	}
	if conversationID <= 0 {
		return nil, ErrInvalidInput
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, ErrInvalidInput
	}

	conversation, err := s.conversationRepo.GetByIDForParticipant(ctx, conversationID, actorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	recipientID := conversation.UserID
	if actorID == conversation.UserID {
		recipientID = conversation.CoachID
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txMessageRepo := repository.NewMessageRepository(tx)
	txConversationRepo := repository.NewConversationRepository(tx)

	message, err := txMessageRepo.Create(ctx, conversationID, actorID, trimmed)
	if err != nil {
		return nil, err
	}

	if err := txConversationRepo.Touch(ctx, conversationID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ChatDelivery{
		Conversation: conversation,
		Message:      message,
		RecipientID:  recipientID,
	}, nil
}

func FormatChatTimestamp(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}
