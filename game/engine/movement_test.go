package engine

import (
	"strings"
	"testing"
	"time"
)

func createTestGameState() (*GameState, *GameConfig) {
	config := &GameConfig{
		Name:            "Test Config",
		Description:     "Test configuration for movement tests",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 5,
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
			Welcome:            "Welcome to test!",
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

	state := InitGameStateFromConfig(config)
	return state, config
}

func TestCanMoveTo_ValidPositions(t *testing.T) {
	state, _ := createTestGameState()

	tests := []struct {
		name     string
		x, y     int
		expected bool
	}{
		{"road cell", 1, 1, true},
		{"home cell", 2, 1, true},
		{"park cell", 3, 1, true},
		{"supercharger cell", 3, 2, true},
		{"water cell", 2, 2, false},
		{"building cell", 0, 0, false},
		{"out of bounds negative", -1, 0, false},
		{"out of bounds positive", 5, 0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := state.CanMoveTo(test.x, test.y)
			if result != test.expected {
				t.Errorf("CanMoveTo(%d, %d): expected %v, got %v", test.x, test.y, test.expected, result)
			}
		})
	}
}

func TestMovePlayer_BasicMovement(t *testing.T) {
	state, config := createTestGameState()
	initialPos := state.PlayerPos
	initialBattery := state.Battery

	// Test valid move
	success := state.MovePlayer("right", config)
	if !success {
		t.Error("Expected successful move")
	}
	if state.PlayerPos.X != initialPos.X+1 {
		t.Errorf("Expected X to be %d, got %d", initialPos.X+1, state.PlayerPos.X)
	}
	if state.PlayerPos.Y != initialPos.Y {
		t.Errorf("Expected Y to remain %d, got %d", initialPos.Y, state.PlayerPos.Y)
	}
	if state.Battery != initialBattery-1 {
		t.Errorf("Expected battery to decrease by 1, was %d now %d", initialBattery, state.Battery)
	}
}

func TestMovePlayer_DirectionMapping(t *testing.T) {
	tests := []struct {
		direction string
		deltaX    int
		deltaY    int
	}{
		{"up", 0, -1},
		{"down", 0, 1},
		{"left", -1, 0},
		{"right", 1, 0},
	}

	for _, test := range tests {
		t.Run(test.direction, func(t *testing.T) {
			state, config := createTestGameState()
			// Move to center of grid where all directions are valid
			state.PlayerPos = Position{X: 2, Y: 2}
			state.Grid[2][2] = Cell{Type: Road} // Ensure it's passable

			initialPos := state.PlayerPos
			state.MovePlayer(test.direction, config)

			expectedX := initialPos.X + test.deltaX
			expectedY := initialPos.Y + test.deltaY

			if state.PlayerPos.X != expectedX || state.PlayerPos.Y != expectedY {
				t.Errorf("Move %s: expected (%d,%d), got (%d,%d)",
					test.direction, expectedX, expectedY, state.PlayerPos.X, state.PlayerPos.Y)
			}
		})
	}
}

func TestMovePlayer_InvalidDirection(t *testing.T) {
	state, config := createTestGameState()
	initialPos := state.PlayerPos
	initialBattery := state.Battery

	success := state.MovePlayer("invalid", config)
	if success {
		t.Error("Expected move to fail for invalid direction")
	}
	if state.PlayerPos != initialPos {
		t.Error("Position should not change for invalid direction")
	}
	if state.Battery != initialBattery {
		t.Error("Battery should not change for invalid direction")
	}
}

func TestMovePlayer_WallCollision(t *testing.T) {
	state, config := createTestGameState()
	initialPos := state.PlayerPos
	initialBattery := state.Battery

	// Try to move into water
	success := state.MovePlayer("down", config)
	if success {
		t.Error("Expected move to fail when hitting water")
	}
	if state.PlayerPos != initialPos {
		t.Error("Position should not change when hitting obstacle")
	}
	if state.Battery != initialBattery {
		t.Error("Battery should not change when move fails")
	}
	if !strings.Contains(state.Message, "Can't move") {
		t.Errorf("Expected 'Can't move' message, got: %s", state.Message)
	}
}

func TestMovePlayer_WallCrashEndsGame(t *testing.T) {
	state, config := createTestGameState()
	config.WallCrashEndsGame = true

	// Try to move into water
	success := state.MovePlayer("down", config)
	if success {
		t.Error("Expected move to fail when hitting wall with crash ending game")
	}
	if !state.GameOver {
		t.Error("Expected game to be over after wall crash")
	}
	if !strings.Contains(state.Message, "Hit wall!") {
		t.Errorf("Expected hit wall message, got: %s", state.Message)
	}
}

