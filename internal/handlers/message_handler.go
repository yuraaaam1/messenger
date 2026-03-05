package handlers

import (
	"messenger/internal/store"
	"net/http"
)

type MessageHandler struct {
	store *store.Store
}

func NewMessageHandler(store *store.Store) *MessageHandler {
	return &MessageHandler{store: store}
}

func (h *MessageHandler) GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := h.store.MessageStore.GetMessages(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось получить сообщения"})
		return
	}

	writeJSON(w, http.StatusOK, messages)
}
