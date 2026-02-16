// Package api provides HTTP REST API handlers for the Tesla Road Trip Game.
//
// The api package implements:
//   - RESTful endpoints for game operations
//   - Session management endpoints
//   - Configuration listing and selection
//   - Save/load game functionality
//   - WebSocket upgrade handling
//   - Static file serving
//
// Endpoints:
//
// Game Operations:
//   - GET /api - Get current game state
//   - POST /api - Execute action (move, reset, save, load)
//   - GET /api/history - Get move history with pagination
//
// Session Management:
//   - POST /api/sessions - Create new session
//   - GET /api/sessions - List all sessions
//   - GET /api/sessions/{id} - Get specific session
//
// Configuration:
//   - GET /api/configs - List available configurations
//
// Save/Load:
//   - GET /api/saves - List saved games
//   - POST /api with action:"save" - Save current game
//   - POST /api with action:"load" - Load saved game
//
// Request/Response Format:
//
// All endpoints accept and return JSON. Session-specific operations
// support optional sessionId query parameter for targeting specific sessions.
//
// Actions are sent as POST with JSON body:
//
//	{
//	  "action": "up|down|left|right|reset|save|load",
//	  "actions": ["up", "down", "left"], // for bulk moves
//	  "reset": true|false,                // optional reset before move
//	  "saveFile": "save_123.json"         // for load action
//	}
//
// Usage:
//
//	handler := api.NewHandler(gameService)
//	http.HandleFunc("/api", handler.HandleAPI)
//	http.HandleFunc("/api/sessions", handler.HandleSessions)
//	http.HandleFunc("/api/configs", handler.HandleConfigs)
//
// Error Handling:
//
// Errors are returned as JSON with appropriate HTTP status codes:
//
//	{
//	  "error": "error message",
//	  "code": 400
//	}
package api

//
// Enriched Responses (Move and Bulk Move)
//
// Move (POST /api/sessions/{id}/move)
//   Response:
//     - step: { dir, from{x,y}, to{x,y}, tile_char, tile_type, battery_before, battery_after, success }
//     - attempted_to: { x, y, tile_char, tile_type, passable } // present when blocked
//     - game_state additions:
//         local_view_3x3: ["...","...","..."] // 3x3 characters around player (T centered)
//         battery_risk: "SAFE|LOW|CAUTION|DANGER|CRITICAL|WARNING"
//
// Bulk Move (POST /api/sessions/{id}/bulk-move)
//   Response:
//     - requested_moves, moves_executed
//     - stopped_reason (text), stop_reason_code (enum), stopped_on_move (1-based), truncated, limit
//     - steps: [{ idx, dir, from, to, tile_char, tile_type, battery_before, battery_after, success, charged?, park?, victory? }]
//     - attempted_to: failed target cell on first block
//     - start_pos, end_pos, start_battery, end_battery, score_delta
//     - possible_moves: ["up","right"], local_view_3x3, battery_risk
