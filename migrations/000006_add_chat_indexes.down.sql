DROP INDEX IF EXISTS idx_messages_conversation_unread;
DROP INDEX IF EXISTS idx_conversations_coach_id;
DROP INDEX IF EXISTS idx_conversations_user_id;

ALTER TABLE messages
    ALTER COLUMN sender_id DROP NOT NULL,
    ALTER COLUMN conversation_id DROP NOT NULL;

ALTER TABLE conversations
    ALTER COLUMN coach_id DROP NOT NULL,
    ALTER COLUMN user_id DROP NOT NULL;
