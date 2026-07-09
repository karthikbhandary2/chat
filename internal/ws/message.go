package ws

type IncomingMessage struct {
	RecipientID string `json:"recipient_id"`
	Content     string `json:"content"`
}