package models

import "time"

type Chat struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IsGroup   bool      `json:"is_group"`
	CreatedAt time.Time `json:"created_at"`
}
