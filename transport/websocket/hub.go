package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wricardo/tesla-road-trip-game/game/engine"
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

// newUpgraderFromEnv builds a WebSocket upgrader whose CheckOrigin respects
// the ALLOWED_ORIGINS environment variable.
//
// ALLOWED_ORIGINS: comma-separated list of allowed origins (e.g.
// "http://localhost:5173,https://myapp.example.com").
// Empty or "*" → allow all origins (development default).
func newUpgraderFromEnv() websocket.Upgrader {
	var allowed []string
	raw := os.Getenv("ALLOWED_ORIGINS")
	if raw != "" && raw != "*" {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowed = append(allowed, o)
			}
		}
	}
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     makeCheckOrigin(allowed),
	}
}

// makeCheckOrigin returns a CheckOrigin func that allows all origins when
// allowed is empty, or performs exact-match against the allowed list otherwise.
// Non-browser requests (no Origin header) are always allowed.
func makeCheckOrigin(allowed []string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if len(allowed) == 0 {
			return true
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser client, allow
		}
		for _, a := range allowed {
			if a == origin {
				return true
			}
		}
		log.Printf("WebSocket: rejected origin %q (not in ALLOWED_ORIGINS)", origin)
		return false
	}
}

// DefaultUpgrader returns a WebSocket upgrader configured from ALLOWED_ORIGINS env.
func DefaultUpgrader() websocket.Upgrader {
	return newUpgraderFromEnv()
}

// Message represents a WebSocket message
type Message struct {
	SessionID string            `json:"session_id"`
	GameState *WSGameState      `json:"game_state,omitempty"`
	Event     string            `json:"event,omitempty"`
	Data      interface{}       `json:"data,omitempty"`
}

// WSGameState is the websocket-safe game state payload.
// It intentionally excludes heavy fields (grid, move_history).
type WSGameState struct {
	PlayerPos         engine.Position                 `json:"player_pos"`
	Battery           int                             `json:"battery"`
	MaxBattery        int                             `json:"max_battery"`
	Score             int                             `json:"score"`
	VisitedParks      map[string]bool                 `json:"visited_parks"`
	Message           string                          `json:"message"`
	GameOver          bool                            `json:"game_over"`
	Victory           bool                            `json:"victory"`
	MapName           string                          `json:"map_name"`
	TotalMoves        int                             `json:"total_moves"`
	LocalView         []engine.SurroundingCell        `json:"local_view,omitempty"`
	CurrentMoves      []engine.MoveHistoryEntry       `json:"current_moves"`
	CurrentMovesCount int                             `json:"current_moves_count"`
	LocalView3x3      []string                        `json:"local_view_3x3,omitempty"`
	BatteryRisk       string                          `json:"battery_risk,omitempty"`
}

func newWSGameState(state *engine.GameState) *WSGameState {
	if state == nil {
		return nil
	}
	return &WSGameState{
		PlayerPos:         state.PlayerPos,
		Battery:           state.Battery,
		MaxBattery:        state.MaxBattery,
		Score:             state.Score,
		VisitedParks:      state.VisitedParks,
		Message:           state.Message,
		GameOver:          state.GameOver,
		Victory:           state.Victory,
		MapName:           state.MapName,
		TotalMoves:        state.TotalMoves,
		LocalView:         state.LocalView,
		CurrentMoves:      state.CurrentMoves,
		CurrentMovesCount: state.CurrentMovesCount,
		LocalView3x3:      state.LocalView3x3,
		BatteryRisk:       state.BatteryRisk,
	}
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
	upgrader websocket.Upgrader

	// Registered clients by session ID. Protected by sessionsMu because
	// BroadcastToSession can be called directly by HTTP/GraphQL handlers while
	// Run concurrently registers/unregisters clients.
	sessionsMu sync.RWMutex
	sessions   map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// GraphQL subscription channels per session
	mu          sync.RWMutex
	sessionSubs map[string]map[chan *engine.GameState]bool
	lobbySubs   map[chan *engine.GameState]bool
}

