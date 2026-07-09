package ws

type Message struct {
	RecipientID string
	Content     []byte
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
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
			client, ok := h.clients[msg.RecipientID]
			if ok {
				client.send <- msg.Content
			}
		}
	}
}
