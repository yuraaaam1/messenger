package handlers

import (
	"messenger/internal/auth"
	"messenger/internal/store"
	"net/http"
	"strconv"
)

type MessageHandler struct {
	store *store.Store
}

func NewMessageHandler(store *store.Store) *MessageHandler {
	return &MessageHandler{store: store}
}

func (h *MessageHandler) GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.JWTClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Не авторизован"})
		return
	}

	chatIDStr := r.URL.Query().Get("chat_id")
	if chatIDStr == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "параметр chat_id обязателен"})
		return
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "chat_id должен быть числом"})
		return
	}

	isMember, err := h.store.ChatStore.IsMember(r.Context(), chatID, claims.UserID)
	if err != nil || !isMember {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "Нет доступа к этому чату"})
		return
	}

	messages, err := h.store.MessageStore.GetMessages(r.Context(), chatID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось получить сообщения"})
		return
	}

	writeJSON(w, http.StatusOK, messages)
}
