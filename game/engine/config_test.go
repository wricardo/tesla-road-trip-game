package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createValidConfig() *GameConfig {
	return &GameConfig{
		Name:            "Test Config",
		Description:     "A valid test configuration",
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
	}
}

func TestValidateGameConfig_ValidConfig(t *testing.T) {
	config := createValidConfig()
	err := ValidateGameConfig(config)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}
}

func TestValidateGameConfig_MissingName(t *testing.T) {
	config := createValidConfig()
	config.Name = ""
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected name validation error, got: %v", err)
	}
}

func TestValidateGameConfig_BlankDescriptionAllowed(t *testing.T) {
	config := createValidConfig()
	config.Description = ""
	err := ValidateGameConfig(config)
	if err != nil {
		t.Errorf("Expected blank description to pass validation, got: %v", err)
	}
}

func TestValidateGameConfig_InvalidGridSize(t *testing.T) {
	tests := []struct {
		name     string
		gridSize int
	}{
		{"too small", 4},
		{"too large", 51},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := createValidConfig()
			config.GridSize = test.gridSize
			err := ValidateGameConfig(config)
			if err == nil {
				t.Errorf("Expected error for grid size %d", test.gridSize)
			}
			if !strings.Contains(err.Error(), "grid_size must be between") {
				t.Errorf("Expected grid size validation error, got: %v", err)
			}
		})
	}
}

func TestValidateGameConfig_InvalidBattery(t *testing.T) {
	tests := []struct {
		name            string
		maxBattery      int
		startingBattery int
		expectedError   string
	}{
		{"max battery too small", 0, 5, "max_battery must be between"},
		{"max battery too large", 101, 5, "max_battery must be between"},
		{"starting battery too small", 10, 0, "starting_battery must be between"},
		{"starting battery larger than max", 10, 15, "starting_battery must be between"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := createValidConfig()
			config.MaxBattery = test.maxBattery
			config.StartingBattery = test.startingBattery
			err := ValidateGameConfig(config)
			if err == nil {
				t.Errorf("Expected error for battery config: max=%d, start=%d", test.maxBattery, test.startingBattery)
			}
			if !strings.Contains(err.Error(), test.expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", test.expectedError, err)
			}
		})
	}
}

func TestValidateGameConfig_LayoutSizeMismatch(t *testing.T) {
	config := createValidConfig()
	config.GridSize = 7
	// Layout still has 5 rows, but grid size is 7 - need 7x7 layout
	config.Layout = []string{
		"BBBBBBB",
		"BHRRRRB",
		"BRPRRRB",
		"BRRRRRB",
		"BRRRRRB",
		"BRRRRRB",
		"BBBBBBB",
	}
	// Now create a mismatch by changing grid size back to 5
	config.GridSize = 5
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for layout size mismatch")
	}
	if !strings.Contains(err.Error(), "layout must have 5 rows") {
		t.Errorf("Expected layout row validation error, got: %v", err)
	}
}

func TestValidateGameConfig_LayoutRowSizeMismatch(t *testing.T) {
	config := createValidConfig()
	config.Layout[0] = "BBB" // Row too short
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for layout row size mismatch")
	}
	if !strings.Contains(err.Error(), "must have 5 characters") {
		t.Errorf("Expected layout column validation error, got: %v", err)
	}
}

func TestValidateGameConfig_InvalidCharacters(t *testing.T) {
	config := createValidConfig()
	config.Layout[1] = "BRXPB" // X is invalid
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for invalid character")
	}
	if !strings.Contains(err.Error(), "invalid character 'X'") {
		t.Errorf("Expected invalid character error, got: %v", err)
	}
}

func TestValidateGameConfig_NoHome(t *testing.T) {
	config := createValidConfig()
	config.Layout = []string{
		"BBBBB",
		"BRRRB",
		"BRPPB",
		"BRRRB",
		"BBBBB",
	}
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for no home cell")
	}
	if !strings.Contains(err.Error(), "must contain at least one home") {
		t.Errorf("Expected no home validation error, got: %v", err)
	}
}

func TestValidateGameConfig_NoParks(t *testing.T) {
	config := createValidConfig()
	config.Layout = []string{
		"BBBBB",
		"BRHSB",
		"BRRRB",
		"BRRRB",
		"BBBBB",
	}
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for no park cells")
	}
	if !strings.Contains(err.Error(), "must contain at least one park") {
		t.Errorf("Expected no park validation error, got: %v", err)
	}
}

func TestValidateGameConfig_InvalidLegend(t *testing.T) {
	config := createValidConfig()
	config.Legend["R"] = "wrong" // Should be "road"
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for invalid legend")
	}
	if !strings.Contains(err.Error(), "legend['R'] must be 'road'") {
		t.Errorf("Expected legend validation error, got: %v", err)
	}
}

func TestValidateGameConfig_Winnability(t *testing.T) {
	config := createValidConfig()
	// Create a layout where parks are unreachable with low battery
	config.GridSize = 9
	config.MaxBattery = 5
	config.StartingBattery = 5
	config.Layout = []string{
		"BBBBBBBBB",
		"BHRRRRRRB",
		"BWWWWWWWB",
		"BWWWWWWWB",
		"BWWWWWWWB",
		"BWWWWWWWB",
		"BWWWWWWWB",
		"BWWWWWWPB", // Park at (7,7) - distance from home (1,1) is 12 moves
		"BBBBBBBBB",
	}
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for unreachable park")
	}
	if err != nil && !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("Expected unreachable park validation error, got: %v", err)
	}
}

