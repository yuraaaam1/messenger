package handlers

import (
	"messenger/internal/auth"
	"messenger/internal/store"
	"net/http"
)

type UserHandler struct {
	store *store.Store
}

func NewUserHandler(store *store.Store) *UserHandler {
	return &UserHandler{store: store}
}

func (h *UserHandler) SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserContextKey).(*auth.JWTClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Не авторизован"})
		return
	}

	query := r.URL.Query().Get("q")

	users, err := h.store.UserStore.SearchUsers(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка поиска"})
		return
	}

	writeJSON(w, http.StatusOK, users)
}
