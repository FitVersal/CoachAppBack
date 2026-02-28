package chatws

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	websocket "github.com/gofiber/contrib/websocket"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

type Hub struct {
	clients    map[string]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	send   chan []byte
}

type sender interface {
	SendMessage(
		ctx context.Context,
		actorID int64,
		role string,
		conversationID int64,
		content string,
	) (*services.ChatDelivery, error)
}

type Message struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	RecipientID    string `json:"recipient_id,omitempty"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 64),
	}
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 32),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			set, ok := h.clients[client.userID]
			if !ok {
				set = make(map[*Client]struct{})
				h.clients[client.userID] = set
			}
			set[client] = struct{}{}
		case client := <-h.unregister:
			set, ok := h.clients[client.userID]
			if !ok {
				continue
			}
			if _, exists := set[client]; exists {
				delete(set, client)
				close(client.send)
			}
			if len(set) == 0 {
				delete(h.clients, client.userID)
			}
		case message := <-h.broadcast:
			h.deliver(message)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) deliver(message *Message) {
	encoded, err := encodeMessage(message)
	if err != nil {
		log.Printf("chat hub encode message: %v", err)
		return
	}

	h.sendToUser(message.SenderID, encoded)
	if message.RecipientID != "" && message.RecipientID != message.SenderID {
		h.sendToUser(message.RecipientID, encoded)
	}
}

func (h *Hub) sendToUser(userID string, payload []byte) {
	set, ok := h.clients[userID]
	if !ok {
		return
	}

	for client := range set {
		select {
		case client.send <- payload:
		default:
			delete(set, client)
			close(client.send)
		}
	}
	if len(set) == 0 {
		delete(h.clients, userID)
	}
}

func encodeMessage(message *Message) ([]byte, error) {
	return json.Marshal(message)
}

func (c *Client) ReadPump(service sender, role string) {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	actorID, err := strconv.ParseInt(c.userID, 10, 64)
	if err != nil {
		writeError(c, "invalid user")
		return
	}

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var incoming struct {
			Type           string `json:"type"`
			ConversationID string `json:"conversation_id"`
			Content        string `json:"content"`
		}
		if err := json.Unmarshal(payload, &incoming); err != nil {
			writeError(c, "invalid message payload")
			continue
		}
		if incoming.Type != "message" {
			writeError(c, "unsupported message type")
			continue
		}

		conversationID, err := strconv.ParseInt(incoming.ConversationID, 10, 64)
		if err != nil || conversationID <= 0 {
			writeError(c, "invalid conversation id")
			continue
		}

		delivery, err := service.SendMessage(
			context.Background(),
			actorID,
			role,
			conversationID,
			incoming.Content,
		)
		if err != nil {
			writeError(c, "failed to send message")
			continue
		}

		c.hub.broadcast <- &Message{
			Type:           "message",
			ConversationID: strconv.FormatInt(delivery.Message.ConversationID, 10),
			SenderID:       strconv.FormatInt(delivery.Message.SenderID, 10),
			RecipientID:    strconv.FormatInt(delivery.RecipientID, 10),
			Content:        delivery.Message.Content,
			Timestamp:      services.FormatChatTimestamp(delivery.Message.CreatedAt),
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for payload := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}

func writeError(client *Client, message string) {
	payload, err := json.Marshal(Message{
		Type:      "error",
		Content:   message,
		Timestamp: services.FormatChatTimestamp(time.Now().UTC()),
	})
	if err != nil {
		return
	}
	select {
	case client.send <- payload:
	default:
		client.hub.Unregister(client)
	}
}
