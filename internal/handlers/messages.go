package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"github.com/karthikbhandary2/chat/internal/middleware"
)

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conversationID, err := uuid.Parse(id)
	if err != nil {
		log.Println("parse conversation id error:", err)
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	uID, ok := middleware.GetUserID(r.Context())
	if !ok {
		log.Println("missing user id in context")
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		log.Println("parse user id error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	yes, err := h.Store.IsParticipant(r.Context(), db.IsParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		log.Println("Is participant error:", err)
		http.Error(w, "participant not found", http.StatusNotFound)
		return
	}
	if !yes {
		http.Error(w, "not authorized", http.StatusForbidden)
	}

	beforeStr := r.URL.Query().Get("before")
	var before time.Time
	if beforeStr == "" {
		before = time.Now()
	} else {
		before, err = time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			http.Error(w, "invalid before parameter", http.StatusBadRequest)
			return
		}
	}

	limit := r.URL.Query().Get("limit")
	Limit, err := strconv.Atoi(limit)
	if err != nil {
		log.Println("string to int conversion error", err)
	}
	messages, err := h.Store.GetConversationMessages(r.Context(), db.GetConversationMessagesParams{
		ConversationID: conversationID,
		CreatedAt:      pgtype.Timestamptz{Time: before, Valid: true},
		Limit:          int32(Limit),
	})
	if err != nil {
		log.Println("get conversation messages error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(messages)

}
