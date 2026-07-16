package ws

import "github.com/google/uuid"

type IncomingMessage struct {
	Type           string    `json:"type"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Content        string    `json:"content"`
}

type Envelope struct {
	Type string `json:"type"`
}

type TypingEvent struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}
