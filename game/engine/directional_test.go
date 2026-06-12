package engine

import (
	"testing"
)

// minConfig builds a minimal valid GameConfig for directional road tests.
func minConfig(layout []string, cellConfigs map[string]CellConfig) *GameConfig {
	size := len(layout)
	c := &GameConfig{
		Name:            "test",
		GridSize:        size,
		MaxBattery:      20,
		StartingBattery: 20,
		Layout:          layout,
		Legend: map[string]string{
			"R": "road", "H": "home", "P": "park",
			"S": "supercharger", "W": "water", "B": "building",
		},
		CellConfigs: cellConfigs,
	}
	c.WallCrashEndsGame = false
	return c
}

func TestDirectional_PlainRoadAllowsAll(t *testing.T) {
	cfg := minConfig([]string{
		"BBBBB",
		"BHRRB",
		"BPRRB",
		"BBBBB",
		"BBBBB",
	}, nil)
	gs := InitGameStateFromConfig(cfg)
	// plain road has no direction constraint
	cell := gs.Grid[1][2] // R
	if len(cell.AllowedDirections) != 0 {
		t.Fatalf("plain road should have no AllowedDirections, got %v", cell.AllowedDirections)
	}
	if !directionAllowed(cell, "north") || !directionAllowed(cell, "south") ||
		!directionAllowed(cell, "east") || !directionAllowed(cell, "west") {
		t.Fatal("plain road should allow all directions")
	}
}

