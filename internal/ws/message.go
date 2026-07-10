package ws

import "github.com/google/uuid"

type IncomingMessage struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Content        string    `json:"content"`
}
