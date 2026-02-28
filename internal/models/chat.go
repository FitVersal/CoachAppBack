package models

import "time"

type Conversation struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	CoachID   int64     `json:"coach_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	SenderID       int64     `json:"sender_id"`
	Content        string    `json:"content"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationSummary struct {
	Conversation
	LastMessage *ChatMessage `json:"last_message,omitempty"`
	UnreadCount int          `json:"unread_count"`
}
