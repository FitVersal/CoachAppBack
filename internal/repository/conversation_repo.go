package repository

import (
	"context"
	"database/sql"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type ConversationRepository struct {
	db DBTX
}

func NewConversationRepository(db DBTX) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) CreateOrGet(
	ctx context.Context,
	userID int64,
	coachID int64,
) (*models.Conversation, error) {
	query := `
		INSERT INTO conversations (user_id, coach_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, coach_id)
		DO UPDATE SET updated_at = conversations.updated_at
		RETURNING id, user_id, coach_id, created_at, updated_at
	`

	var conversation models.Conversation
	err := r.db.QueryRow(ctx, query, userID, coachID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.CoachID,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *ConversationRepository) GetByID(ctx context.Context, conversationID int64) (*models.Conversation, error) {
	query := `
		SELECT id, user_id, coach_id, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	var conversation models.Conversation
	err := r.db.QueryRow(ctx, query, conversationID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.CoachID,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *ConversationRepository) GetByIDForParticipant(
	ctx context.Context,
	conversationID int64,
	participantID int64,
) (*models.Conversation, error) {
	query := `
		SELECT id, user_id, coach_id, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND (user_id = $2 OR coach_id = $2)
	`

	var conversation models.Conversation
	err := r.db.QueryRow(ctx, query, conversationID, participantID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.CoachID,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *ConversationRepository) ListForParticipant(
	ctx context.Context,
	participantID int64,
) ([]models.ConversationSummary, error) {
	query := `
		SELECT
			c.id,
			c.user_id,
			c.coach_id,
			c.created_at,
			c.updated_at,
			lm.id,
			lm.conversation_id,
			lm.sender_id,
			lm.content,
			lm.is_read,
			lm.created_at,
			COALESCE(uc.unread_count, 0)
		FROM conversations c
		LEFT JOIN LATERAL (
			SELECT id, conversation_id, sender_id, content, is_read, created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS unread_count
			FROM messages
			WHERE conversation_id = c.id
			  AND sender_id <> $1
			  AND is_read = FALSE
		) uc ON TRUE
		WHERE c.user_id = $1 OR c.coach_id = $1
		ORDER BY COALESCE(lm.created_at, c.updated_at, c.created_at) DESC, c.id DESC
	`

	rows, err := r.db.Query(ctx, query, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]models.ConversationSummary, 0)
	for rows.Next() {
		var summary models.ConversationSummary
		var messageID sql.NullInt64
		var messageConversationID sql.NullInt64
		var messageSenderID sql.NullInt64
		var messageContent sql.NullString
		var messageIsRead sql.NullBool
		var messageCreatedAt sql.NullTime

		if err := rows.Scan(
			&summary.ID,
			&summary.UserID,
			&summary.CoachID,
			&summary.CreatedAt,
			&summary.UpdatedAt,
			&messageID,
			&messageConversationID,
			&messageSenderID,
			&messageContent,
			&messageIsRead,
			&messageCreatedAt,
			&summary.UnreadCount,
		); err != nil {
			return nil, err
		}

		if messageID.Valid {
			summary.LastMessage = &models.ChatMessage{
				ID:             messageID.Int64,
				ConversationID: messageConversationID.Int64,
				SenderID:       messageSenderID.Int64,
				Content:        messageContent.String,
				IsRead:         messageIsRead.Bool,
				CreatedAt:      messageCreatedAt.Time,
			}
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}

func (r *ConversationRepository) Touch(ctx context.Context, conversationID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversations
		SET updated_at = NOW()
		WHERE id = $1
	`, conversationID)
	return err
}