func TestDirectional_CellConfigParsedIntoGrid(t *testing.T) {
	cfg := minConfig([]string{
		"BBBBB",
		"BH^PB",
		"BBBBB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	// ^ at (2,1)
	gs := InitGameStateFromConfig(cfg)
	cell := gs.Grid[1][2]
	if cell.Type != Road {
		t.Fatalf("expected road, got %s", cell.Type)
	}
	if len(cell.AllowedDirections) != 1 || cell.AllowedDirections[0] != "north" {
		t.Fatalf("expected [north], got %v", cell.AllowedDirections)
	}
}

func TestDirectional_ExitConstraintBlocked(t *testing.T) {
	// Player on a north-only cell should not be able to move south/east/west
	cfg := minConfig([]string{
		"BBBBB",
		"BHR^B",
		"BRPRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	gs := InitGameStateFromConfig(cfg)
	// Manually place player on ^ at (3,1)
	gs.PlayerPos = Position{X: 3, Y: 1}

	if gs.MovePlayer("down", cfg) {
		t.Fatal("south move from north-only cell should be blocked")
	}
	if gs.MovePlayer("left", cfg) {
		t.Fatal("west move from north-only cell should be blocked")
	}
	if gs.MovePlayer("right", cfg) {
		t.Fatal("east move from north-only cell should be blocked (and OOB)")
	}
}

func TestDirectional_ExitConstraintAllowed(t *testing.T) {
	cfg2 := minConfig([]string{
		"BRRRB",
		"BHR^B",
		"BBBBB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	gs2 := InitGameStateFromConfig(cfg2)
	gs2.PlayerPos = Position{X: 3, Y: 1}
	if !gs2.MovePlayer("up", cfg2) {
		t.Fatal("north move from north-only cell should succeed")
	}
	if gs2.PlayerPos.Y != 0 || gs2.PlayerPos.X != 3 {
		t.Fatalf("expected (3,0), got %v", gs2.PlayerPos)
	}
}

func TestDirectional_EntryConstraintBlocked(t *testing.T) {
	// north-only cell at (2,2): can only enter moving north (from row 3 → row 2 going up)
	// Player tries to enter from left (moving right) or from above (moving down)
	cfg := minConfig([]string{
		"BRRRB",
		"BHR^B",
		"BRRRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	// Approach from west (player at (2,1), moving right into (3,1))
	gs := InitGameStateFromConfig(cfg)
	gs.PlayerPos = Position{X: 2, Y: 1}
	if gs.MovePlayer("right", cfg) {
		t.Fatal("entering north-only cell moving east should be blocked")
	}
	// Approach from north (player at (3,0), moving down into (3,1))
	gs.PlayerPos = Position{X: 3, Y: 0}
	if gs.MovePlayer("down", cfg) {
		t.Fatal("entering north-only cell moving south should be blocked")
	}
}

func TestDirectional_LTurnNorthEast(t *testing.T) {
	// J cell allows north and east only
	cfg := minConfig([]string{
		"BRRRB",
		"BHrJB",
		"BRRRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"J": {Type: "road", AllowedDirections: []string{"north", "east"}},
		"r": {Type: "road"},
	})
	gs := InitGameStateFromConfig(cfg)
	// Player on J at (3,1): can only exit north or east
	gs.PlayerPos = Position{X: 3, Y: 1}

	if gs.MovePlayer("down", cfg) {
		t.Fatal("south exit from NE L-turn should be blocked")
	}
	if gs.MovePlayer("left", cfg) {
		t.Fatal("west exit from NE L-turn should be blocked")
	}

	// north: row 0 col 3 is R, allowed
	if !gs.MovePlayer("up", cfg) {
		t.Fatal("north exit from NE L-turn should succeed")
	}
}

func TestDirectional_CanMoveToEnforcesDirectionConstraints(t *testing.T) {
	cfg := minConfig([]string{
		"BRRRB",
		"BHR^B",
		"BRRRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	gs := InitGameStateFromConfig(cfg)

	// Legacy passability-only check remains true for a direction-constrained road.
	gs.PlayerPos = Position{X: 2, Y: 1}
	if !gs.CanMoveTo(3, 1) {
		t.Fatal("CanMoveTo without direction should only check passability")
	}

	// Entry into a north-only cell while moving east is rejected.
	if gs.CanMoveTo(3, 1, "right") {
		t.Fatal("CanMoveTo should reject entry when destination disallows move direction")
	}

	// Entry while moving north is allowed.
	gs.PlayerPos = Position{X: 3, Y: 2}
	if !gs.CanMoveTo(3, 1, "up") {
		t.Fatal("CanMoveTo should allow entry when destination allows move direction")
	}

	// Exit from a north-only cell while moving west is rejected.
	gs.PlayerPos = Position{X: 3, Y: 1}
	if gs.CanMoveTo(2, 1, "left") {
		t.Fatal("CanMoveTo should reject exit when current cell disallows move direction")
	}
}

func TestDirectional_GameEngineCanMoveUsesDirectionConstraints(t *testing.T) {
	cfg := minConfig([]string{
		"BRRRB",
		"BHR^B",
		"BRRRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"north"}},
	})
	engine := &GameEngine{config: cfg, state: InitGameStateFromConfig(cfg)}
	engine.state.PlayerPos = Position{X: 2, Y: 1}

	if engine.CanMove("right") {
		t.Fatal("GameEngine.CanMove should reject moves blocked by destination direction constraints")
	}

	engine.state.PlayerPos = Position{X: 3, Y: 2}
	if !engine.CanMove("up") {
		t.Fatal("GameEngine.CanMove should allow moves permitted by destination direction constraints")
	}
}

func TestDirectional_ValidateConfigRejectsInvalidDirection(t *testing.T) {
	cfg := minConfig([]string{
		"BBBBB",
		"BHRRB",
		"BPRRB",
		"BBBBB",
		"BBBBB",
	}, map[string]CellConfig{
		"^": {Type: "road", AllowedDirections: []string{"diagonal"}},
	})
	if err := ValidateGameConfig(cfg); err == nil {
		t.Fatal("expected validation error for invalid direction 'diagonal'")
	}
}

func TestDirectional_ExistingMapsUnaffected(t *testing.T) {
	cfg := minConfig([]string{
		"BBBBBBBBBB",
		"BRRRRSRRRB",
		"BRPRRRRPRB",
		"BRRRRRRRRB",
		"BSRHHHRRBB",
		"BRRHHHRRBB",
		"BRRRRRRRBB",
		"BRPRRRRPRB",
		"BRRRRSRRRB",
		"BBBBBBBBBB",
	}, nil)
	cfg.GridSize = 10
	cfg.MaxBattery = 15
	cfg.StartingBattery = 15
	if err := ValidateGameConfig(cfg); err != nil {
		t.Fatalf("existing map should still validate: %v", err)
	}
	gs := InitGameStateFromConfig(cfg)
	// Every non-building cell should have empty AllowedDirections
	for y, row := range gs.Grid {
		for x, cell := range row {
			if cell.Type != Building && cell.Type != Water {
				if len(cell.AllowedDirections) != 0 {
					t.Errorf("cell (%d,%d) type=%s has unexpected AllowedDirections %v", x, y, cell.Type, cell.AllowedDirections)
				}
			}
		}
	}
}
