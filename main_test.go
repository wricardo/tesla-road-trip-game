package main

import (
	"os"
	"testing"

	"github.com/wricardo/tesla-road-trip-game/appconfig"
)

func TestConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if AppName == "" {
		t.Error("AppName should not be empty")
	}

	expectedVersion := "2.0.0"
	if Version != expectedVersion {
		t.Errorf("Expected version %s, got %s", expectedVersion, Version)
	}

	expectedAppName := "Tesla Road Trip Game Server"
	if AppName != expectedAppName {
		t.Errorf("Expected app name %s, got %s", expectedAppName, AppName)
	}
}

func TestInitializeServices(t *testing.T) {
	tempSessions := t.TempDir()
	cfg := appconfig.Default()
	cfg.ConfigDir = "maps"
	cfg.SessionsDir = tempSessions

	// Create config directory if it doesn't exist for test
	if _, err := os.Stat("maps"); os.IsNotExist(err) {
		t.Skip("Skipping test - configs directory not found")
	}

	gameService, err := initializeServices(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}

	if gameService == nil {
		t.Fatal("Expected game service to be initialized")
	}
}

func TestInitializeServices_InvalidConfigDir(t *testing.T) {
	cfg := appconfig.Default()
	cfg.ConfigDir = "/non/existent/path"
	cfg.SessionsDir = t.TempDir()

	_, err := initializeServices(cfg)
	if err == nil {
		t.Error("Expected error for non-existent config directory")
	}
}

func TestFlagDefaults(t *testing.T) {
	// Test that flags have reasonable defaults
	if *port <= 0 || *port > 65535 {
		t.Errorf("Invalid default port: %d", *port)
	}

	if *host == "" {
		t.Error("Host should have a default value")
	}

	// Directory defaults live in appconfig; the flags are overrides and may be
	// empty so the environment/default supplies the value.
	cfg := appconfig.Default()
	if cfg.ConfigDir == "" {
		t.Error("Config directory should have a default value")
	}

	if cfg.SessionsDir == "" {
		t.Error("Sessions directory should have a default value")
	}
	if *configDir != "" || *sessionsDir != "" {
		t.Errorf("directory flags must default to empty so env wins, got %q/%q", *configDir, *sessionsDir)
	}
}

func TestGetSessionsDirDefault(t *testing.T) {
	t.Setenv("SESSIONS_DIR", "/tmp/tesla-sessions-test")
	cfg, err := appconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SessionsDir != "/tmp/tesla-sessions-test" {
		t.Errorf("Expected SESSIONS_DIR override, got %q", cfg.SessionsDir)
	}
}

// Note: We can't easily test main(), runHTTPServer(), and runStdioMCPWithInternalServer()
// without significant mocking or refactoring, as they start servers and block.
// These functions would be better tested in integration tests that start actual servers
// and test their endpoints.

func TestServiceInitialization(t *testing.T) {
	// Test that we can initialize services without panicking
	cfg := appconfig.Default()
	cfg.ConfigDir = "maps"
	cfg.SessionsDir = t.TempDir()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Service initialization panicked: %v", r)
		}
	}()

	// Create config directory if it doesn't exist for test
	if _, err := os.Stat("maps"); os.IsNotExist(err) {
		t.Skip("Skipping test - configs directory not found")
	}

	_, err := initializeServices(cfg)
	if err != nil {
		// This is expected if configs are missing, but shouldn't panic
		t.Logf("Service initialization failed as expected: %v", err)
	}
}
