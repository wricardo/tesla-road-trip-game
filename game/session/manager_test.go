package session

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/engine"
)

func createTestConfig() *engine.GameConfig {
	return &engine.GameConfig{
		Name:            "Test Config",
		Description:     "Test configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 8,
		Layout: []string{
			"BBBBB",
			"BRHPB",
			"BRRSB",
			"BPPPB",
			"BBBBB",
		},
		Legend: map[string]string{
			"R": "road", "H": "home", "P": "park",
			"S": "supercharger", "W": "water", "B": "building",
		},
		WallCrashEndsGame: false,
		Messages: struct {
			Welcome            string `json:"welcome"`
			HomeCharge         string `json:"home_charge"`
			SuperchargerCharge string `json:"supercharger_charge"`
			ParkVisited        string `json:"park_visited"`
			ParkAlreadyVisited string `json:"park_already_visited"`
			Victory            string `json:"victory"`
			OutOfBattery       string `json:"out_of_battery"`
			Stranded           string `json:"stranded"`
			CantMove           string `json:"cant_move"`
			BatteryStatus      string `json:"battery_status"`
			HitWall            string `json:"hit_wall"`
		}{
			Welcome:            "Welcome!",
			HomeCharge:         "Home charged!",
			SuperchargerCharge: "Supercharged!",
			ParkVisited:        "Park visited! Score: %d",
			ParkAlreadyVisited: "Already visited",
			Victory:            "Victory! All %d parks!",
			OutOfBattery:       "No battery!",
			Stranded:           "Stranded!",
			CantMove:           "Can't move!",
			BatteryStatus:      "Battery: %d/%d",
			HitWall:            "Hit wall!",
		},
	}
}

func TestManager_Create(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	t.Run("create with custom ID", func(t *testing.T) {
		session, err := manager.Create("test-session", config)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		if session.ID != "test-session" {
			t.Errorf("Expected session ID 'test-session', got '%s'", session.ID)
		}
		if session.Engine == nil {
			t.Error("Expected engine to be initialized")
		}
	})

	t.Run("create with auto-generated ID", func(t *testing.T) {
		session, err := manager.Create("", config)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		if session.ID == "" {
			t.Error("Expected auto-generated session ID")
		}
		if len(session.ID) != 4 {
			t.Errorf("Expected 4-character session ID, got %d characters", len(session.ID))
		}
	})

	t.Run("duplicate session ID", func(t *testing.T) {
		_, err := manager.Create("test-session", config)
		if err != ErrSessionAlreadyExists {
			t.Errorf("Expected ErrSessionAlreadyExists, got %v", err)
		}
	})

	t.Run("case-insensitive duplicate check", func(t *testing.T) {
		_, err := manager.Create("TEST-SESSION", config)
		if err != ErrSessionAlreadyExists {
			t.Errorf("Expected ErrSessionAlreadyExists for case variant, got %v", err)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		invalidConfig := createTestConfig()
		invalidConfig.Name = "" // Make config invalid
		_, err := manager.Create("invalid-test", invalidConfig)
		if err == nil {
			t.Error("Expected error for invalid config")
		}
	})
}

func TestManager_Get(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Create test session
	created, _ := manager.Create("get-test", config)

	t.Run("get existing session", func(t *testing.T) {
		session, err := manager.Get("get-test")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}
		if session.ID != created.ID {
			t.Errorf("Expected session ID '%s', got '%s'", created.ID, session.ID)
		}
	})

	t.Run("case-insensitive get", func(t *testing.T) {
		session, err := manager.Get("GET-TEST")
		if err != nil {
			t.Fatalf("Failed to get session with different case: %v", err)
		}
		if session.ID != created.ID {
			t.Errorf("Expected same session regardless of case")
		}
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := manager.Get("non-existent")
		if err != ErrSessionNotFound {
			t.Errorf("Expected ErrSessionNotFound, got %v", err)
		}
	})
}

func TestManager_GetOrCreate(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	t.Run("create new session", func(t *testing.T) {
		session, err := manager.GetOrCreate("new-session", config)
		if err != nil {
			t.Fatalf("Failed to get or create session: %v", err)
		}
		if session.ID != "new-session" {
			t.Errorf("Expected session ID 'new-session', got '%s'", session.ID)
		}
	})

	t.Run("get existing session", func(t *testing.T) {
		// Should get the same session without creating new one
		session, err := manager.GetOrCreate("new-session", config)
		if err != nil {
			t.Fatalf("Failed to get existing session: %v", err)
		}
		if session.ID != "new-session" {
			t.Errorf("Expected same session ID")
		}
	})
}

