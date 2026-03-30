package handlers

import (
	"encoding/json"
	"messenger/internal/auth"
	"messenger/internal/store"
	"net/http"
)

type ChatHandler struct {
	store *store.Store
}

func NewChatHandler(store *store.Store) *ChatHandler {
	return &ChatHandler{store: store}
}

type CreateChatRequest struct {
	Name      string  `json:"name"`
	IsGroup   bool    `json:"is_group"`
	MemberIDs []int64 `json:"member_ids"`
}

func (h *ChatHandler) GetChatHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.JWTClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Не авторизован"})
		return
	}

	chats, err := h.store.ChatStore.GetUserChats(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось получить чаты"})
		return
	}

	writeJSON(w, http.StatusOK, chats)
}

func (h *ChatHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.JWTClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Не авторизован"})
		return
	}

	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Не верный формат запроса"})
		return
	}

	chat, err := h.store.ChatStore.CreateChat(r.Context(), req.Name, req.IsGroup, claims.UserID, req.MemberIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось создать чат"})
		return
	}

	writeJSON(w, http.StatusCreated, chat)
}
