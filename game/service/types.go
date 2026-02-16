package service

import (
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/engine"
)

// SessionInfo provides information about a game session
type SessionInfo struct {
	ID             string             `json:"id"`
	ConfigName     string             `json:"config_name"`
	CreatedAt      time.Time          `json:"created_at"`
	LastAccessedAt time.Time          `json:"last_accessed_at"`
	GameState      *engine.GameState  `json:"game_state"`
	GameConfig     *engine.GameConfig `json:"game_config"`
}

// MoveResult contains the result of a move operation
type MoveResult struct {
	Success     bool              `json:"success"`
	GameState   *engine.GameState `json:"game_state"`
	Message     string            `json:"message"`
	Events      []GameEvent       `json:"events,omitempty"`
	Step        *StepInfo         `json:"step,omitempty"`
	AttemptedTo *AttemptInfo      `json:"attempted_to,omitempty"`
}

// BulkMoveResult contains the result of multiple moves
type BulkMoveResult struct {
	// Summary
	MovesExecuted  int               `json:"moves_executed"`
	TotalMoves     int               `json:"total_moves"`     // Deprecated: kept for backward compatibility (same as requested_moves)
	RequestedMoves int               `json:"requested_moves"` // The number of moves requested in this call
	Success        bool              `json:"success"`
	GameState      *engine.GameState `json:"game_state"`
	Events         []GameEvent       `json:"events"`
	StoppedReason  string            `json:"stopped_reason,omitempty"`   // Human-readable reason
	StopReasonCode string            `json:"stop_reason_code,omitempty"` // Machine-friendly code: blocked_boundary|blocked_building|blocked_water|out_of_battery|stranded|game_over|victory
	StoppedOnMove  int               `json:"stopped_on_move,omitempty"`  // 1-based index of the move that caused stop
	Truncated      bool              `json:"truncated,omitempty"`
	Limit          int               `json:"limit,omitempty"`

	// Start/end snapshot
	StartPos     engine.Position `json:"start_pos"`
	EndPos       engine.Position `json:"end_pos"`
	StartBattery int             `json:"start_battery"`
	EndBattery   int             `json:"end_battery"`
	ScoreDelta   int             `json:"score_delta"`

	// Per-step compact trace (only for this call)
	Steps []StepInfo `json:"steps,omitempty"`

	// Failure diagnostics
	AttemptedTo *AttemptInfo `json:"attempted_to,omitempty"`

	// Final status aids
	GameOver      bool     `json:"game_over"`
	GameOverCode  string   `json:"game_over_code,omitempty"`
	Message       string   `json:"message,omitempty"`
	PossibleMoves []string `json:"possible_moves,omitempty"`
	LocalView3x3  []string `json:"local_view_3x3,omitempty"`
	BatteryRisk   string   `json:"battery_risk,omitempty"`
}

// StepInfo is a compact record for each executed move in the bulk call
type StepInfo struct {
	Idx           int             `json:"idx"`
	Dir           string          `json:"dir"`
	From          engine.Position `json:"from"`
	To            engine.Position `json:"to"`
	TileChar      string          `json:"tile_char"`
	TileType      string          `json:"tile_type"`
	BatteryBefore int             `json:"battery_before"`
	BatteryAfter  int             `json:"battery_after"`
	Success       bool            `json:"success"`
	Charged       bool            `json:"charged,omitempty"`
	Park          bool            `json:"park,omitempty"`
	Victory       bool            `json:"victory,omitempty"`
}

// AttemptInfo details the first failed target cell attempted
type AttemptInfo struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	TileChar string `json:"tile_char"`
	TileType string `json:"tile_type"`
	Passable bool   `json:"passable"`
}

// GameEvent represents an event that occurred during gameplay
type GameEvent struct {
	Type      string          `json:"type"` // "move", "charge", "park_visited", "game_over", "victory", "reset"
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"timestamp"`
	Position  engine.Position `json:"position,omitempty"`
}

// HistoryOptions configures move history retrieval
type HistoryOptions struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Order string `json:"order"` // "asc" or "desc"
}

// HistoryResponse contains paginated move history
type HistoryResponse struct {
	Moves       []engine.MoveHistoryEntry `json:"moves"`
	TotalMoves  int                       `json:"total_moves"`
	Page        int                       `json:"page"`
	PageSize    int                       `json:"page_size"`
	TotalPages  int                       `json:"total_pages"`
	HasNext     bool                      `json:"has_next"`
	HasPrevious bool                      `json:"has_previous"`
}

// ConfigInfo provides information about a game configuration
type ConfigInfo struct {
	Filename    string `json:"filename"`
	ConfigID    string `json:"config_id"` // The identifier to use for session creation
	Name        string `json:"name"`      // Display name
	Description string `json:"description"`
	GridSize    int    `json:"grid_size"`
	MaxBattery  int    `json:"max_battery"`
}
