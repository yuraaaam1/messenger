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
	rows, err := s.db.Query(ctx,
		`SELECT u.username, m.encrypted_content
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

		if err := rows.Scan(&msg.User, &contentBytes); err != nil {
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
