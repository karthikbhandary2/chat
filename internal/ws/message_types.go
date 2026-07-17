package ws

import "github.com/google/uuid"

// Envelope is used to peek at an incoming payload's type before
// deciding which concrete struct to unmarshal it into.
type Envelope struct {
	Type string `json:"type"`
}

// IncomingMessage represents a persisted chat message sent into a conversation.
type IncomingMessage struct {
	Type           string    `json:"type"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Content        string    `json:"content"`
}

// TypingEvent represents an ephemeral, non-persisted typing notification.
type TypingEvent struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}
