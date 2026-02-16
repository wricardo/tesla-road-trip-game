package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wricardo/mcp-training/statefullgame/game/engine"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Configure this for production
		return true
	},
}

// Message represents a WebSocket message
type Message struct {
	SessionID string            `json:"session_id"`
	GameState *engine.GameState `json:"game_state,omitempty"`
	Event     string            `json:"event,omitempty"`
	Data      interface{}       `json:"data,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	sessionID string
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by session ID
	sessions map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		sessions:   make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		sessionID: sessionID,
	}

	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// BroadcastToSession sends a game state update to all clients in a session
func (h *Hub) BroadcastToSession(sessionID string, state *engine.GameState) {
	message := &Message{
		SessionID: sessionID,
		GameState: state,
		Event:     "state_update",
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	// Send to all clients in this session
	if clients, ok := h.sessions[sessionID]; ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				// Client's send channel is full, close it
				h.unregisterClient(client)
			}
		}
	}
}

// BroadcastEvent sends a custom event to all clients in a session
func (h *Hub) BroadcastEvent(sessionID string, event string, data interface{}) {
	message := &Message{
		SessionID: sessionID,
		Event:     event,
		Data:      data,
	}

	h.broadcast <- message
}

// registerClient adds a client to a session
func (h *Hub) registerClient(client *Client) {
	if h.sessions[client.sessionID] == nil {
		h.sessions[client.sessionID] = make(map[*Client]bool)
	}
	h.sessions[client.sessionID][client] = true

	log.Printf("Client registered for session %s (total clients: %d)",
		client.sessionID, len(h.sessions[client.sessionID]))
}

// unregisterClient removes a client from a session
func (h *Hub) unregisterClient(client *Client) {
	if clients, ok := h.sessions[client.sessionID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			// Clean up empty sessions
			if len(clients) == 0 {
				delete(h.sessions, client.sessionID)
			}

			log.Printf("Client unregistered from session %s (remaining clients: %d)",
				client.sessionID, len(clients))
		}
	}
}

// broadcastMessage sends a message to all clients in a session
func (h *Hub) broadcastMessage(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	if clients, ok := h.sessions[message.SessionID]; ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				h.unregisterClient(client)
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// We don't process incoming messages from clients currently
		// Just keep the connection alive
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
