package handlers

import (
	"encoding/json"
	"log"
	"messenger/internal/store"
	"net/http"
)

type RoomHandler struct {
	store *store.Store
}

func NewRoomHandler(store *store.Store) *RoomHandler {
	return &RoomHandler{
		store: store,
	}
}

type CreateRoomRequest struct {
	Name    string `json:"name"`
	KeyHash string `json:"key_hash"`
}

type JoinRoomRequest struct {
	KeyHash       string `json:"key_hash"`
	DeviceKeyHash string `json:"device_key_hash"`
}

type JoinRoomResponse struct {
	RoomID int64  `json:"room_id"`
	Name   string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Неверный формат запроса"})
		return
	}

	if req.KeyHash == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name и key_hash обязательны"})
		return
	}

	room, err := h.store.RoomStore.CreateRoom(r.Context(), req.Name, req.KeyHash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Непредвиденная ошибка, не удалось создать комнату"})
		return
	}

	writeJSON(w, http.StatusCreated, room)
}

func (h *RoomHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	var req JoinRoomRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Неверный формат запроса"})
		return
	}

	if req.KeyHash == "" || req.DeviceKeyHash == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "key_hash и device_key_hash обязательны"})
		return
	}

	room, err := h.store.RoomStore.GetRoomByHash(r.Context(), req.KeyHash)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Комната не найдена"})
		return
	}

	_, err = h.store.RoomStore.RegisterDevice(r.Context(), room.ID, req.DeviceKeyHash)
	if err != nil {
		log.Printf("Ошибка регистрации устройства: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Непредвиденная ошибка, не удалось зарегистрировать устройство"})
		return
	}

	writeJSON(w, http.StatusOK, JoinRoomResponse{RoomID: room.ID, Name: room.Name})
}
