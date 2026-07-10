package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationParticipant struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	JoinedAt       time.Time `json:"joined_at"`
}

type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}
