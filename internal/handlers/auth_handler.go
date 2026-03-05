package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"messenger/internal/auth"
	"messenger/internal/models"
	"messenger/internal/store"
	"net/http"
)

type AuthHandler struct {
	store     *store.Store
	jwtSecret string
}

func NewAuthHandler(store *store.Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

// Структура запроса на регистрацию;
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Структура запроса на вход;
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Структура успешного ответа
type AuthResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Неверный формат запроса"})
		return
	}

	// TODO: добавить валидацию (длина пароля, формат email и т.д);

	user, err := h.store.UserStore.CreateUser(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		// TODO: необходимо различать ошибку "пользователь уже существует";
		log.Printf("ОШИБКА СОЗДАНИЯ ПОЛЬЗОВАТЕЛЯ: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось создать пользователя"})
		return
	}

	tokenString, err := auth.GenerateJWT(user, h.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось сгенерировать токен"})
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{Token: tokenString})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Неверный формат запроса"})
		return
	}

	user, err := h.store.UserStore.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверные учётные данные"})
		} else {
			log.Printf("ОШИБКА ПОИСКА ПОЛЬЗОВАТЕЛЯ: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка сервера"})
		}
		return
	}

	if !store.CheckPasswordHash(req.Password, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверные учётные данные"})
		return
	}

	tokenString, err := auth.GenerateJWT(user, h.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Не удалось сгенерировать токен"})
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: tokenString})
}
