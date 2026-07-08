package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/karthikbhandary2/chat/internal/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
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

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}
	defer ws.Close()

	// userID is now available here for registering this connection with the hub later
	_ = userID
	for {
		messageType, msg, err := ws.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		if err := ws.WriteMessage(messageType, msg); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
