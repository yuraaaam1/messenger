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
}

func loadConfig() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("Переменная окружения \"DATABASE_URL\" не установлена")
	}

	return Config{
		DatabaseURL: dbURL,
	}, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используюстя переменные окружения системы")
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Ошибка при запуске конфигурации: %v", err)
	}

	dbpool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer dbpool.Close()

	mainStore := store.NewStore(dbpool)
	messageHub := websocket.NewHub()
	go messageHub.Run()

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	roomHandler := handlers.NewRoomHandler(mainStore)
	http.HandleFunc("POST /api/rooms", roomHandler.CreateRoom)
	http.HandleFunc("POST /api/rooms/join", roomHandler.JoinRoom)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(messageHub, w, r, mainStore)
	})

	log.Println("Сервер запущен на http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

}
