package repository

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type MessageRepository struct {
	db DBTX
}

func NewMessageRepository(db DBTX) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(
	ctx context.Context,
	conversationID int64,
	senderID int64,
	content string,
) (*models.ChatMessage, error) {
	query := `
		INSERT INTO messages (conversation_id, sender_id, content, is_read)
		VALUES ($1, $2, $3, FALSE)
		RETURNING id, conversation_id, sender_id, content, is_read, created_at
	`

	var message models.ChatMessage
	err := r.db.QueryRow(ctx, query, conversationID, senderID, content).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Content,
		&message.IsRead,
		&message.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *MessageRepository) ListByConversation(
	ctx context.Context,
	conversationID int64,
	limit int,
	offset int,
) ([]models.ChatMessage, int, error) {
	totalQuery := `
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = $1
	`

	var total int
	if err := r.db.QueryRow(ctx, totalQuery, conversationID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, conversation_id, sender_id, content, is_read, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	messages := make([]models.ChatMessage, 0)
	for rows.Next() {
		var message models.ChatMessage
		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.IsRead,
			&message.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *MessageRepository) MarkConversationRead(
	ctx context.Context,
	conversationID int64,
	readerID int64,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE messages
		SET is_read = TRUE
		WHERE conversation_id = $1
		  AND sender_id <> $2
		  AND is_read = FALSE
	`, conversationID, readerID)
	return err
}

func (r *MessageRepository) MarkMessagesRead(
	ctx context.Context,
	messageIDs []int64,
	readerID int64,
) error {
	if len(messageIDs) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE messages
		SET is_read = TRUE
		WHERE id = ANY($1)
		  AND sender_id <> $2
		  AND is_read = FALSE
	`, messageIDs, readerID)
	return err
}
