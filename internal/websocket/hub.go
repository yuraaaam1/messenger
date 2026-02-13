package websocket

import "log"

type Hub struct {
	clients    map[*Client]bool // Зарегестрированные клиенты
	broadcast  chan []byte      // Входящие сообщения
	register   chan *Client     // Запросы на регистрацию
	unregister chan *Client     // Запросы на отмену регистрации
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
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
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
