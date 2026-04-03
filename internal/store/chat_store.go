package store

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatStore struct {
	db *pgxpool.Pool
}

func NewChatStore(db *pgxpool.Pool) *ChatStore {
	return &ChatStore{db: db}
}

// GetUserChats для возврата списка чатов пользователя
func (s *ChatStore) GetUserChats(ctx context.Context, userID int64) ([]*models.Chat, error) {
	query := `
	SELECT c.id, c.name, c.is_group, c.created_at 
	FROM chats c
	JOIN chat_participants cp ON cp.chat_id = c.id
	WHERE cp.user_id = $1
	ORDER BY c.created_at DESC`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*models.Chat
	for rows.Next() {
		chat := &models.Chat{}
		if err := rows.Scan(
			&chat.ID,
			&chat.Name,
			&chat.IsGroup,
			&chat.CreatedAt,
		); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

// Создание чата с добавлением создателя и других членов в чат
func (s *ChatStore) CreateChat(ctx context.Context, name string, isGroup bool, creatorID int64, memberIDs []int64) (*models.Chat, error) {
	chat := &models.Chat{}

	err := s.db.QueryRow(ctx,
		`INSERT INTO chats (name, is_group) VALUES($1, $2) RETURNING id, name, is_group, created_at`,
		name, isGroup).Scan(
		&chat.ID,
		&chat.Name,
		&chat.IsGroup,
		&chat.CreatedAt)
	if err != nil {
		return nil, err
	}

	allMembers := append([]int64{creatorID}, memberIDs...)

	for _, uid := range allMembers {
		_, err := s.db.Exec(ctx,
			`INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			chat.ID, uid)
		if err != nil {
			return nil, err
		}
	}

	return chat, nil
}

// Проверка является ли пользователь участником чата
func (s *ChatStore) IsMember(ctx context.Context, chatID, userID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id=$1 AND user_id=$2)`,
		chatID, userID).Scan(&exists)

	return exists, err
}

// Получение всех пользователей чата
func (s *ChatStore) GetMemberIDs(ctx context.Context, chatID int64) ([]int64, error) {
	rows, err := s.db.Query(ctx,
		`SELECT user_id FROM chat_participants
	WHERE chat_id = $1`, chatID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}
