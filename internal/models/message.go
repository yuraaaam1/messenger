package models

import "time"

type Message struct {
	User   string    `json:"user"`
	Text   string    `json:"text"`
	SentAt time.Time `json:"sent_at"`
}
