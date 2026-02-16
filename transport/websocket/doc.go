// Package websocket provides WebSocket transport for the Tesla Road Trip Game.
//
// The websocket package implements:
//   - Real-time bidirectional communication
//   - Session-aware WebSocket connections
//   - Automatic state broadcasting on changes
//   - Connection lifecycle management
//   - Message routing and handling
//
// Architecture:
//
// The package uses a hub-and-spoke model where a central Hub manages all
// WebSocket connections. Each client connection is handled by a dedicated
// goroutine that manages reading, writing, and cleanup.
//
// Message Protocol:
//
// Messages are JSON-encoded with the following structure:
//   - Incoming: {action: "move", direction: "up", sessionId: "abc1"}
//   - Outgoing: Complete GameState JSON after each state change
//
// Session Integration:
//
// WebSocket connections are session-aware. Clients specify their session ID
// via query parameter (?sessionId=abc1) when establishing the connection.
// State updates are broadcast only to clients connected to the same session.
//
// Usage:
//
//	hub := websocket.NewHub()
//	go hub.Run()
//
//	handler := websocket.NewHandler(hub, gameService)
//	http.HandleFunc("/ws", handler.HandleWebSocket)
//
// Connection Lifecycle:
//
// 1. Client connects with session ID
// 2. Connection registered with hub
// 3. Initial state sent to client
// 4. Client sends actions, receives state updates
// 5. Disconnection triggers cleanup
//
// Concurrency:
//
// The hub and client handlers are designed for concurrent operation.
// Multiple clients can connect, disconnect, and send messages simultaneously
// without blocking each other.
package websocket
