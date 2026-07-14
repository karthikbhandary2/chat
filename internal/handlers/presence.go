package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/karthikbhandary2/chat/internal/presence"
)

func (h *Handler) GetPresence(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	count, err := h.RedisClient.Exists(r.Context(), presence.Key(userID)).Result()
	if err != nil {
		log.Println("exists error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	online := count > 0
	response := struct {
		UserID string `json:"user_id"`
		Online bool   `json:"online"`
	}{
		UserID: userID,
		Online: online,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

}
