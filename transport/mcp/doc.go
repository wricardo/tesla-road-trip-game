// Package mcp provides Model Context Protocol server implementation for the Tesla Road Trip Game.
//
// The mcp package implements:
//   - MCP server for AI agent integration
//   - Tool definitions for game operations
//   - Session-aware command execution
//   - Stdio and HTTP transport modes
//
// MCP Tools:
//
// The package exposes the following tools for AI agents:
//   - game_state: Get current game state with grid visualization
//   - move: Execute single directional movement
//   - bulk_move: Execute multiple moves in sequence
//   - reset_game: Reset game to initial state
//   - move_history: Retrieve move history with pagination
//   - create_session: Create new game session with config selection
//   - get_session: Get specific session details
//   - list_sessions: List all active sessions
//   - list_configs: List available game configurations
//
// Transport Modes:
//
// The server supports two transport modes:
//   - Stdio: Direct stdio communication for local MCP clients
//   - HTTP: HTTP endpoint for remote MCP integration
//
// Session Management:
//
// All game tools support optional session_id parameter for multi-session
// gameplay. Without session_id, operations target the default session.
// AI agents can manage multiple concurrent game sessions independently.
//
// Usage:
//
//	// Stdio mode
//	server := mcp.NewServer(gameService)
//	server.RunStdio()
//
//	// HTTP mode
//	handler := mcp.NewHTTPHandler(gameService)
//	http.HandleFunc("/mcp", handler.Handle)
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
