package session

import (
	"os"
	"testing"
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/config"
)

func TestManagerWithPersistence(t *testing.T) {
	// Create temporary directory for test sessions
	tempDir, err := os.MkdirTemp("", "manager_persistence_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config manager
	configManager, err := config.NewManager("../../configs")
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	// Create persistence layer
	persistence, err := NewFilePersistence(tempDir, configManager)
	if err != nil {
		t.Fatalf("Failed to create file persistence: %v", err)
	}

	// Create manager with persistence
	manager := NewManagerWithPersistence(persistence)

	t.Run("Create Session Auto-Saves", func(t *testing.T) {
		gameConfig := configManager.GetDefault()
		session, err := manager.Create("auto1", gameConfig)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Verify session was auto-saved
		if !persistence.Exists(session.ID) {
			t.Error("Session should be auto-saved on creation")
		}

		// Verify we can load it directly from persistence
		loadedSession, err := persistence.Load(session.ID)
		if err != nil {
			t.Fatalf("Failed to load auto-saved session: %v", err)
		}

		if loadedSession.ID != session.ID {
			t.Errorf("Expected ID %s, got %s", session.ID, loadedSession.ID)
		}
	})

	t.Run("Get Session Loads from Persistence", func(t *testing.T) {
		// Create new manager (no in-memory sessions)
		manager2 := NewManagerWithPersistence(persistence)

		// Try to get session that exists only in persistence
		session, err := manager2.Get("auto1")
		if err != nil {
			t.Fatalf("Failed to get session from persistence: %v", err)
		}

		if session.ID != "auto1" {
			t.Errorf("Expected ID auto1, got %s", session.ID)
		}

		// Verify it's now in memory too
		session2, err := manager2.Get("auto1")
		if err != nil {
			t.Fatalf("Failed to get session from memory: %v", err)
		}

		if session2.ID != session.ID {
			t.Error("Session should be cached in memory after loading from persistence")
		}
	})

	t.Run("Save Method Persists Changes", func(t *testing.T) {
		// Get session and make changes
		session, err := manager.Get("auto1")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		// Make a move to change state
		originalPos := session.Engine.GetPlayerPosition()
		success := session.Engine.Move("right")
		if !success {
			// Try different directions
			success = session.Engine.Move("down") || session.Engine.Move("left") || session.Engine.Move("up")
		}
		if !success {
			t.Skip("Cannot test persistence without successful move")
		}

		// Save manually
		err = manager.Save("auto1")
		if err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Create new manager and load session
		manager3 := NewManagerWithPersistence(persistence)
		loadedSession, err := manager3.Get("auto1")
		if err != nil {
			t.Fatalf("Failed to load session after manual save: %v", err)
		}

		// Verify changes were persisted
		if loadedSession.Engine.GetPlayerPosition() == originalPos {
			t.Error("Player position changes should be persisted")
		}

		if len(loadedSession.Engine.GetMoveHistory()) == 0 {
			t.Error("Move history should be persisted")
		}
	})

	t.Run("Delete Removes from Persistence", func(t *testing.T) {
		// Create session
		gameConfig := configManager.GetDefault()
		session, err := manager.Create("delete_test", gameConfig)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Verify it exists in persistence
		if !persistence.Exists(session.ID) {
			t.Error("Session should exist in persistence")
		}

		// Delete session
		err = manager.Delete(session.ID)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Verify it's gone from persistence
		if persistence.Exists(session.ID) {
			t.Error("Session should be removed from persistence on delete")
		}

		// Verify we can't get it anymore
		_, err = manager.Get(session.ID)
		if err == nil {
			t.Error("Should not be able to get deleted session")
		}
	})

	t.Run("Load Persisted Sessions on Startup", func(t *testing.T) {
		// Create some sessions with first manager
		gameConfig := configManager.GetDefault()
		sessions := []string{"startup1", "startup2", "startup3"}
		for _, id := range sessions {
			_, err := manager.Create(id, gameConfig)
			if err != nil {
				t.Fatalf("Failed to create session %s: %v", id, err)
			}
		}

		// Create new manager (simulates server restart)
		manager4 := NewManagerWithPersistence(persistence)

		// Load persisted sessions
		err := manager4.LoadPersistedSessions()
		if err != nil {
			t.Fatalf("Failed to load persisted sessions: %v", err)
		}

		// Verify all sessions are accessible
		for _, id := range sessions {
			session, err := manager4.Get(id)
			if err != nil {
				t.Errorf("Failed to get session %s after loading persisted sessions: %v", id, err)
			}
			if session.ID != id {
				t.Errorf("Expected ID %s, got %s", id, session.ID)
			}
		}

		// Check that sessions list includes loaded sessions
		allSessions := manager4.List()
		if len(allSessions) < len(sessions) {
			t.Errorf("Expected at least %d sessions, got %d", len(sessions), len(allSessions))
		}
	})

	t.Run("Update Last Accessed Persists", func(t *testing.T) {
		// Get session
		session, err := manager.Get("startup1")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		originalTime := session.LastAccessedAt
		time.Sleep(10 * time.Millisecond) // Ensure time difference

		// Update last accessed
		err = manager.UpdateLastAccessed("startup1")
		if err != nil {
			t.Fatalf("Failed to update last accessed: %v", err)
		}

		// Create new manager and load session
		manager5 := NewManagerWithPersistence(persistence)
		loadedSession, err := manager5.Get("startup1")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}

		// Verify last accessed time was persisted and updated
		if !loadedSession.LastAccessedAt.After(originalTime) {
			t.Error("Last accessed time should be updated and persisted")
		}
	})
}
