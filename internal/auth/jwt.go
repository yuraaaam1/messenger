package auth

import (
	"fmt"
	"messenger/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateJWT(user *models.User, jwtSecret string) (string, error) {
	secretKey := []byte(jwtSecret)

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &JWTClaims{
		UserID:   int64(user.ID),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("Ошибка при подписи токена: %w", err)
	}

	return tokenString, nil
}

func ValidateJWT(tokenString string, jwtSecret string) (*JWTClaims, error) {
	secretKey := []byte(jwtSecret)

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Неожиданный метод подписи: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("Ошибка при парсинге токена: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("Невалидный токен")
	}
}
