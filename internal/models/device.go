package models

import "time"

type Device struct {
	ID            int64      `json:"id"`
	RoomID        int64      `json:"room_id"`
	DeviceKeyHash string     `json:"device_key_hash"`
	CreatedAt     time.Time  `json:"created_at"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
}
