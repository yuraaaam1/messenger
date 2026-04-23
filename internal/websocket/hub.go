package websocket

import (
	"log"
)

type ClientMessage struct {
	Client  *Client
	Message []byte
}

type Hub struct {
	rooms      map[int64]map[*Client]bool // Зарегистрированные клиенты
	broadcast  chan *ClientMessage        // Входящие сообщения
	register   chan *Client               // Запросы на регистрацию
	unregister chan *Client               // Запросы на отмену регистрации
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[int64]map[*Client]bool),
		broadcast:  make(chan *ClientMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			log.Printf("Клиент подключен к комнате %d. Клиентов в комнате: %d", client.RoomID, len(h.rooms[client.RoomID]))

		case client := <-h.unregister:
			if room, ok := h.rooms[client.RoomID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.send)
					log.Printf("Клиент отключён от комнаты %d. Клиентов в комнате: %d", client.RoomID, len(h.rooms[client.RoomID]))
				}
			}

		case clientMsg := <-h.broadcast:
			room := h.rooms[clientMsg.Client.RoomID]

			for client := range room {
				if client == clientMsg.Client {
					continue
				}
				select {
				case client.send <- clientMsg.Message:
				default:
					close(client.send)
					delete(room, client)
				}
			}
		}
	}
}
