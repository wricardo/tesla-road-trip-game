package engine

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCellTypeConstants(t *testing.T) {
	tests := []struct {
		cellType CellType
		expected string
	}{
		{Road, "road"},
		{Home, "home"},
		{Park, "park"},
		{Supercharger, "supercharger"},
		{Water, "water"},
		{Building, "building"},
	}

	for _, test := range tests {
		if string(test.cellType) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.cellType))
		}
	}
}

func TestValidationConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   int
		expected int
	}{
		{"MinGridSize", MinGridSize, 5},
		{"MaxGridSize", MaxGridSize, 50},
		{"MinBattery", MinBattery, 1},
		{"MaxBattery", MaxBattery, 100},
		{"MaxBulkMoves", MaxBulkMoves, 50},
		{"UnreachableDistance", UnreachableDistance, 999999},
		{"WebSocketBufferSize", WebSocketBufferSize, 256},
	}

	for _, test := range tests {
		if test.actual != test.expected {
			t.Errorf("%s: expected %d, got %d", test.name, test.expected, test.actual)
		}
	}
}

func TestCellJSONMarshaling(t *testing.T) {
	cell := Cell{
		Type:    Park,
		Visited: true,
		ID:      "park_1",
	}

	data, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("Failed to marshal cell: %v", err)
	}

	var unmarshaled Cell
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal cell: %v", err)
	}

	if unmarshaled.Type != cell.Type {
		t.Errorf("Type: expected %v, got %v", cell.Type, unmarshaled.Type)
	}
	if unmarshaled.Visited != cell.Visited {
		t.Errorf("Visited: expected %v, got %v", cell.Visited, unmarshaled.Visited)
	}
	if unmarshaled.ID != cell.ID {
		t.Errorf("ID: expected %v, got %v", cell.ID, unmarshaled.ID)
	}
}

func TestPositionJSONMarshaling(t *testing.T) {
	pos := Position{X: 10, Y: 25}

	data, err := json.Marshal(pos)
	if err != nil {
		t.Fatalf("Failed to marshal position: %v", err)
	}

	var unmarshaled Position
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal position: %v", err)
	}

	if unmarshaled.X != pos.X {
		t.Errorf("X: expected %d, got %d", pos.X, unmarshaled.X)
	}
	if unmarshaled.Y != pos.Y {
		t.Errorf("Y: expected %d, got %d", pos.Y, unmarshaled.Y)
	}
}

func TestMoveHistoryEntryJSONMarshaling(t *testing.T) {
	timestamp := time.Now().Unix()
	entry := MoveHistoryEntry{
		Action:       "right",
		FromPosition: Position{X: 5, Y: 7},
		ToPosition:   Position{X: 6, Y: 7},
		Battery:      18,
		Timestamp:    timestamp,
		Success:      true,
		MoveNumber:   1,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal move history entry: %v", err)
	}

	var unmarshaled MoveHistoryEntry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal move history entry: %v", err)
	}

	if unmarshaled.Action != entry.Action {
		t.Errorf("Action: expected %v, got %v", entry.Action, unmarshaled.Action)
	}
	if unmarshaled.FromPosition != entry.FromPosition {
		t.Errorf("FromPosition: expected %v, got %v", entry.FromPosition, unmarshaled.FromPosition)
	}
	if unmarshaled.ToPosition != entry.ToPosition {
		t.Errorf("ToPosition: expected %v, got %v", entry.ToPosition, unmarshaled.ToPosition)
	}
	if unmarshaled.Battery != entry.Battery {
		t.Errorf("Battery: expected %d, got %d", entry.Battery, unmarshaled.Battery)
	}
	if unmarshaled.Timestamp != entry.Timestamp {
		t.Errorf("Timestamp: expected %d, got %d", entry.Timestamp, unmarshaled.Timestamp)
	}
	if unmarshaled.Success != entry.Success {
		t.Errorf("Success: expected %v, got %v", entry.Success, unmarshaled.Success)
	}
	if unmarshaled.MoveNumber != entry.MoveNumber {
		t.Errorf("MoveNumber: expected %d, got %d", entry.MoveNumber, unmarshaled.MoveNumber)
	}
}

