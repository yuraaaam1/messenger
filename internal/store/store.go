package store

import "github.com/jackc/pgx/v5/pgxpool"

type Store struct {
	RoomStore *RoomStore
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		RoomStore: NewRoomStore(db),
	}
}
