// Package session provides session management for the Tesla Road Trip Game.
//
// The session package implements:
//   - Thread-safe session storage and retrieval
//   - Unique session ID generation
//   - Session lifecycle management
//   - Concurrent access control
//   - Session cleanup and expiration
//
// Core Types:
//
// Manager is the main session manager that handles all session operations.
// Session represents an individual game session with its own engine instance
// and metadata like creation time and last access time.
//
// Session Identifiers:
//
// Sessions use 4-character alphanumeric IDs for easy reference. The manager
// ensures IDs are unique and provides collision-resistant generation using
// cryptographic randomness.
//
// Concurrency:
//
// The session manager is thread-safe and supports concurrent operations.
// Multiple goroutines can safely create, retrieve, and modify different
// sessions simultaneously. Internal locking ensures data consistency.
//
// Usage:
//
//	manager := session.NewManager()
//
//	// Create a new session
//	sess, err := manager.Create("", config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Retrieve existing session
//	sess, err = manager.Get(sessionID)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// List all active sessions
//	sessions := manager.List()
//
// Cleanup:
//
// Sessions can be explicitly deleted or may expire based on inactivity.
// The manager provides cleanup methods for removing stale sessions and
// freeing resources.
package session
