// Package service provides the business logic layer for the Tesla Road Trip Game.
//
// The service package implements:
//   - Multi-session game management
//   - Configuration management and loading
//   - Move processing and validation
//   - Session lifecycle management
//   - Move history tracking
//
// Core Interfaces:
//
// GameService is the main service interface providing high-level game operations.
// SessionManager handles session creation, retrieval, and lifecycle.
// ConfigManager manages game configuration loading and validation.
//
// Architecture:
//
// The service layer sits between the transport layer (HTTP/WebSocket/MCP) and
// the game engine, providing session isolation, configuration management, and
// business logic orchestration. Each session maintains its own game engine
// instance with independent state.
//
// Usage:
//
//	sessionMgr := session.NewManager()
//	configMgr := config.NewManager("configs")
//	gameService := service.NewGameService(sessionMgr, configMgr)
//
//	// Create a new session
//	sessionInfo, err := gameService.CreateSession(ctx, "easy")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Execute moves
//	events, err := gameService.Move(ctx, sessionInfo.ID, "up", false)
//
// Session Management:
//
// Sessions are identified by unique 4-character IDs and maintain independent
// game state. Multiple sessions can run concurrently with different
// configurations. Sessions track creation time, last access time, and move
// history for analytics and debugging.
package service
