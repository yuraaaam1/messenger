package store

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5"
)

type Store struct {
	db *pgx.Conn
}

func NewStore(db *pgx.Conn) *Store {
	return &Store{db: db}
}

func (s *Store) GetMessages(ctx context.Context) ([]models.Message, error) {
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

func (s *Store) CreateMessage(ctx context.Context, msg *models.Message) (*models.Message, error) {
	var senderID int64 = 1
	var chatID int64 = 1

	if msg.User == "Bob" {
		senderID = 2
	}

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

	err := s.db.QueryRow(ctx, query, senderID, chatID, msg.Text).Scan(
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
