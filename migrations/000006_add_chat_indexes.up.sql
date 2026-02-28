ALTER TABLE conversations
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN coach_id SET NOT NULL;

ALTER TABLE messages
    ALTER COLUMN conversation_id SET NOT NULL,
    ALTER COLUMN sender_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_coach_id ON conversations(coach_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_unread
    ON messages(conversation_id, is_read, created_at DESC);
