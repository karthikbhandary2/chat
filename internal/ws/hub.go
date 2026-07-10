package ws

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
)

type Message struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        []byte
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	store      db.Store
}

func NewHub(store db.Store) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
		store:      store,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.userID] = client
		case client := <-h.unregister:
			delete(h.clients, client.userID)
		case msg := <-h.broadcast:
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)

			participantIDs, err := h.store.GetConversationParticipants(ctx, msg.ConversationID)
			cancel()
			if err != nil {
				log.Println("error fetching participants:", err)
				continue
			}

			for _, participantID := range participantIDs {
				client, ok := h.clients[participantID.String()]
				if ok {
					select {
					case client.send <- msg.Content:
					default:
						log.Printf("droppingm message for slow client %s", participantID)
					}
				}
			}
		}
	}
}