func TestManager_Delete(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Create test session
	manager.Create("delete-test", config)

	t.Run("delete existing session", func(t *testing.T) {
		err := manager.Delete("delete-test")
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Verify session is deleted
		_, err = manager.Get("delete-test")
		if err != ErrSessionNotFound {
			t.Error("Expected session to be deleted")
		}
	})

	t.Run("delete non-existent session", func(t *testing.T) {
		err := manager.Delete("non-existent")
		if err != ErrSessionNotFound {
			t.Errorf("Expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("case-insensitive delete", func(t *testing.T) {
		manager.Create("case-test", config)
		err := manager.Delete("CASE-TEST")
		if err != nil {
			t.Fatalf("Failed to delete with different case: %v", err)
		}
		_, err = manager.Get("case-test")
		if err != ErrSessionNotFound {
			t.Error("Expected session to be deleted regardless of case")
		}
	})
}

func TestManager_List(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Create multiple sessions
	session1, _ := manager.Create("list-1", config)
	session2, _ := manager.Create("list-2", config)
	session3, _ := manager.Create("list-3", config)

	sessions := manager.List()

	if len(sessions) < 3 {
		t.Errorf("Expected at least 3 sessions, got %d", len(sessions))
	}

	// Verify all created sessions are in the list
	foundSessions := make(map[string]bool)
	for _, s := range sessions {
		foundSessions[s.ID] = true
	}

	if !foundSessions[session1.ID] {
		t.Error("Session 1 not found in list")
	}
	if !foundSessions[session2.ID] {
		t.Error("Session 2 not found in list")
	}
	if !foundSessions[session3.ID] {
		t.Error("Session 3 not found in list")
	}
}

func TestManager_CleanupExpired(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Create sessions with different last access times
	active, _ := manager.Create("active", config)
	expired, _ := manager.Create("expired", config)

	// Simulate expired session
	expired.LastAccessedAt = time.Now().Add(-2 * time.Hour)
	active.LastAccessedAt = time.Now()

	// Clean up sessions older than 1 hour
	deleted := manager.CleanupExpiredSessions(1 * time.Hour)

	if deleted != 1 {
		t.Errorf("Expected 1 session to be deleted, got %d", deleted)
	}

	// Verify expired session is deleted
	_, err := manager.Get("expired")
	if err != ErrSessionNotFound {
		t.Error("Expected expired session to be deleted")
	}

	// Verify active session still exists
	_, err = manager.Get("active")
	if err != nil {
		t.Error("Expected active session to still exist")
	}
}

func TestManager_UpdateLastAccessed(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	session, _ := manager.Create("access-test", config)
	originalTime := session.LastAccessedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	err := manager.UpdateLastAccessed("access-test")
	if err != nil {
		t.Fatalf("Failed to update last accessed: %v", err)
	}

	// Get session again to verify update
	updated, _ := manager.Get("access-test")
	if !updated.LastAccessedAt.After(originalTime) {
		t.Error("Expected LastAccessedAt to be updated")
	}
}

func TestManager_Exists(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	manager.Create("exists-test", config)

	t.Run("existing session", func(t *testing.T) {
		if !manager.sessionExists("exists-test") {
			t.Error("Expected session to exist")
		}
	})

	t.Run("case-insensitive existence check", func(t *testing.T) {
		if !manager.sessionExists("EXISTS-TEST") {
			t.Error("Expected session to exist regardless of case")
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		if manager.sessionExists("non-existent") {
			t.Error("Expected session not to exist")
		}
	})
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Test concurrent session creation
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sessionID := strings.ToLower(generateRandomID())
			_, err := manager.Create(sessionID, config)
			if err != nil && err != ErrSessionAlreadyExists {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for unexpected errors
	for err := range errors {
		t.Errorf("Unexpected error during concurrent access: %v", err)
	}

	// Verify sessions were created
	sessions := manager.List()
	if len(sessions) == 0 {
		t.Error("Expected sessions to be created")
	}
}

func TestManager_SessionIsolation(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	// Create two sessions
	session1, _ := manager.Create("iso-1", config)
	session2, _ := manager.Create("iso-2", config)

	// Modify session 1
	session1.Engine.Move("right")

	// Verify session 2 is not affected
	if session2.Engine.GetPlayerPosition().X != 2 {
		t.Error("Session 2 should not be affected by session 1 moves")
	}

	if session1.Engine.GetPlayerPosition().X == session2.Engine.GetPlayerPosition().X {
		t.Error("Sessions should have independent game state")
	}
}

func TestManager_SessionIDGeneration(t *testing.T) {
	manager := NewManager()
	config := createTestConfig()

	generatedIDs := make(map[string]bool)

	// Generate multiple sessions and check for uniqueness
	for i := 0; i < 50; i++ {
		session, err := manager.Create("", config)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if generatedIDs[session.ID] {
			t.Errorf("Duplicate session ID generated: %s", session.ID)
		}
		generatedIDs[session.ID] = true

		// Verify ID format (4 alphanumeric characters)
		if len(session.ID) != 4 {
			t.Errorf("Expected 4-character ID, got %d", len(session.ID))
		}
	}
}

// Helper function to generate random ID for testing
func generateRandomID() string {
	return "test-" + time.Now().Format("150405")
}