func TestMovePlayer_OutOfBattery(t *testing.T) {
	state, config := createTestGameState()
	state.Battery = 0

	success := state.MovePlayer("right", config)
	if success {
		t.Error("Expected move to fail when out of battery")
	}
	if !state.GameOver {
		t.Error("Expected game to be over when out of battery")
	}
	if state.Message != config.Messages.OutOfBattery {
		t.Errorf("Expected out of battery message, got: %s", state.Message)
	}
}

func TestMovePlayer_GameOverState(t *testing.T) {
	state, config := createTestGameState()
	state.GameOver = true
	initialPos := state.PlayerPos

	success := state.MovePlayer("right", config)
	if success {
		t.Error("Expected move to fail when game is over")
	}
	if state.PlayerPos != initialPos {
		t.Error("Position should not change when game is over")
	}
}

func TestMovePlayer_HomeCharging(t *testing.T) {
	state, config := createTestGameState()
	state.Battery = 3 // Set battery below max

	// Player starts at home, move away and back
	state.MovePlayer("right", config)
	state.MovePlayer("left", config) // Back to home

	if state.Battery != config.MaxBattery {
		t.Errorf("Expected battery to be at max (%d) after visiting home, got %d", config.MaxBattery, state.Battery)
	}
	if state.Message != config.Messages.HomeCharge {
		t.Errorf("Expected home charge message, got: %s", state.Message)
	}
}

func TestMovePlayer_SuperchargerCharging(t *testing.T) {
	state, config := createTestGameState()
	state.Battery = 3 // Set battery below max

	// Move to supercharger position (3,2)
	state.PlayerPos = Position{X: 2, Y: 2} // Start adjacent to supercharger
	state.MovePlayer("right", config)      // Move to supercharger

	if state.Battery != config.MaxBattery {
		t.Errorf("Expected battery to be at max (%d) after visiting supercharger, got %d", config.MaxBattery, state.Battery)
	}
	if state.Message != config.Messages.SuperchargerCharge {
		t.Errorf("Expected supercharger message, got: %s", state.Message)
	}
}

func TestMovePlayer_ParkVisit(t *testing.T) {
	state, config := createTestGameState()
	initialScore := state.Score

	// Move to park position
	state.MovePlayer("right", config) // Move to park at (3,1)

	if state.Score != initialScore+1 {
		t.Errorf("Expected score to increase by 1, was %d now %d", initialScore, state.Score)
	}
	if !strings.Contains(state.Message, "Park visited") {
		t.Errorf("Expected park visited message, got: %s", state.Message)
	}

	// Check park is marked as visited
	parkCell := state.Grid[1][3]
	if !parkCell.Visited {
		t.Error("Expected park cell to be marked as visited")
	}
	if len(state.VisitedParks) != 1 {
		t.Errorf("Expected 1 visited park, got %d", len(state.VisitedParks))
	}
}

func TestMovePlayer_AlreadyVisitedPark(t *testing.T) {
	state, config := createTestGameState()

	// Visit park twice
	state.MovePlayer("right", config) // First visit
	initialScore := state.Score
	state.MovePlayer("left", config)  // Move away
	state.MovePlayer("right", config) // Second visit

	if state.Score != initialScore {
		t.Errorf("Expected score to remain %d, got %d", initialScore, state.Score)
	}
	if state.Message != config.Messages.ParkAlreadyVisited {
		t.Errorf("Expected already visited message, got: %s", state.Message)
	}
}

func TestMovePlayer_Victory(t *testing.T) {
	state, config := createTestGameState()

	// Visit all parks to trigger victory
	parks := []Position{{3, 1}, {1, 3}, {2, 3}, {3, 3}}
	for _, park := range parks {
		state.PlayerPos = Position{X: park.X - 1, Y: park.Y} // Position adjacent to park
		state.MovePlayer("right", config)
		if state.GameOver && state.Victory {
			break // Victory achieved early
		}
	}

	if !state.Victory {
		t.Error("Expected victory after visiting all parks")
	}
	if !state.GameOver {
		t.Error("Expected game to be over after victory")
	}
	if !strings.Contains(state.Message, "Victory") {
		t.Errorf("Expected victory message, got: %s", state.Message)
	}
}

func TestMovePlayer_Stranded(t *testing.T) {
	state, config := createTestGameState()

	// Position player away from chargers with 1 battery
	state.PlayerPos = Position{X: 1, Y: 3} // At a park, away from home/supercharger
	state.Battery = 1

	// Move to use last battery
	state.MovePlayer("right", config)

	if !state.GameOver {
		t.Error("Expected game to be over when stranded")
	}
	if state.Message != config.Messages.Stranded {
		t.Errorf("Expected stranded message, got: %s", state.Message)
	}
}

