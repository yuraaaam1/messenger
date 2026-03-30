package websocket

import (
	"context"
	"encoding/json"
	"log"
	"messenger/internal/models"
	"messenger/internal/store"
)

type ClientMessage struct {
	Client  *Client
	Message []byte
}

type Hub struct {
	clients    map[*Client]bool    // Зарегистрированные клиенты
	broadcast  chan *ClientMessage // Входящие сообщения
	register   chan *Client        // Запросы на регистрацию
	unregister chan *Client        // Запросы на отмену регистрации
	store      *store.Store
}

func NewHub(s *store.Store) *Hub {
	return &Hub{
		broadcast:  make(chan *ClientMessage),
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
			log.Printf("Клиент %s (ID: %d) подключен. Всего подключений: %d", client.Username, client.UserID, len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Клиент %s (ID: %d) отключён. Всего подключений: %d", client.Username, client.UserID, len(h.clients))
			}
		case clientMsg := <-h.broadcast:
			var msg models.Message
			if err := json.Unmarshal(clientMsg.Message, &msg); err != nil {
				log.Printf("Ошибка при распаковке JSON: %v", err)
				continue
			}

			msg.User = clientMsg.Client.Username

			savedMsg, err := h.store.MessageStore.CreateMessage(context.Background(), &msg, clientMsg.Client.UserID)
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