// NewHub creates a new WebSocket hub with origin policy from ALLOWED_ORIGINS env.
func NewHub() *Hub {
	return &Hub{
		upgrader:    newUpgraderFromEnv(),
		sessions:    make(map[string]map[*Client]bool),
		broadcast:   make(chan *Message),
		sessionSubs: make(map[string]map[chan *engine.GameState]bool),
		lobbySubs:   make(map[chan *engine.GameState]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
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
	conn, err := h.upgrader.Upgrade(w, r, nil)
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
		GameState: newWSGameState(state),
		Event:     "state_update",
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	// Send to all WebSocket clients in this session.
	// Hold sessionsMu while sending so unregisterClient cannot close a send
	// channel concurrently with this non-blocking send.
	var stale []*Client
	h.sessionsMu.Lock()
	if clients, ok := h.sessions[sessionID]; ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				stale = append(stale, client)
			}
		}
	}
	for _, client := range stale {
		h.unregisterClientLocked(client)
	}
	h.sessionsMu.Unlock()

	// Fan out to GraphQL subscription channels
	h.mu.RLock()
	defer h.mu.RUnlock()
	if subs, ok := h.sessionSubs[sessionID]; ok {
		for ch := range subs {
			select {
			case ch <- state:
			default:
			}
		}
	}
	for ch := range h.lobbySubs {
		select {
		case ch <- state:
		default:
		}
	}
}

// SubscribeSession returns a channel that receives GameState on every change for sessionID.
// Cancel ctx to unsubscribe.
func (h *Hub) SubscribeSession(ctx context.Context, sessionID string) <-chan *engine.GameState {
	ch := make(chan *engine.GameState, 8)
	h.mu.Lock()
	if h.sessionSubs[sessionID] == nil {
		h.sessionSubs[sessionID] = make(map[chan *engine.GameState]bool)
	}
	h.sessionSubs[sessionID][ch] = true
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.mu.Lock()
		if subs, ok := h.sessionSubs[sessionID]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.sessionSubs, sessionID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}()
	return ch
}

// SubscribeLobby returns a channel that receives any GameState change across all sessions.
// Cancel ctx to unsubscribe.
func (h *Hub) SubscribeLobby(ctx context.Context) <-chan *engine.GameState {
	ch := make(chan *engine.GameState, 32)
	h.mu.Lock()
	h.lobbySubs[ch] = true
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.lobbySubs, ch)
		h.mu.Unlock()
		close(ch)
	}()
	return ch
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
	h.sessionsMu.Lock()
	defer h.sessionsMu.Unlock()

	if h.sessions[client.sessionID] == nil {
		h.sessions[client.sessionID] = make(map[*Client]bool)
	}
	h.sessions[client.sessionID][client] = true

	log.Printf("Client registered for session %s (total clients: %d)",
		client.sessionID, len(h.sessions[client.sessionID]))
}

// unregisterClient removes a client from a session
func (h *Hub) unregisterClient(client *Client) {
	h.sessionsMu.Lock()
	defer h.sessionsMu.Unlock()
	h.unregisterClientLocked(client)
}

// unregisterClientLocked removes a client from a session. h.sessionsMu must be held.
func (h *Hub) unregisterClientLocked(client *Client) {
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

// SessionClientCount returns the number of connected clients for a session.
func (h *Hub) SessionClientCount(sessionID string) int {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()
	return len(h.sessions[sessionID])
}

// HasSessionClients reports whether the hub tracks any clients for a session.
func (h *Hub) HasSessionClients(sessionID string) bool {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()
	_, ok := h.sessions[sessionID]
	return ok
}

// broadcastMessage sends a message to all clients in a session
func (h *Hub) broadcastMessage(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	var stale []*Client
	h.sessionsMu.Lock()
	if clients, ok := h.sessions[message.SessionID]; ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				stale = append(stale, client)
			}
		}
	}
	for _, client := range stale {
		h.unregisterClientLocked(client)
	}
	h.sessionsMu.Unlock()
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
