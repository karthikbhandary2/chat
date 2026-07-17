package handlers

import (
	"encoding/json"
	"fmt"
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

type CreateConversationRequest struct {
	Type        string   `json:"type" validate:"required,oneof=direct group"`
	Usernames   []string `json:"usernames" validate:"required,min=1"`
}

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

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	conversationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	uID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	isParticipant, err := h.Store.IsParticipant(r.Context(), db.IsParticipantParams{
		ConversationID: conversationID, UserID: userID,
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

	count, err := h.Store.GetUnreadCount(r.Context(), db.GetUnreadCountParams{
		ConversationID: conversationID, UserID: userID,
	})
	if err != nil {
		log.Println("get unread count error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int64{"unread_count": count})
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	conversationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	uID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.Store.MarkAsRead(r.Context(), db.MarkAsReadParams{
		ConversationID: conversationID, UserID: userID,
	}); err != nil {
		log.Println("mark as read error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	uID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing search query", http.StatusBadRequest)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := h.Store.SearchMessages(r.Context(), db.SearchMessagesParams{
		UserID:         userID,
		PlaintoTsquery: query,
		Limit:          int32(limit),
	})
	if err != nil {
		log.Println("search messages error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(messages)
}

func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validate.Struct(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	callerIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	callerID, err := uuid.Parse(callerIDStr)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	participantIDs := []uuid.UUID{callerID}
	for _, username := range req.Usernames {
		user, err := h.Store.GetUserByUsername(r.Context(), username)
		if err != nil {
			http.Error(w, fmt.Sprintf("user not found: %s", username), http.StatusNotFound)
			return
		}
		participantIDs = append(participantIDs, user.ID)
	}

	if req.Type == "direct" && len(participantIDs) != 2 {
		http.Error(w, "a direct conversation needs exactly one other participant", http.StatusBadRequest)
		return
	}

	conversation, err := h.Store.CreateConversationWithParticipants(r.Context(), req.Type, participantIDs)
	if err != nil {
		log.Println("create conversation error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conversation)
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	uID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	conversations, err := h.Store.ListUserConversations(r.Context(), userID)
	if err != nil {
		log.Println("list conversations error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(conversations)
}

func (h *Handler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := h.Store.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"id":       user.ID.String(),
		"username": user.Username,
	})
}