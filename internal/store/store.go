package store

import "github.com/jackc/pgx/v5/pgxpool"

type Store struct {
	*UserStore
	*MessageStore
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		UserStore:    NewUserStore(db),
		MessageStore: NewMessageStore(db),
	}
}
