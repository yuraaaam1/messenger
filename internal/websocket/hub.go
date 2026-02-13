package websocket

import (
	"context"
	"encoding/json"
	"log"
	"messenger/internal/models"
	"messenger/internal/store"
)

type Hub struct {
	clients    map[*Client]bool // Зарегестрированные клиенты
	broadcast  chan []byte      // Входящие сообщения
	register   chan *Client     // Запросы на регистрацию
	unregister chan *Client     // Запросы на отмену регистрации
	store      *store.Store
}

func NewHub(s *store.Store) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		store:      s,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Успешное подключение. Всего подключений:", len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Успешное отключение. Всего подключений:", len(h.clients))
			}
		case messageData := <-h.broadcast:
			var msg models.Message
			if err := json.Unmarshal(messageData, &msg); err != nil {
				log.Printf("Ошибка при распаковке JSON: %v", err)
				continue
			}

			savedMsg, err := h.store.CreateMessage(context.Background(), &msg)
			if err != nil {
				log.Printf("Ошибка при сохранении сообщения в БД: %v", err)
				continue
			}

			broadcastData, err := json.Marshal(savedMsg)
			if err != nil {
				log.Printf("Ошибка при упаковке JSON для рассылки: %v", err)
				continue
			}

			for client := range h.clients {
				select {
				case client.send <- broadcastData:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
