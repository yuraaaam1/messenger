package models

import "time"

type Room struct {
	ID        int64     `json:"id"`
	KeyHash   string    `json:"key_hash"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
