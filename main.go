package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type Message struct {
	User string `json:"user"`
	Text string `json:"text"`
}

func main() {
	connString := "postgres://messenger_user:pass1905word@localhost:5432/messenger?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v\n", err)
	}
	defer conn.Close(context.Background())
	log.Println("Успешное подключение к базе данных!")

	_, err = conn.Exec(context.Background(), `
		INSERT INTO users (id, username, email, password_hash) VALUES (1, 'Alice', 'alice@example.com', 'hash1') ON CONFLICT (id) DO NOTHING;
		INSERT INTO users (id, username, email, password_hash) VALUES (2, 'Bob', 'bob@example.com', 'hash2') ON CONFLICT (id) DO NOTHING;
        INSERT INTO chats (id, name) VALUES (1, 'General') ON CONFLICT (id) DO NOTHING;
        INSERT INTO chat_participants (chat_id, user_id) VALUES (1, 1) ON CONFLICT DO NOTHING;
        INSERT INTO chat_participants (chat_id, user_id) VALUES (1, 2) ON CONFLICT DO NOTHING;
	`)

	if err != nil {
		log.Printf("Не удалось добавить тестовые данные (возможно, они уже есть): %v\n", err)
	}

	_, err = conn.Exec(context.Background(), `
		INSERT INTO messages (chat_id, sender_id, encrypted_content, iv) SELECT 1, 1, 'Привет из базы данных!', 'iv1' WHERE NOT EXISTS (SELECT 1 FROM messages WHERE encrypted_content = 'Привет из базы данных!');
		INSERT INTO messages (chat_id, sender_id, encrypted_content, iv) SELECT 1, 2, 'А вот и я!', 'iv2' WHERE NOT EXISTS (SELECT 1 FROM messages WHERE encrypted_content = 'А вот и я!');
    `)
	if err != nil {
		log.Printf("Не удалось добавить тестовые сообщения (возможно, они уже существуют): %v\n", err)
	}

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	http.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		rows, err := conn.Query(context.Background(),
			`SELECT u.username, m.encrypted_content
			FROM messages m
			JOIN users u ON m.sender_id = u.id
			ORDER BY m.created_at ASC`)

		if err != nil {
			log.Printf("Ошибка при запросе к базе данных: %v\n", err)
			http.Error(w, "InternalServerError", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []Message
		for rows.Next() {
			var msg Message
			var contentBytes []byte

			if err := rows.Scan(&msg.User, &contentBytes); err != nil {
				log.Printf("Ошибка при сканировании строки: %v\n", err)
				continue
			}
			msg.Text = string(contentBytes)
			messages = append(messages, msg)
		}

		if messages == nil {
			messages = []Message{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	log.Println("Запуск сервера на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
