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
			Welcome:            "Welcome to the test game!",
			HomeCharge:         "Home charging!",
			SuperchargerCharge: "Supercharger!",
			ParkVisited:        "Park visited! Score: %d",
			ParkAlreadyVisited: "Already visited",
			Victory:            "Victory! All %d parks visited!",
			OutOfBattery:       "Out of battery!",
			Stranded:           "Stranded!",
			CantMove:           "Can't move!",
			BatteryStatus:      "Battery: %d/%d",
			HitWall:            "Hit wall!",
		},
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

func TestValidateGameConfig_MissingDescription(t *testing.T) {
	config := createValidConfig()
	config.Description = ""
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for missing description")
	}
	if !strings.Contains(err.Error(), "description is required") {
		t.Errorf("Expected description validation error, got: %v", err)
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

func TestValidateGameConfig_MissingMessages(t *testing.T) {
	tests := []struct {
		name         string
		messageField string
		modifier     func(*GameConfig)
	}{
		{"welcome", "messages.welcome", func(c *GameConfig) { c.Messages.Welcome = "" }},
		{"victory", "messages.victory", func(c *GameConfig) { c.Messages.Victory = "" }},
		{"out of battery", "messages.out_of_battery", func(c *GameConfig) { c.Messages.OutOfBattery = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := createValidConfig()
			test.modifier(config)
			err := ValidateGameConfig(config)
			if err == nil {
				t.Errorf("Expected error for missing %s", test.messageField)
			}
			if !strings.Contains(err.Error(), test.messageField+" is required") {
				t.Errorf("Expected %s validation error, got: %v", test.messageField, err)
			}
		})
	}
}

func TestValidateGameConfig_HitWallMessage(t *testing.T) {
	config := createValidConfig()
	config.WallCrashEndsGame = true
	config.Messages.HitWall = ""
	err := ValidateGameConfig(config)
	if err == nil {
		t.Error("Expected error for missing hit wall message when wall crash ends game")
	}
	if !strings.Contains(err.Error(), "messages.hit_wall is required when wall_crash_ends_game is true") {
		t.Errorf("Expected hit wall message validation error, got: %v", err)
	}
}

func TestValidateGameConfig_FormatStrings(t *testing.T) {
	tests := []struct {
		name     string
		modifier func(*GameConfig)
		expected string
	}{
		{"park visited", func(c *GameConfig) { c.Messages.ParkVisited = "No format" }, "park_visited must contain %d"},
		{"victory", func(c *GameConfig) { c.Messages.Victory = "No format" }, "victory must contain %d"},
		{"battery status", func(c *GameConfig) { c.Messages.BatteryStatus = "No format" }, "battery_status must contain %d"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := createValidConfig()
			test.modifier(config)
			err := ValidateGameConfig(config)
			if err == nil {
				t.Errorf("Expected error for %s format string", test.name)
			}
			if !strings.Contains(err.Error(), test.expected) {
				t.Errorf("Expected format string validation error containing '%s', got: %v", test.expected, err)
			}
		})
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
	os.MkdirAll("configs", 0755)

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

	err := os.WriteFile(filepath.Join("configs", "test.json"), []byte(configContent), 0644)
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
