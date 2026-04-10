package store

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomStore struct {
	db *pgxpool.Pool
}

func NewRoomStore(db *pgxpool.Pool) *RoomStore {
	return &RoomStore{db: db}
}

// Создание новой комнаты
func (s *RoomStore) CreateRoom(ctx context.Context, name, keyHash string) (*models.Room, error) {
	query := `
	INSERT INTO rooms (name, key_hash)
	VALUES ($1, $2)
	RETURNING id, key_hash, name, created_at`

	room := &models.Room{}
	err := s.db.QueryRow(ctx, query, name, keyHash).Scan(
		&room.ID,
		&room.KeyHash,
		&room.Name,
		&room.CreatedAt)

	if err != nil {
		return nil, err
	}

	return room, nil
}

// Поиск комнаты по хешу ключа
func (s *RoomStore) GetRoomByHash(ctx context.Context, key_hash string) (*models.Room, error) {
	query := `
	SELECT id, key_hash, name, created_at FROM rooms
	WHERE key_hash = $1`

	room := &models.Room{}
	err := s.db.QueryRow(ctx, query, key_hash).Scan(
		&room.ID,
		&room.KeyHash,
		&room.Name,
		&room.CreatedAt)

	if err != nil {
		return nil, err
	}

	return room, nil
}

// Регистрация нового устройства в комнате
func (s *RoomStore) RegisterDevice(ctx context.Context, roomID int64, deviceKeyHash string) (*models.Device, error) {
	query := `
	INSERT INTO devices (room_id, device_key_hash)
	VALUES ($1, $2)
	ON CONFLICT (device_key_hash) DO UPDATE SET last_seen_at = NOW()
	RETURNING id, room_id, device_key_hash, created_at, last_seen_at`

	device := &models.Device{}
	err := s.db.QueryRow(ctx, query, roomID, deviceKeyHash).Scan(
		&device.ID,
		&device.RoomID,
		&device.DeviceKeyHash,
		&device.CreatedAt,
		&device.LastSeenAt)

	if err != nil {
		return nil, err
	}

	return device, nil
}

func (s *RoomStore) GetDeviceRoomID(ctx context.Context, deviceKeyHash string) (int64, error) {
	query := `
	SELECT room_id FROM devices
	WHERE device_key_hash = $1`

	var roomID int64
	err := s.db.QueryRow(ctx, query, deviceKeyHash).Scan(
		&roomID)

	if err != nil {
		return 0, err
	}

	return roomID, nil
}
