package store

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageStore struct {
	db *pgxpool.Pool
}

func NewMessageStore(db *pgxpool.Pool) *MessageStore {
	return &MessageStore{db: db}
}

func (s *MessageStore) GetMessages(ctx context.Context) ([]models.Message, error) {
	rows, err := s.db.Query(ctx, `
		SELECT u.username, m.encrypted_content, m.created_at
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		ORDER BY m.created_at ASC`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var messages []models.Message

	for rows.Next() {
		var msg models.Message
		var contentBytes []byte

		if err := rows.Scan(&msg.User, &contentBytes, &msg.SentAt); err != nil {
			return nil, err
		}
		msg.Text = string(contentBytes)
		messages = append(messages, msg)

	}

	if messages == nil {
		messages = []models.Message{}
	}

	return messages, nil
}

func (s *MessageStore) CreateMessage(ctx context.Context, msg *models.Message, userID int64) (*models.Message, error) {

	var chatID int64 = 1

	const query = `
	WITH new_msg AS (
		INSERT INTO messages (sender_id, chat_id, encrypted_content, iv)
		VALUES ($1, $2, $3, 'temp_iv')
		RETURNING id, sender_id, encrypted_content, created_at
	)
	SELECT u.username, nm.encrypted_content, nm.created_at
	FROM new_msg nm
	JOIN users u ON nm.sender_id = u.id;
	`

	var savedMsg models.Message
	var contentBytes []byte

	err := s.db.QueryRow(ctx, query, userID, chatID, msg.Text).Scan(
		&savedMsg.User,
		&contentBytes,
		&savedMsg.SentAt,
	)

	if err != nil {
		return nil, err
	}

	savedMsg.Text = string(contentBytes)

	return &savedMsg, nil
}
