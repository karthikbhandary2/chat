package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
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
		message := &Message{
			RecipientID: incoming.RecipientID,
			Content:     []byte(incoming.Content),
		}
		c.hub.broadcast <- *message
		log.Printf("message from %s to %s: %s", c.userID, incoming.RecipientID, incoming.Content)
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
