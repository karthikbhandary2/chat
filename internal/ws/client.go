package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"golang.org/x/time/rate"
	"log/slog"
)

type Client struct {
	conn    *websocket.Conn
	userID  string
	send    chan []byte
	hub     *Hub
	limiter *rate.Limiter
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		close(c.send)
	}()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		var envelope Envelope
		if err := json.Unmarshal(msg, &envelope); err != nil {
			log.Println("error unmarshaling envelope:", err)
			continue
		}

		switch envelope.Type {
		case "message":
			c.handleChatMessage(msg)
		case "typing":
			c.handleTypingEvent(msg)
		default:
			log.Println("unknown message type:", envelope.Type)
		}
	}
}

func (c *Client) handleChatMessage(raw []byte) {

	if !c.limiter.Allow() {
		log.Printf("rate limit exceeded for user %s", c.userID)
		return
	}

	var incoming IncomingMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		log.Println("error unmarshaling json:", err)
		return
	}

	if len(incoming.Content) == 0 || len(incoming.Content) > 2000 {
		log.Printf("invalid message length from user %s", c.userID)
		return
	}

	senderID, err := uuid.Parse(c.userID)
	if err != nil {
		log.Println("invalid sender id:", err)
		return
	}

	isParticipant, err := c.hub.store.IsParticipant(context.Background(), db.IsParticipantParams{
		ConversationID: incoming.ConversationID,
		UserID:         senderID,
	})
	if err != nil {
		log.Println("error checking participant:", err)
		return
	}
	if !isParticipant {
		log.Printf("user %s is not a participant of conversation %s", c.userID, incoming.ConversationID)
		return
	}

	message, err := c.hub.store.CreateMessage(context.Background(), db.CreateMessageParams{
		ConversationID: incoming.ConversationID,
		SenderID:       senderID,
		Content:        incoming.Content,
	})
	if err != nil {
		slog.Error("error creating message", "user_id", c.userID, "error", err)
		return
	}

	c.hub.broadcast <- Message{
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Content:        []byte(message.Content),
	}
}

func (c *Client) handleTypingEvent(raw []byte) {
	var typing TypingEvent
	if err := json.Unmarshal(raw, &typing); err != nil {
		log.Println("error unmarshaling typing event:", err)
		return
	}

	senderID, err := uuid.Parse(c.userID)
	if err != nil {
		log.Println("invalid sender id:", err)
		return
	}

	outgoing, err := json.Marshal(struct {
		Type           string    `json:"type"`
		ConversationID uuid.UUID `json:"conversation_id"`
		UserID         string    `json:"user_id"`
	}{
		Type:           "typing",
		ConversationID: typing.ConversationID,
		UserID:         c.userID,
	})
	if err != nil {
		log.Println("error marshaling typing event:", err)
		return
	}

	c.hub.broadcast <- Message{
		ConversationID: typing.ConversationID,
		SenderID:       senderID,
		Content:        outgoing,
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// channel was closed — send a close frame and stop
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("write error:", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("ping error:", err)
				return
			}
		}
	}
}
