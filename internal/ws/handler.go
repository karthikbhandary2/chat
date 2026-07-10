package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/karthikbhandary2/chat/internal/auth"
)

type Server struct {
	hub *Hub
}

func NewServer(hub *Hub) *Server {
	return &Server{hub: hub}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) HandleConnections(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}

	client := &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan []byte),
		hub:    s.hub,
	}

	s.hub.register <- client

	go client.readPump()
	go client.writePump()
}
