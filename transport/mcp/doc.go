// Package mcp provides Model Context Protocol server implementation for the Tesla Road Trip Game.
//
// The mcp package implements:
//   - MCP server for AI agent integration
//   - Tool definitions for game operations
//   - Session-aware command execution
//   - Streamable HTTP transport (stdio is not supported)
//
// MCP Tools:
//
// The package exposes the following tools for AI agents:
//   - game_state: Get current game state (optional full grid for non-fog sessions)
//   - move: Execute single directional movement
//   - bulk_move: Execute up to 50 moves in sequence
//   - reset_game: Reset game to initial state
//   - move_history: Retrieve move history with pagination
//   - create_session: Create new game session (map, fog, move delay)
//   - get_session: Get specific session details
//   - update_session: Update session display name
//   - delete_session: Delete a session
//   - list_sessions: List sessions with sort/order/limit
//   - unified_sessions: List sessions with game state, optionally per map
//   - list_maps: List available game maps
//   - get_map: Get full map definition
//   - validate_map: Validate a map definition without saving
//   - create_map, update_map, delete_map: Admin map management
//
// Transport:
//
// The server is exposed over HTTP via Handler(), mounted at /mcp.
//
// Session Management:
//
// Session-scoped tools require a session_id (update_session and
// delete_session take id). AI agents can manage multiple concurrent game
// sessions independently.
//
// Usage:
//
//	hub := websocket.NewHub()
//	go hub.Run()
//	server := mcp.NewServer(gameService, hub)
//	http.Handle("/mcp", server.Handler())
//
// AI Integration:
//
// The MCP interface enables AI agents to:
//   - Autonomously play the game
//   - Develop and test strategies
//   - Analyze game states and make decisions
//   - Manage multiple game sessions
//   - Learn from move history
package mcp
