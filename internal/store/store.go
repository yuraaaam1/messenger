package store

import "github.com/jackc/pgx/v5/pgxpool"

type Store struct {
	UserStore    *UserStore
	MessageStore *MessageStore
	ChatStore    *ChatStore
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		UserStore:    NewUserStore(db),
		MessageStore: NewMessageStore(db),
		ChatStore:    NewChatStore(db),
	}
}
