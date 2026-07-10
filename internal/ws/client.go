package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
)

type Client struct {
	conn   *websocket.Conn
	userID string
	send   chan []byte
	hub    *Hub
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
		var incoming IncomingMessage
		if err = json.Unmarshal(msg, &incoming); err != nil {
			log.Println("error unmarshaling json:", err)
			continue
		}

		senderID, err := uuid.Parse(c.userID)
		if err != nil {
			log.Println("invalid sender id:", err)
			continue
		}

		isParticipant, err := c.hub.store.IsParticipant(context.Background(), db.IsParticipantParams{
			ConversationID: incoming.ConversationID,
			UserID:         senderID,
		})

		if err != nil {
			log.Println("error checking participant:", err)
			continue
		}
		if !isParticipant {
			log.Printf("user %s is not a participant of conversation %s", c.userID, incoming.ConversationID)
			continue
		}

		message, err := c.hub.store.CreateMessage(context.Background(), db.CreateMessageParams{
			ConversationID: incoming.ConversationID,
			SenderID:       senderID,
			Content:        incoming.Content,
		})
		if err != nil {
			log.Println("error creating message:", err)
			continue
		}

		c.hub.broadcast <- Message{
			ConversationID: message.ConversationID,
			SenderID:       message.SenderID,
			Content:        []byte(message.Content),
		}
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
