package ws

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        []byte
}

type Hub struct {
	clients     map[string]*Client
	register    chan *Client
	unregister  chan *Client
	broadcast   chan Message
	store       db.Store
	redisClient *redis.Client
}

func NewHub(store db.Store, redisClient *redis.Client) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan Message),
		store:       store,
		redisClient: redisClient,
	}
}

func presenceKey(userID string) string {
	return fmt.Sprintf("presence:%s", userID)
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.userID] = client
			if err := h.redisClient.Set(context.Background(), presenceKey(client.userID), 1, 60*time.Second).Err(); err != nil {
				log.Println("error setting presence:", err)
			}

		case client := <-h.unregister:
			delete(h.clients, client.userID)
			if err := h.redisClient.Del(context.Background(), presenceKey(client.userID)).Err(); err != nil {
				log.Println("error deleting presence:", err)
			}

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
						log.Printf("dropping message for slow client %s", participantID)
					}
				}
			}
		}
	}
}
