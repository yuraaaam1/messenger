package main

import (
	"context"
	"log"
	"messenger/internal/handlers"
	"messenger/internal/store"
	"net/http"

	"messenger/internal/database"

	"github.com/jackc/pgx/v5"
)

func main() {
	connString := "postgres://messenger_user:pass1905word@localhost:5432/messenger?sslmode=disable"

	db, err := database.NewConnection(connString)
	if err != nil {
		log.Fatalf("Не удалось инициализировать подклчючение к базе данных: %v", err)
	}
	defer db.Close(context.Background())

	addTestData(db)

	messageStore := store.NewStore(db)

	messageHandler := handlers.NewMessageHandler(messageStore)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)
	mux.HandleFunc("/api/messages", messageHandler.GetMessagesHandler)

	log.Println("Запуск сервера на http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}

func addTestData(db *pgx.Conn) {
	_, err := db.Exec(context.Background(), `
		INSERT INTO users (id, username, email, password_hash) VALUES (1, 'Alice', 'alice@example.com', 'hash1') ON CONFLICT (id) DO NOTHING;
		INSERT INTO users (id, username, email, password_hash) VALUES (2, 'Bob', 'bob@example.com', 'hash2') ON CONFLICT (id) DO NOTHING;
        INSERT INTO chats (id, name) VALUES (1, 'General') ON CONFLICT (id) DO NOTHING;
        INSERT INTO chat_participants (chat_id, user_id) VALUES (1, 1) ON CONFLICT DO NOTHING;
        INSERT INTO chat_participants (chat_id, user_id) VALUES (1, 2) ON CONFLICT DO NOTHING;
	`)

	if err != nil {
		log.Printf("Не удалось добавить базовые сущности (возможно, они уже существуют): %v\n", err)
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO messages (chat_id, sender_id, encrypted_content, iv) SELECT 1, 1, 'Привет из базы данных!', 'iv1' WHERE NOT EXISTS (SELECT 1 FROM messages WHERE encrypted_content = 'Привет из базы данных!');
		INSERT INTO messages (chat_id, sender_id, encrypted_content, iv) SELECT 1, 2, 'А вот и я!', 'iv2' WHERE NOT EXISTS (SELECT 1 FROM messages WHERE encrypted_content = 'А вот и я!');
    `)

	if err != nil {
		log.Printf("Не удалось добавить тестовые сообщения (возможно, они уже существуют): %v\n", err)
	}
}
