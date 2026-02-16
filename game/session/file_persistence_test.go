package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/config"
	"github.com/wricardo/mcp-training/statefullgame/game/engine"
	"github.com/wricardo/mcp-training/statefullgame/game/service"
)

func TestFilePersistence(t *testing.T) {
	// Create temporary directory for test sessions
	tempDir, err := os.MkdirTemp("", "session_test_*")
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

	// Create test session
	gameConfig := configManager.GetDefault()
	engine, err := engine.NewEngine(gameConfig)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	session := &service.Session{
		ID:             "test1",
		Engine:         engine,
		Config:         gameConfig,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	t.Run("Save and Load Session", func(t *testing.T) {
		// Save session
		err := persistence.Save(session)
		if err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Check file exists
		if !persistence.Exists("test1") {
			t.Error("Session file should exist after save")
		}

		// Load session
		loadedSession, err := persistence.Load("test1")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}

		// Verify session data
		if loadedSession.ID != session.ID {
			t.Errorf("Expected ID %s, got %s", session.ID, loadedSession.ID)
		}
		if loadedSession.Config.Name != session.Config.Name {
			t.Errorf("Expected config name %s, got %s", session.Config.Name, loadedSession.Config.Name)
		}
		if loadedSession.Engine.GetState().Battery != session.Engine.GetState().Battery {
			t.Errorf("Expected battery %d, got %d", session.Engine.GetState().Battery, loadedSession.Engine.GetState().Battery)
		}
	})

	t.Run("Save State Changes", func(t *testing.T) {
		// Make a move to change state
		success := session.Engine.Move("right")
		if !success {
			t.Skip("Cannot test state persistence without successful move")
		}

		// Save updated session
		err := persistence.Save(session)
		if err != nil {
			t.Fatalf("Failed to save updated session: %v", err)
		}

		// Load and verify state was persisted
		loadedSession, err := persistence.Load("test1")
		if err != nil {
			t.Fatalf("Failed to load updated session: %v", err)
		}

		if loadedSession.Engine.GetState().PlayerPos != session.Engine.GetState().PlayerPos {
			t.Errorf("Player position not persisted correctly")
		}
		if len(loadedSession.Engine.GetMoveHistory()) != len(session.Engine.GetMoveHistory()) {
			t.Errorf("Move history not persisted correctly")
		}
	})

	t.Run("List All Sessions", func(t *testing.T) {
		// Create another session
		session2 := &service.Session{
			ID:             "test2",
			Engine:         engine,
			Config:         gameConfig,
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
		}
		err := persistence.Save(session2)
		if err != nil {
			t.Fatalf("Failed to save second session: %v", err)
		}

		// List all sessions
		sessionIDs, err := persistence.ListAll()
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(sessionIDs) < 2 {
			t.Errorf("Expected at least 2 sessions, got %d", len(sessionIDs))
		}

		// Check that our sessions are in the list
		found := make(map[string]bool)
		for _, id := range sessionIDs {
			found[id] = true
		}
		if !found["test1"] || !found["test2"] {
			t.Error("Expected sessions not found in list")
		}
	})

	t.Run("Delete Session", func(t *testing.T) {
		// Delete session
		err := persistence.Delete("test2")
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Verify it no longer exists
		if persistence.Exists("test2") {
			t.Error("Session should not exist after delete")
		}

		// Verify we can't load it
		_, err = persistence.Load("test2")
		if err == nil {
			t.Error("Should not be able to load deleted session")
		}
	})

	t.Run("Error Cases", func(t *testing.T) {
		// Try to load non-existent session
		_, err := persistence.Load("nonexistent")
		if err == nil {
			t.Error("Should get error when loading non-existent session")
		}

		// Try to delete non-existent session
		err = persistence.Delete("nonexistent")
		if err == nil {
			t.Error("Should get error when deleting non-existent session")
		}

		// Try to save nil session
		err = persistence.Save(nil)
		if err == nil {
			t.Error("Should get error when saving nil session")
		}
	})
}

func TestFilePersistenceFileStructure(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "session_file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configManager, err := config.NewManager("../../configs")
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	persistence, err := NewFilePersistence(tempDir, configManager)
	if err != nil {
		t.Fatalf("Failed to create file persistence: %v", err)
	}

	// Create and save session
	gameConfig := configManager.GetDefault()
	engine, err := engine.NewEngine(gameConfig)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	session := &service.Session{
		ID:             "file_test",
		Engine:         engine,
		Config:         gameConfig,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	err = persistence.Save(session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Check file exists in correct location
	expectedFile := filepath.Join(tempDir, "file_test.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedFile)
	}

	// Check file contains valid JSON
	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Session file should not be empty")
	}

	// Check it contains expected fields (basic validation)
	content := string(data)
	expectedFields := []string{"\"id\"", "\"config_name\"", "\"created_at\"", "\"game_state\""}
	for _, field := range expectedFields {
		if !containsString(content, field) {
			t.Errorf("Session file should contain field %s", field)
		}
	}
}

func containsString(str, substr string) bool {
	return strings.Contains(str, substr)
}
