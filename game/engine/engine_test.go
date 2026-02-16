package engine

import (
	"testing"
)

func createTestConfig() *GameConfig {
	return &GameConfig{
		Name:            "Engine Test Config",
		Description:     "Configuration for engine integration tests",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 8,
		Layout: []string{
			"BBBBB",
			"BRHPB",
			"BRWSB",
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
			Welcome:            "Welcome to engine test!",
			HomeCharge:         "Home charged!",
			SuperchargerCharge: "Supercharged!",
			ParkVisited:        "Park visited! Score: %d",
			ParkAlreadyVisited: "Already visited this park",
			Victory:            "Victory! All %d parks visited!",
			OutOfBattery:       "Out of battery!",
			Stranded:           "Stranded!",
			CantMove:           "Can't move there!",
			BatteryStatus:      "Battery: %d/%d",
			HitWall:            "Hit wall!",
		},
	}
}

func TestNewEngine(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create new engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	// Test initial state
	if engine.GetBattery() != config.StartingBattery {
		t.Errorf("Expected starting battery %d, got %d", config.StartingBattery, engine.GetBattery())
	}
	if engine.GetScore() != 0 {
		t.Errorf("Expected initial score 0, got %d", engine.GetScore())
	}
	if engine.IsGameOver() {
		t.Error("Expected game not to be over initially")
	}
	if engine.IsVictory() {
		t.Error("Expected game not to be victory initially")
	}
}

