package models

import "time"

type Message struct {
	ChatID int64     `json:"chat_id"`
	User   string    `json:"user"`
	Text   string    `json:"text"`
	SentAt time.Time `json:"sent_at"`
}
