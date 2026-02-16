package engine

import "fmt"

// Engine provides the main interface for game operations
type Engine interface {
	// Game state management
	GetState() *GameState
	SetState(state *GameState) error
	Reset() *GameState
	IsGameOver() bool
	IsVictory() bool
	GetScore() int
	GetBattery() int
	GetPlayerPosition() Position

	// Movement operations
	Move(direction string) bool
	CanMove(direction string) bool
	GetPossibleMoves() []string

	// Configuration
	GetConfig() *GameConfig
	SetConfig(config *GameConfig) error

	// History
	GetMoveHistory() []MoveHistoryEntry
	GetLastMove() *MoveHistoryEntry

	// Local view
	GetLocalView() []SurroundingCell

	// Parks and objectives
	GetTotalParks() int
	GetVisitedParks() map[string]bool
	GetRemainingParks() int
}

// GameEngine implements the Engine interface
type GameEngine struct {
	state  *GameState
	config *GameConfig
}

// NewEngine creates a new game engine with the provided configuration
func NewEngine(config *GameConfig) (*GameEngine, error) {
	if err := ValidateGameConfig(config); err != nil {
		return nil, err
	}

	engine := &GameEngine{
		config: config,
		state:  InitGameStateFromConfig(config),
	}

	return engine, nil
}

// NewEngineWithDefaults creates a new game engine with default configuration
func NewEngineWithDefaults() *GameEngine {
	engine := &GameEngine{
		config: nil, // Will use defaults in InitGameStateFromConfig
	}
	engine.state = InitGameStateFromConfig(nil)
	return engine
}

// GetState returns the current game state
func (e *GameEngine) GetState() *GameState {
	return e.state
}

// SetState sets the game state (used for persistence loading)
func (e *GameEngine) SetState(state *GameState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	e.state = state
	return nil
}

// Reset resets the game to initial state
func (e *GameEngine) Reset() *GameState {
	// Preserve cumulative history and totals across resets
	prevHistory := e.state.MoveHistory
	prevTotal := e.state.TotalMoves

	// Reinitialize core state from config
	e.state = InitGameStateFromConfig(e.config)

	// Restore cumulative history and totals; clear only the current segment
	e.state.MoveHistory = prevHistory
	e.state.TotalMoves = prevTotal
	e.state.CurrentMoves = []MoveHistoryEntry{}
	e.state.CurrentMovesCount = 0

	return e.state
}

// IsGameOver returns whether the game is over
func (e *GameEngine) IsGameOver() bool {
	return e.state.GameOver
}

// IsVictory returns whether the player has won
func (e *GameEngine) IsVictory() bool {
	return e.state.Victory
}

// GetScore returns the current score
func (e *GameEngine) GetScore() int {
	return e.state.Score
}

// GetBattery returns the current battery level
func (e *GameEngine) GetBattery() int {
	return e.state.Battery
}

// GetPlayerPosition returns the current player position
func (e *GameEngine) GetPlayerPosition() Position {
	return e.state.PlayerPos
}

// Move attempts to move the player in the specified direction
func (e *GameEngine) Move(direction string) bool {
	if e.config == nil {
		return false
	}

	// Store previous position for history
	prevPos := e.state.PlayerPos
	success := e.state.MovePlayer(direction, e.config)

	// Add to history
	e.state.AddMoveToHistory(direction, prevPos, e.state.PlayerPos, success)

	return success
}

// CanMove checks if the player can move in the specified direction
func (e *GameEngine) CanMove(direction string) bool {
	if e.state.GameOver {
		return false
	}

	newX, newY := e.state.PlayerPos.X, e.state.PlayerPos.Y

	switch direction {
	case "up":
		newY--
	case "down":
		newY++
	case "left":
		newX--
	case "right":
		newX++
	default:
		return false
	}

	return e.state.CanMoveTo(newX, newY) && e.state.Battery > 0
}

// GetPossibleMoves returns all valid directions the player can move
func (e *GameEngine) GetPossibleMoves() []string {
	directions := []string{"up", "down", "left", "right"}
	var possible []string

	for _, dir := range directions {
		if e.CanMove(dir) {
			possible = append(possible, dir)
		}
	}

	return possible
}

// GetConfig returns the current game configuration
func (e *GameEngine) GetConfig() *GameConfig {
	return e.config
}

// SetConfig sets a new game configuration and resets the game
func (e *GameEngine) SetConfig(config *GameConfig) error {
	if err := ValidateGameConfig(config); err != nil {
		return err
	}

	e.config = config
	e.state = InitGameStateFromConfig(config)
	return nil
}

// GetMoveHistory returns the complete move history
func (e *GameEngine) GetMoveHistory() []MoveHistoryEntry {
	return e.state.MoveHistory
}

// GetLastMove returns the last move made, or nil if no moves
func (e *GameEngine) GetLastMove() *MoveHistoryEntry {
	if len(e.state.MoveHistory) == 0 {
		return nil
	}
	return &e.state.MoveHistory[len(e.state.MoveHistory)-1]
}

// GetLocalView returns the local view around the player
func (e *GameEngine) GetLocalView() []SurroundingCell {
	return e.state.GenerateLocalView()
}

// GetTotalParks returns the total number of parks in the game
func (e *GameEngine) GetTotalParks() int {
	return CountTotalParks(e.state.Grid)
}

// GetVisitedParks returns the map of visited parks
func (e *GameEngine) GetVisitedParks() map[string]bool {
	return e.state.VisitedParks
}

// GetRemainingParks returns the number of parks not yet visited
func (e *GameEngine) GetRemainingParks() int {
	return e.GetTotalParks() - len(e.state.VisitedParks)
}

// BulkMove executes multiple moves in sequence, returning success status for each
func (e *GameEngine) BulkMove(moves []string) []bool {
	results := make([]bool, 0, len(moves))

	for _, direction := range moves {
		// Stop if game is over
		if e.IsGameOver() {
			break
		}

		success := e.Move(direction)
		results = append(results, success)
	}

	return results
}