func TestLoadConfigByName(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()

	// Change to temp directory temporarily
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create configs directory
	os.MkdirAll("maps", 0755)

	configContent := `{
		"name": "Test Config",
		"description": "Test description",
		"grid_size": 5,
		"max_battery": 10,
		"starting_battery": 8,
		"layout": [
			"BBBBB",
			"BRHPB",
			"BRRRB",
			"BPPPB",
			"BBBBB"
		],
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park",
			"S": "supercharger",
			"W": "water",
			"B": "building"
		},
		"wall_crash_ends_game": false,
		"messages": {
			"welcome": "Welcome!",
			"home_charge": "Home!",
			"supercharger_charge": "Supercharger!",
			"park_visited": "Park! Score: %d",
			"park_already_visited": "Already visited",
			"victory": "Victory! %d parks!",
			"out_of_battery": "No battery!",
			"stranded": "Stranded!",
			"cant_move": "Can't move!",
			"battery_status": "Battery: %d/%d",
			"hit_wall": "Hit wall!"
		}
	}`

	err := os.WriteFile(filepath.Join("maps", "test.json"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading by name without extension
	config, err := LoadConfigByName("test")
	if err != nil {
		t.Fatalf("Failed to load config by name: %v", err)
	}
	if config.Name != "Test Config" {
		t.Errorf("Expected config name 'Test Config', got '%s'", config.Name)
	}

	// Test loading by name with extension
	config2, err := LoadConfigByName("test.json")
	if err != nil {
		t.Fatalf("Failed to load config by name with extension: %v", err)
	}
	if config2.Name != "Test Config" {
		t.Errorf("Expected config name 'Test Config', got '%s'", config2.Name)
	}

	// Test loading non-existent config
	_, err = LoadConfigByName("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestLoadGameConfig(t *testing.T) {
	// Create a temporary config file
	tempFile := filepath.Join(t.TempDir(), "test_config.json")

	configContent := `{
		"name": "Test Config",
		"description": "Test description",
		"grid_size": 5,
		"max_battery": 10,
		"starting_battery": 8,
		"layout": [
			"BBBBB",
			"BRHPB",
			"BRRRB",
			"BPPPB",
			"BBBBB"
		],
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park",
			"S": "supercharger",
			"W": "water",
			"B": "building"
		},
		"wall_crash_ends_game": false,
		"messages": {
			"welcome": "Welcome!",
			"home_charge": "Home!",
			"supercharger_charge": "Supercharger!",
			"park_visited": "Park! Score: %d",
			"park_already_visited": "Already visited",
			"victory": "Victory! %d parks!",
			"out_of_battery": "No battery!",
			"stranded": "Stranded!",
			"cant_move": "Can't move!",
			"battery_status": "Battery: %d/%d",
			"hit_wall": "Hit wall!"
		}
	}`

	err := os.WriteFile(tempFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadGameConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Name != "Test Config" {
		t.Errorf("Expected config name 'Test Config', got '%s'", config.Name)
	}
	if config.GridSize != 5 {
		t.Errorf("Expected grid size 5, got %d", config.GridSize)
	}

	// Test loading non-existent file
	_, err = LoadGameConfig("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestInitGameStateFromConfig(t *testing.T) {
	config := createValidConfig()
	state := InitGameStateFromConfig(config)

	// Test basic state initialization
	if state.Battery != config.StartingBattery {
		t.Errorf("Expected battery %d, got %d", config.StartingBattery, state.Battery)
	}
	if state.MaxBattery != config.MaxBattery {
		t.Errorf("Expected max battery %d, got %d", config.MaxBattery, state.MaxBattery)
	}
	if state.Score != 0 {
		t.Errorf("Expected score 0, got %d", state.Score)
	}
	if state.GameOver {
		t.Error("Expected game not to be over initially")
	}
	if state.Victory {
		t.Error("Expected game not to be victory initially")
	}

	// Test grid initialization
	if len(state.Grid) != config.GridSize {
		t.Errorf("Expected grid size %d, got %d", config.GridSize, len(state.Grid))
	}

	// Test player starts at home
	homeCell := state.Grid[state.PlayerPos.Y][state.PlayerPos.X]
	if homeCell.Type != Home {
		t.Errorf("Expected player to start at home, got %v", homeCell.Type)
	}

	// Test visited parks map is initialized
	if state.VisitedParks == nil {
		t.Error("Expected VisitedParks map to be initialized")
	}
	if len(state.VisitedParks) != 0 {
		t.Errorf("Expected empty VisitedParks initially, got %d entries", len(state.VisitedParks))
	}

	// Test nil config uses defaults
	defaultState := InitGameStateFromConfig(nil)
	if defaultState.MaxBattery != 10 {
		t.Errorf("Expected default max battery 10, got %d", defaultState.MaxBattery)
	}
}
