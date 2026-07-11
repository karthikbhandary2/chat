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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		log.Println("parse user id error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	isParticipant, err := h.Store.IsParticipant(r.Context(), db.IsParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		log.Println("is participant error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !isParticipant {
		http.Error(w, "not found", http.StatusNotFound)
		return
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

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := h.Store.GetConversationMessages(r.Context(), db.GetConversationMessagesParams{
		ConversationID: conversationID,
		CreatedAt:      pgtype.Timestamptz{Time: before, Valid: true},
		Limit:          int32(limit),
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
