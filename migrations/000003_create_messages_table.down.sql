DROP TRIGGER IF EXISTS trigger_update_chat_on_message ON messages;
DROP FUNCTION IF EXISTS update_chat_updated_at;

DROP INDEX IF EXISTS idx_messages_uuid;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_sender_id;
DROP INDEX IF EXISTS idx_messages_chat_id;

DROP TABLE IF EXISTS messages;