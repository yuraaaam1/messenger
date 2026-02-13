package handlers

import (
	"encoding/json"
	"log"
	"messenger/internal/store"
	"net/http"
)

type MessageHandler struct {
	store *store.Store
}

func NewMessageHandler(s *store.Store) *MessageHandler {
	return &MessageHandler{store: s}
}

func (h *MessageHandler) GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := h.store.GetMessages(r.Context())
	if err != nil {
		log.Printf("Ошибка при  получении сообщений из хранилища: %v\n", err)
		http.Error(w, "InternalServerError", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
