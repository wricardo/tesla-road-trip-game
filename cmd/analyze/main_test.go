package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalysisConfig(t *testing.T) {
	config := AnalysisConfig{
		Name:            "Test Config",
		Description:     "Test configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 8,
		Layout: []string{
			"BBBBB",
			"BRHPB",
			"BRRRB",
			"BPPPB",
			"BBBBB",
		},
		Legend: map[string]string{
			"R": "road",
			"H": "home",
			"P": "park",
			"S": "supercharger",
			"W": "water",
			"B": "building",
		},
		WallCrashEndsGame: false,
		Messages: map[string]string{
			"welcome": "Welcome!",
		},
	}

	if config.Name != "Test Config" {
		t.Errorf("Expected Name 'Test Config', got '%s'", config.Name)
	}

	if config.GridSize != 5 {
		t.Errorf("Expected GridSize 5, got %d", config.GridSize)
	}

	if len(config.Layout) != 5 {
		t.Errorf("Expected 5 layout rows, got %d", len(config.Layout))
	}
}

func TestAnalysisPoint(t *testing.T) {
	point := AnalysisPoint{X: 3, Y: 5}

	if point.X != 3 {
		t.Errorf("Expected X 3, got %d", point.X)
	}

	if point.Y != 5 {
		t.Errorf("Expected Y 5, got %d", point.Y)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-10, 10},
		{100, 100},
	}

	for _, test := range tests {
		result := abs(test.input)
		if result != test.expected {
			t.Errorf("abs(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestAnalyzeConfig_ValidFile(t *testing.T) {
	// Create a temporary test config file
	validConfig := `{
		"name": "Test Config",
		"description": "Test configuration",
		"grid_size": 3,
		"max_battery": 10,
		"starting_battery": 8,
		"layout": [
			"RRR",
			"RHR",
			"RPR"
		],
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park"
		},
		"wall_crash_ends_game": false,
		"messages": {
			"welcome": "Welcome!"
		}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(validConfig)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	// Test that analyzeConfig doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("analyzeConfig panicked: %v", r)
		}
	}()

	analyzeConfig(tmpfile.Name())
}

func TestAnalyzeConfig_InvalidFile(t *testing.T) {
	// Test with non-existent file
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("analyzeConfig panicked with invalid file: %v", r)
		}
	}()

	analyzeConfig("/non/existent/file.json")
}

func TestAnalyzeConfig_InvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	invalidJSON := `{"name": "test", invalid json}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(invalidJSON)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	// Test that analyzeConfig doesn't panic with invalid JSON
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("analyzeConfig panicked with invalid JSON: %v", r)
		}
	}()

	analyzeConfig(tmpfile.Name())
}

func TestMain_Integration(t *testing.T) {
	// Create a temporary configs directory for testing
	tmpDir, err := os.MkdirTemp("", "test_configs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file
	testConfig := `{
		"name": "Test Config",
		"description": "Test configuration",
		"grid_size": 3,
		"max_battery": 10,
		"starting_battery": 8,
		"layout": [
			"RRR",
			"RHR",
			"RPR"
		],
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park"
		},
		"wall_crash_ends_game": false,
		"messages": {
			"welcome": "Welcome!"
		}
	}`

	configPath := filepath.Join(tmpDir, "classic.json")
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Save original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create configs subdirectory and move the file there
	if err := os.Mkdir("configs", 0755); err != nil {
		t.Fatalf("Failed to create configs dir: %v", err)
	}

	if err := os.Rename("classic.json", "configs/classic.json"); err != nil {
		t.Fatalf("Failed to move config file: %v", err)
	}

	// Test that main doesn't panic (we can't easily test output without complex mocking)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// We can't call main() directly as it would process all hardcoded configs,
	// but we can test analyzeConfig with our test file
	analyzeConfig("configs/classic.json")
}

func TestAnalyzeConfig_ReachabilityAnalysis(t *testing.T) {
	// Test config with unreachable areas
	configWithUnreachableArea := `{
		"name": "Unreachable Test",
		"description": "Config with unreachable areas",
		"grid_size": 5,
		"max_battery": 2,
		"starting_battery": 2,
		"layout": [
			"HRRRR",
			"WWWWR",
			"RWWWR",
			"RWWWR",
			"RPRRR"
		],
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park",
			"W": "water"
		},
		"wall_crash_ends_game": false,
		"messages": {
			"welcome": "Welcome!"
		}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configWithUnreachableArea)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	// Test that analyzeConfig handles unreachable areas without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("analyzeConfig panicked with unreachable areas: %v", r)
		}
	}()

	analyzeConfig(tmpfile.Name())
}
