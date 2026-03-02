package websocket

import (
	"log"
	"messenger/internal/auth"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	maxMessageSize = 512
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // В будущем необходимо допилить проверку домена
	},
}

type Client struct {
	hub *Hub

	conn *websocket.Conn

	send chan []byte

	UserID int64

	Username string
}

// readPump считывает сообщения от клиента и передаёт их в хаб
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		c.hub.broadcast <- &ClientMessage{Client: c, Message: message} // Передача в хаб
	}

}

// writePump отправляет сообщения из хаба клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok { // Проверка на открытый или закрытый коннект
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			n := len(c.send)

			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Обрабатывает websocket запрросы от клиента
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, jwtSecret string) {
	// Получаем токен из параметров запроса;
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		log.Println("Токен отсутствует в запросе")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}
	// Валидируем токен;
	claims, err := auth.ValidateJWT(tokenString, jwtSecret)
	if err != nil {
		log.Printf("Ошибка валидации токена: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	// Обновляемся до WebSocket;
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Создаём клиента
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),

		UserID:   claims.UserID,
		Username: claims.Username,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
