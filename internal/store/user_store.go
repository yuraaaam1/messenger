package store

import (
	"context"
	"errors"
	"fmt"

	"messenger/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, username, email, password string) (*models.User, error) {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {

		return nil, fmt.Errorf("ошибка при хешировании пароля: %w", err)
	}

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	query := `
	INSERT INTO users (username, email, password_hash)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at`

	err = s.db.QueryRow(
		ctx,
		query,
		user.Username,
		user.Email,
		user.PasswordHash).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt)

	if err != nil {

		return nil, fmt.Errorf("ошибка при создании пользователя: %w", err)
	}

	return user, nil
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1`

	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("Ошибка при поиске пользователя по email: %w", err)
	}
	return user, nil

}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
