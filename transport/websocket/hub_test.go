package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wricardo/mcp-training/statefullgame/game/engine"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.sessions == nil {
		t.Error("Hub sessions map is nil")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel is nil")
	}

	if hub.register == nil {
		t.Error("Hub register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel is nil")
	}
}

func TestHubRegisterClient(t *testing.T) {
	hub := NewHub()

	// Create a mock client
	client := &Client{
		hub:       hub,
		sessionID: "test-session",
		send:      make(chan []byte, 256),
	}

	// Register the client
	hub.registerClient(client)

	// Check if session was created
	if _, exists := hub.sessions["test-session"]; !exists {
		t.Error("Session was not created")
	}

	// Check if client was added to session
	if !hub.sessions["test-session"][client] {
		t.Error("Client was not registered in session")
	}

	// Check session count
	if len(hub.sessions["test-session"]) != 1 {
		t.Errorf("Expected 1 client in session, got %d", len(hub.sessions["test-session"]))
	}
}

func TestHubUnregisterClient(t *testing.T) {
	hub := NewHub()

	client := &Client{
		hub:       hub,
		sessionID: "test-session",
		send:      make(chan []byte, 256),
	}

	// Register then unregister
	hub.registerClient(client)
	hub.unregisterClient(client)

	// Check if session was cleaned up
	if _, exists := hub.sessions["test-session"]; exists {
		t.Error("Session should have been cleaned up after last client unregistered")
	}
}

func TestHubMultipleClientsInSession(t *testing.T) {
	hub := NewHub()
	sessionID := "multi-client-session"

	// Create multiple clients for the same session
	client1 := &Client{
		hub:       hub,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
	}
	client2 := &Client{
		hub:       hub,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
	}

	// Register both clients
	hub.registerClient(client1)
	hub.registerClient(client2)

	// Check session has 2 clients
	if len(hub.sessions[sessionID]) != 2 {
		t.Errorf("Expected 2 clients in session, got %d", len(hub.sessions[sessionID]))
	}

	// Unregister one client
	hub.unregisterClient(client1)

	// Session should still exist with 1 client
	if len(hub.sessions[sessionID]) != 1 {
		t.Errorf("Expected 1 client remaining in session, got %d", len(hub.sessions[sessionID]))
	}

	// Check the right client remains
	if !hub.sessions[sessionID][client2] {
		t.Error("client2 should still be registered")
	}
}

func TestHubBroadcastToSession(t *testing.T) {
	hub := NewHub()
	sessionID := "broadcast-test"

	// Create a test client
	client := &Client{
		hub:       hub,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
	}

	hub.registerClient(client)

	// Create test game state
	gameState := &engine.GameState{
		PlayerPos: engine.Position{X: 5, Y: 3},
		Battery:   80,
		Score:     100,
		GameOver:  false,
	}

	// Broadcast to the session
	hub.BroadcastToSession(sessionID, gameState)

	// Check if message was sent to client
	select {
	case data := <-client.send:
		var message Message
		err := json.Unmarshal(data, &message)
		if err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if message.SessionID != sessionID {
			t.Errorf("Expected sessionID %s, got %s", sessionID, message.SessionID)
		}

		if message.Event != "state_update" {
			t.Errorf("Expected event 'state_update', got %s", message.Event)
		}

		if message.GameState.PlayerPos.X != 5 || message.GameState.PlayerPos.Y != 3 {
			t.Error("GameState not correctly transmitted")
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("No message received within timeout")
	}
}

func TestHubBroadcastEvent(t *testing.T) {
	hub := NewHub()
	done := make(chan bool)

	// Start hub in goroutine
	go func() {
		for {
			select {
			case message := <-hub.broadcast:
				// Verify the broadcast message
				if message.SessionID != "event-test" {
					t.Errorf("Expected sessionID 'event-test', got %s", message.SessionID)
				}
				if message.Event != "custom-event" {
					t.Errorf("Expected event 'custom-event', got %s", message.Event)
				}
				if message.Data != "test-data" {
					t.Errorf("Expected data 'test-data', got %v", message.Data)
				}
				done <- true
				return
			case <-time.After(100 * time.Millisecond):
				t.Error("No broadcast message received within timeout")
				done <- false
				return
			}
		}
	}()

	// Send broadcast event
	hub.BroadcastEvent("event-test", "custom-event", "test-data")

	// Wait for verification
	<-done
}

func TestWebSocketUpgrade(t *testing.T) {
	hub := NewHub()

	// Start hub in background
	go hub.Run()

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			sessionID = "default"
		}
		hub.ServeWS(w, r, sessionID)
	}))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?sessionId=ws-test"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Give some time for registration
	time.Sleep(50 * time.Millisecond)

	// Check if client was registered
	if len(hub.sessions["ws-test"]) != 1 {
		t.Errorf("Expected 1 client in session, got %d", len(hub.sessions["ws-test"]))
	}

	// Close connection
	conn.Close()

	// Give some time for unregistration
	time.Sleep(10 * time.Millisecond)

	// Check if client was unregistered and session cleaned up
	if _, exists := hub.sessions["ws-test"]; exists {
		t.Error("Session should have been cleaned up after WebSocket close")
	}
}

func TestWebSocketMessageReceive(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			sessionID = "default"
		}
		hub.ServeWS(w, r, sessionID)
	}))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?sessionId=msg-test"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Give time for connection to establish
	time.Sleep(10 * time.Millisecond)

	// Create and broadcast a test game state
	gameState := &engine.GameState{
		PlayerPos: engine.Position{X: 10, Y: 15},
		Battery:   50,
		Score:     200,
		GameOver:  false,
	}

	hub.BroadcastToSession("msg-test", gameState)

	// Read message from WebSocket
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, messageData, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WebSocket message: %v", err)
	}

	// Parse the message
	var message Message
	err = json.Unmarshal(messageData, &message)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify message content
	if message.SessionID != "msg-test" {
		t.Errorf("Expected sessionID 'msg-test', got %s", message.SessionID)
	}

	if message.GameState.PlayerPos.X != 10 || message.GameState.PlayerPos.Y != 15 {
		t.Error("GameState position not correctly received")
	}

	if message.GameState.Battery != 50 || message.GameState.Score != 200 {
		t.Error("GameState battery/score not correctly received")
	}
}