func TestCanReachCharger(t *testing.T) {
	state, _ := createTestGameState()

	tests := []struct {
		name     string
		pos      Position
		cellType CellType
		expected bool
	}{
		{"at home", Position{2, 1}, Home, true},
		{"at supercharger", Position{3, 2}, Supercharger, true},
		{"at road", Position{1, 1}, Road, false},
		{"at park", Position{3, 1}, Park, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state.PlayerPos = test.pos
			state.Grid[test.pos.Y][test.pos.X] = Cell{Type: test.cellType}

			result := state.CanReachCharger()
			if result != test.expected {
				t.Errorf("CanReachCharger at %v (%v): expected %v, got %v",
					test.pos, test.cellType, test.expected, result)
			}
		})
	}
}

func TestGenerateLocalView(t *testing.T) {
	state, _ := createTestGameState()
	state.PlayerPos = Position{X: 2, Y: 2} // Center position

	localView := state.GenerateLocalView()

	if len(localView) != 8 {
		t.Errorf("Expected 8 surrounding cells, got %d", len(localView))
	}

	// Test that all 8 directions are covered
	expectedPositions := []Position{
		{2, 1}, {3, 1}, {3, 2}, {3, 3},
		{2, 3}, {1, 3}, {1, 2}, {1, 1},
	}

	for i, expected := range expectedPositions {
		if localView[i].X != expected.X || localView[i].Y != expected.Y {
			t.Errorf("Local view position %d: expected (%d,%d), got (%d,%d)",
				i, expected.X, expected.Y, localView[i].X, localView[i].Y)
		}
	}

	// Test out of bounds handling
	state.PlayerPos = Position{X: 0, Y: 0} // Corner position
	localView = state.GenerateLocalView()

	// Some cells should be marked as Building (out of bounds)
	outOfBoundsCount := 0
	for _, cell := range localView {
		if cell.Type == Building && (cell.X < 0 || cell.Y < 0) {
			outOfBoundsCount++
		}
	}
	if outOfBoundsCount == 0 {
		t.Error("Expected some out-of-bounds cells to be marked as Building")
	}
}

func TestAddMoveToHistory(t *testing.T) {
	state, _ := createTestGameState()

	fromPos := Position{X: 1, Y: 1}
	toPos := Position{X: 2, Y: 1}

	// Record time before adding move
	beforeTime := time.Now().Unix()

	state.AddMoveToHistory("right", fromPos, toPos, true)

	// Record time after adding move
	afterTime := time.Now().Unix()

	if len(state.MoveHistory) != 1 {
		t.Errorf("Expected 1 move in history, got %d", len(state.MoveHistory))
	}
	if state.TotalMoves != 1 {
		t.Errorf("Expected total moves to be 1, got %d", state.TotalMoves)
	}

	move := state.MoveHistory[0]
	if move.Action != "right" {
		t.Errorf("Expected action 'right', got '%s'", move.Action)
	}
	if move.FromPosition != fromPos {
		t.Errorf("Expected from position %v, got %v", fromPos, move.FromPosition)
	}
	if move.ToPosition != toPos {
		t.Errorf("Expected to position %v, got %v", toPos, move.ToPosition)
	}
	if move.Battery != state.Battery {
		t.Errorf("Expected battery %d, got %d", state.Battery, move.Battery)
	}
	if !move.Success {
		t.Error("Expected success to be true")
	}
	if move.MoveNumber != 1 {
		t.Errorf("Expected move number 1, got %d", move.MoveNumber)
	}
	if move.Timestamp < beforeTime || move.Timestamp > afterTime {
		t.Errorf("Expected timestamp between %d and %d, got %d", beforeTime, afterTime, move.Timestamp)
	}

	// Add another move to test incrementing
	state.AddMoveToHistory("left", toPos, fromPos, false)

	if len(state.MoveHistory) != 2 {
		t.Errorf("Expected 2 moves in history, got %d", len(state.MoveHistory))
	}
	if state.TotalMoves != 2 {
		t.Errorf("Expected total moves to be 2, got %d", state.TotalMoves)
	}

	secondMove := state.MoveHistory[1]
	if secondMove.MoveNumber != 2 {
		t.Errorf("Expected second move number 2, got %d", secondMove.MoveNumber)
	}
	if secondMove.Success {
		t.Error("Expected second move success to be false")
	}
}

func TestCountTotalParks(t *testing.T) {
	// Create a test grid with known number of parks
	grid := [][]Cell{
		{{Type: Building}, {Type: Road}, {Type: Park}},
		{{Type: Home}, {Type: Water}, {Type: Park}},
		{{Type: Park}, {Type: Supercharger}, {Type: Building}},
	}

	count := CountTotalParks(grid)
	expected := 3
	if count != expected {
		t.Errorf("Expected %d parks, got %d", expected, count)
	}

	// Test empty grid
	emptyGrid := [][]Cell{
		{{Type: Building}, {Type: Road}},
		{{Type: Home}, {Type: Water}},
	}

	count = CountTotalParks(emptyGrid)
	if count != 0 {
		t.Errorf("Expected 0 parks in empty grid, got %d", count)
	}
}
