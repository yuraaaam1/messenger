package main

import (
	"context"
	"fmt"
	"log"
	"messenger/internal/handlers"
	"messenger/internal/store"
	"messenger/internal/websocket"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
}

func loadConfig() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("Переменная окружения \"DATABASE_URL\" не установлена")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("Переменная окружени \"JWT_SECRET\" не установлена")
	}

	return Config{
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения системы")
	}

	// Загружаем конфигурации с env
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	// Создаём пул соединений с базой данных
	dbpool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer dbpool.Close()

	mainStore := store.NewStore(dbpool)
	messageHub := websocket.NewHub(mainStore)
	go messageHub.Run()

	fs := http.FileServer(http.Dir("./frontend"))

	http.Handle("/", fs)

	// Инициализируем хендлер и маршруты для аутентификации
	authHandler := handlers.NewAuthHandler(mainStore, config.JWTSecret)
	http.HandleFunc("/api/auth/register", authHandler.Register)
	http.HandleFunc("/api/auth/login", authHandler.Login)

	// Маршрут для подтягивания истории сообщений
	messageHandler := handlers.NewMessageHandler(mainStore)
	http.HandleFunc("/api/messages", messageHandler.GetMessagesHandler)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(messageHub, w, r, config.JWTSecret)
	})

	log.Println("Сервер запущен на http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

}