func TestNewEngine_InvalidConfig(t *testing.T) {
	config := createTestConfig()
	config.Name = "" // Make config invalid

	_, err := NewEngine(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestNewEngineWithDefaults(t *testing.T) {
	engine := NewEngineWithDefaults()
	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	// Should have reasonable defaults
	if engine.GetBattery() <= 0 {
		t.Error("Expected positive starting battery")
	}
	if engine.GetScore() != 0 {
		t.Errorf("Expected initial score 0, got %d", engine.GetScore())
	}
}

func TestEngine_BasicMovement(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	initialPos := engine.GetPlayerPosition()
	initialBattery := engine.GetBattery()

	// Test successful move
	success := engine.Move("right")
	if !success {
		t.Error("Expected successful move")
	}

	newPos := engine.GetPlayerPosition()
	if newPos.X != initialPos.X+1 {
		t.Errorf("Expected X position to increase by 1, was %d now %d", initialPos.X, newPos.X)
	}
	if engine.GetBattery() != initialBattery-1 {
		t.Errorf("Expected battery to decrease by 1, was %d now %d", initialBattery, engine.GetBattery())
	}

	// Test move history
	history := engine.GetMoveHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 move in history, got %d", len(history))
	}

	lastMove := engine.GetLastMove()
	if lastMove == nil {
		t.Error("Expected last move to be non-nil")
	}
	if lastMove.Action != "right" {
		t.Errorf("Expected last move action 'right', got '%s'", lastMove.Action)
	}
}

func TestEngine_CanMove(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test valid move
	if !engine.CanMove("right") {
		t.Error("Expected to be able to move right")
	}

	// Test invalid move (into water)
	if engine.CanMove("down") {
		t.Error("Expected not to be able to move down into water")
	}

	// Test invalid direction
	if engine.CanMove("invalid") {
		t.Error("Expected not to be able to move in invalid direction")
	}
}

func TestEngine_GetPossibleMoves(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	possibleMoves := engine.GetPossibleMoves()

	// Should be able to move left and right from starting position (Home at 2,1)
	expectedMoves := []string{"left", "right"}
	if len(possibleMoves) != len(expectedMoves) {
		t.Errorf("Expected %d possible moves, got %d: %v", len(expectedMoves), len(possibleMoves), possibleMoves)
	}

	for _, expected := range expectedMoves {
		found := false
		for _, actual := range possibleMoves {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find '%s' in possible moves: %v", expected, possibleMoves)
		}
	}
}

func TestEngine_ConfigManagement(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test getting config
	retrievedConfig := engine.GetConfig()
	if retrievedConfig.Name != config.Name {
		t.Errorf("Expected config name '%s', got '%s'", config.Name, retrievedConfig.Name)
	}

	// Test setting new config
	newConfig := createTestConfig()
	newConfig.Name = "New Config"
	newConfig.MaxBattery = 15

	err = engine.SetConfig(newConfig)
	if err != nil {
		t.Errorf("Failed to set new config: %v", err)
	}

	if engine.GetConfig().Name != newConfig.Name {
		t.Errorf("Expected new config name '%s', got '%s'", newConfig.Name, engine.GetConfig().Name)
	}
	if engine.GetBattery() != newConfig.StartingBattery {
		t.Errorf("Expected battery reset to %d, got %d", newConfig.StartingBattery, engine.GetBattery())
	}

	// Test setting invalid config
	invalidConfig := createTestConfig()
	invalidConfig.Name = ""
	err = engine.SetConfig(invalidConfig)
	if err == nil {
		t.Error("Expected error when setting invalid config")
	}
}

func TestEngine_Reset(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Make some moves to change state
	engine.Move("right")
	engine.Move("right") // Move to park to change score

	// Verify state changed
	if engine.GetScore() == 0 {
		t.Error("Expected score to have changed before reset")
	}
	if len(engine.GetMoveHistory()) == 0 {
		t.Error("Expected move history before reset")
	}

	// Reset and verify state restored
	newState := engine.Reset()
	if newState == nil {
		t.Error("Expected reset to return game state")
	}
	if engine.GetScore() != 0 {
		t.Errorf("Expected score to be reset to 0, got %d", engine.GetScore())
	}
	if engine.GetBattery() != config.StartingBattery {
		t.Errorf("Expected battery reset to %d, got %d", config.StartingBattery, engine.GetBattery())
	}
	// Move history is now cumulative across resets, but current segment is cleared
	if len(engine.GetMoveHistory()) < 2 {
		t.Errorf("Expected cumulative move history retained after reset, got %d moves", len(engine.GetMoveHistory()))
	}
	if len(newState.CurrentMoves) != 0 || newState.CurrentMovesCount != 0 {
		t.Errorf("Expected current moves cleared after reset, got len=%d count=%d", len(newState.CurrentMoves), newState.CurrentMovesCount)
	}
	if engine.IsGameOver() {
		t.Error("Expected game not to be over after reset")
	}
}

func TestEngine_ParkManagement(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test initial park state
	totalParks := engine.GetTotalParks()
	if totalParks <= 0 {
		t.Error("Expected positive number of total parks")
	}

	visitedParks := engine.GetVisitedParks()
	if len(visitedParks) != 0 {
		t.Errorf("Expected no visited parks initially, got %d", len(visitedParks))
	}

	remaining := engine.GetRemainingParks()
	if remaining != totalParks {
		t.Errorf("Expected remaining parks (%d) to equal total parks (%d)", remaining, totalParks)
	}

	// Visit a park
	engine.Move("right") // Move to park

	visitedParks = engine.GetVisitedParks()
	if len(visitedParks) != 1 {
		t.Errorf("Expected 1 visited park, got %d", len(visitedParks))
	}

	remaining = engine.GetRemainingParks()
	if remaining != totalParks-1 {
		t.Errorf("Expected remaining parks to be %d, got %d", totalParks-1, remaining)
	}
}

func TestEngine_LocalView(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	localView := engine.GetLocalView()
	if len(localView) != 8 {
		t.Errorf("Expected 8 cells in local view, got %d", len(localView))
	}

	// Verify local view contains valid cell types
	for i, cell := range localView {
		if cell.Type == "" {
			t.Errorf("Local view cell %d has empty type", i)
		}
	}
}

func TestEngine_GameOverScenarios(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test battery depletion
	// Reduce battery to minimum
	state := engine.GetState()
	state.Battery = 1

	// Move to use last battery and get stranded
	state.PlayerPos = Position{X: 1, Y: 3} // Position away from chargers
	engine.Move("right")

	if !engine.IsGameOver() {
		t.Error("Expected game to be over when stranded")
	}

	// Reset for next test
	engine.Reset()

	// Test wall crash ending game
	config.WallCrashEndsGame = true
	engine.SetConfig(config)

	// Try to move into wall
	engine.Move("down") // Should hit water

	if !engine.IsGameOver() {
		t.Error("Expected game to be over after wall crash")
	}
}

func TestEngine_VictoryScenario(t *testing.T) {
	// Create a minimal config with just one park for easy victory
	config := &GameConfig{
		Name:            "Victory Test",
		Description:     "Test victory condition",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 8,
		Layout: []string{
			"BBBBB",
			"BRHPB",
			"BRRRB",
			"BRRRB",
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
			Welcome:            "Welcome!",
			HomeCharge:         "Home!",
			SuperchargerCharge: "Super!",
			ParkVisited:        "Park! Score: %d",
			ParkAlreadyVisited: "Already visited",
			Victory:            "Victory! All %d parks!",
			OutOfBattery:       "No battery!",
			Stranded:           "Stranded!",
			CantMove:           "Can't move!",
			BatteryStatus:      "Battery: %d/%d",
			HitWall:            "Hit wall!",
		},
	}

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Move to the only park
	engine.Move("right") // Should visit park and trigger victory

	if !engine.IsVictory() {
		t.Error("Expected victory after visiting all parks")
	}
	if !engine.IsGameOver() {
		t.Error("Expected game to be over after victory")
	}
	if engine.GetScore() != 1 {
		t.Errorf("Expected score 1 after victory, got %d", engine.GetScore())
	}
	if engine.GetRemainingParks() != 0 {
		t.Errorf("Expected 0 remaining parks after victory, got %d", engine.GetRemainingParks())
	}
}

func TestEngine_StateConsistency(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test that engine methods are consistent with direct state access
	state := engine.GetState()

	if engine.GetBattery() != state.Battery {
		t.Error("GetBattery() inconsistent with state.Battery")
	}
	if engine.GetScore() != state.Score {
		t.Error("GetScore() inconsistent with state.Score")
	}
	if engine.GetPlayerPosition() != state.PlayerPos {
		t.Error("GetPlayerPosition() inconsistent with state.PlayerPos")
	}
	if engine.IsGameOver() != state.GameOver {
		t.Error("IsGameOver() inconsistent with state.GameOver")
	}
	if engine.IsVictory() != state.Victory {
		t.Error("IsVictory() inconsistent with state.Victory")
	}

	// Test that moves through engine update state consistently
	engine.Move("right")
	newState := engine.GetState()

	if len(engine.GetMoveHistory()) != len(newState.MoveHistory) {
		t.Error("GetMoveHistory() inconsistent with state.MoveHistory")
	}
	if engine.GetBattery() != newState.Battery {
		t.Error("Battery inconsistent after move")
	}
}

func TestEngine_ErrorHandling(t *testing.T) {
	config := createTestConfig()
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test moves when game is over
	state := engine.GetState()
	state.GameOver = true

	success := engine.Move("right")
	if success {
		t.Error("Expected move to fail when game is over")
	}

	// Test current segment is empty after reset (global history persists)
	engine.Reset()
	state = engine.GetState()
	if len(state.CurrentMoves) != 0 || state.CurrentMovesCount != 0 {
		t.Error("Expected no current moves immediately after reset")
	}

	// Test moves with invalid directions
	success = engine.Move("invalid")
	if success {
		t.Error("Expected move to fail with invalid direction")
	}

	success = engine.Move("")
	if success {
		t.Error("Expected move to fail with empty direction")
	}
}