func TestGameStateJSONMarshaling(t *testing.T) {
	grid := [][]Cell{
		{{Type: Building}, {Type: Road}},
		{{Type: Home}, {Type: Park, ID: "park_0"}},
	}

	state := GameState{
		Grid:         grid,
		PlayerPos:    Position{X: 1, Y: 0},
		Battery:      15,
		MaxBattery:   20,
		Score:        1,
		VisitedParks: map[string]bool{"park_0": true},
		Message:      "Test message",
		GameOver:     false,
		Victory:      false,
		ConfigName:   "test_config",
		MoveHistory:  []MoveHistoryEntry{},
		TotalMoves:   0,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal game state: %v", err)
	}

	var unmarshaled GameState
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal game state: %v", err)
	}

	// Verify basic fields
	if unmarshaled.PlayerPos != state.PlayerPos {
		t.Errorf("PlayerPos: expected %v, got %v", state.PlayerPos, unmarshaled.PlayerPos)
	}
	if unmarshaled.Battery != state.Battery {
		t.Errorf("Battery: expected %d, got %d", state.Battery, unmarshaled.Battery)
	}
	if unmarshaled.Score != state.Score {
		t.Errorf("Score: expected %d, got %d", state.Score, unmarshaled.Score)
	}
	if unmarshaled.Message != state.Message {
		t.Errorf("Message: expected %s, got %s", state.Message, unmarshaled.Message)
	}

	// Verify grid structure
	if len(unmarshaled.Grid) != len(state.Grid) {
		t.Errorf("Grid rows: expected %d, got %d", len(state.Grid), len(unmarshaled.Grid))
	}
	if len(unmarshaled.Grid[0]) != len(state.Grid[0]) {
		t.Errorf("Grid cols: expected %d, got %d", len(state.Grid[0]), len(unmarshaled.Grid[0]))
	}

	// Verify visited parks map
	if len(unmarshaled.VisitedParks) != len(state.VisitedParks) {
		t.Errorf("VisitedParks length: expected %d, got %d", len(state.VisitedParks), len(unmarshaled.VisitedParks))
	}
	for key, value := range state.VisitedParks {
		if unmarshaled.VisitedParks[key] != value {
			t.Errorf("VisitedParks[%s]: expected %v, got %v", key, value, unmarshaled.VisitedParks[key])
		}
	}
}

func TestSurroundingCellJSONMarshaling(t *testing.T) {
	cell := SurroundingCell{
		X:    5,
		Y:    10,
		Type: Water,
	}

	data, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("Failed to marshal surrounding cell: %v", err)
	}

	var unmarshaled SurroundingCell
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal surrounding cell: %v", err)
	}

	if unmarshaled.X != cell.X {
		t.Errorf("X: expected %d, got %d", cell.X, unmarshaled.X)
	}
	if unmarshaled.Y != cell.Y {
		t.Errorf("Y: expected %d, got %d", cell.Y, unmarshaled.Y)
	}
	if unmarshaled.Type != cell.Type {
		t.Errorf("Type: expected %v, got %v", cell.Type, unmarshaled.Type)
	}
}

func TestGameConfigJSONMarshaling(t *testing.T) {
	config := GameConfig{
		Name:              "Test Config",
		Description:       "A test configuration",
		GridSize:          10,
		MaxBattery:        25,
		StartingBattery:   20,
		Layout:            []string{"RRRR", "HPPP"},
		Legend:            map[string]string{"R": "road", "H": "home", "P": "park"},
		WallCrashEndsGame: true,
	}
	config.Messages.Welcome = "Welcome to the test!"

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal game config: %v", err)
	}

	var unmarshaled GameConfig
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal game config: %v", err)
	}

	if unmarshaled.Name != config.Name {
		t.Errorf("Name: expected %s, got %s", config.Name, unmarshaled.Name)
	}
	if unmarshaled.GridSize != config.GridSize {
		t.Errorf("GridSize: expected %d, got %d", config.GridSize, unmarshaled.GridSize)
	}
	if unmarshaled.MaxBattery != config.MaxBattery {
		t.Errorf("MaxBattery: expected %d, got %d", config.MaxBattery, unmarshaled.MaxBattery)
	}
	if unmarshaled.WallCrashEndsGame != config.WallCrashEndsGame {
		t.Errorf("WallCrashEndsGame: expected %v, got %v", config.WallCrashEndsGame, unmarshaled.WallCrashEndsGame)
	}
	if unmarshaled.Messages.Welcome != config.Messages.Welcome {
		t.Errorf("Messages.Welcome: expected %s, got %s", config.Messages.Welcome, unmarshaled.Messages.Welcome)
	}
}
