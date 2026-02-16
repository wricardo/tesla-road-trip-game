package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/engine"
)

// gameServiceImpl implements the GameService interface
type gameServiceImpl struct {
	sessions SessionManager
	configs  ConfigManager
	mu       sync.RWMutex
}

// getConfigID returns the config_id for a given config name, used for consistent API responses
func (s *gameServiceImpl) getConfigID(configName string) string {
	availableConfigs, err := s.configs.ListConfigs()
	if err == nil {
		for _, cfg := range availableConfigs {
			if cfg.Name == configName {
				return cfg.ConfigID
			}
		}
	}
	// Fallback: return as-is or "default"
	if configName == "" {
		return "default"
	}
	return configName
}

// NewGameService creates a new game service instance
func NewGameService(sessions SessionManager, configs ConfigManager) GameService {
	return &gameServiceImpl{
		sessions: sessions,
		configs:  configs,
	}
}

// CreateSession creates a new game session
func (s *gameServiceImpl) CreateSession(ctx context.Context, configName string) (*SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load configuration
	var config *engine.GameConfig
	var err error
	if configName != "" {
		config, err = s.configs.LoadConfig(configName)
		if err != nil {
			// Provide helpful error message with available options
			if strings.Contains(err.Error(), "configuration not found") {
				availableConfigs, listErr := s.configs.ListConfigs()
				if listErr == nil && len(availableConfigs) > 0 {
					var configIDs []string
					for _, cfg := range availableConfigs {
						configIDs = append(configIDs, cfg.ConfigID)
					}
					return nil, fmt.Errorf("config '%s' not found. Available configs: %v", configName, configIDs)
				}
				return nil, fmt.Errorf("config '%s' not found. Use /api/configs to list available configurations", configName)
			}
			return nil, fmt.Errorf("failed to load config %s: %w", configName, err)
		}
	} else {
		config = s.configs.GetDefault()
	}

	// Let session manager generate a proper 4-character ID
	session, err := s.sessions.Create("", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Determine the config identifier to return - prefer the input configName if provided,
	// otherwise look up the config_id by display name
	configID := configName
	if configID == "" {
		configID = s.getConfigID(config.Name)
	}

	return &SessionInfo{
		ID:             session.ID,
		ConfigName:     configID, // Return the config_id, not the display name
		CreatedAt:      session.CreatedAt,
		LastAccessedAt: session.LastAccessedAt,
		GameState:      session.Engine.GetState(),
		GameConfig:     session.Config,
	}, nil
}

// GetSession retrieves session information
func (s *gameServiceImpl) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	s.sessions.UpdateLastAccessed(sessionID)

	return &SessionInfo{
		ID:             session.ID,
		ConfigName:     s.getConfigID(session.Config.Name), // Return config_id consistently
		CreatedAt:      session.CreatedAt,
		LastAccessedAt: session.LastAccessedAt,
		GameState:      session.Engine.GetState(),
		GameConfig:     session.Config,
	}, nil
}

// ListSessions returns all active sessions
func (s *gameServiceImpl) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := s.sessions.List()
	result := make([]*SessionInfo, 0, len(sessions))

	for _, sess := range sessions {
		result = append(result, &SessionInfo{
			ID:             sess.ID,
			ConfigName:     s.getConfigID(sess.Config.Name), // Return config_id consistently
			CreatedAt:      sess.CreatedAt,
			LastAccessedAt: sess.LastAccessedAt,
			GameState:      sess.Engine.GetState(),
			GameConfig:     sess.Config,
		})
	}

	return result, nil
}

// DeleteSession removes a session
func (s *gameServiceImpl) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessions.Delete(sessionID)
}

// Move executes a single move for a session
func (s *gameServiceImpl) Move(ctx context.Context, sessionID, direction string, reset bool) (*MoveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get session
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Update last accessed time
	s.sessions.UpdateLastAccessed(sessionID)

	// Collect events
	events := []GameEvent{}

	// Handle reset if requested
	if reset {
		sess.Engine.Reset()
		events = append(events, GameEvent{
			Type:      "reset",
			Message:   "Game reset to initial state",
			Timestamp: time.Now(),
		})
	}

	// Execute move
	prevPos := sess.Engine.GetPlayerPosition()
	prevState := sess.Engine.GetState()
	prevBattery := prevState.Battery
	success := sess.Engine.Move(direction)
	newPos := sess.Engine.GetPlayerPosition()
	state := sess.Engine.GetState()

	// Build result
	result := &MoveResult{
		Success:   success,
		GameState: state,
		Message:   state.Message,
		Events:    events,
	}

	// Add move event
	if success {
		moveEvents := s.extractMoveEvents(sess, prevPos, newPos, direction)
		result.Events = append(result.Events, moveEvents...)

		// Fill compact step info
		tileChar, tileType := "", ""
		if newPos.Y >= 0 && newPos.Y < len(state.Grid) && newPos.X >= 0 && newPos.X < len(state.Grid[0]) {
			tileChar, tileType = mapCellToCharAndType(state.Grid[newPos.Y][newPos.X])
		}
		charged := false
		park := false
		victory := false
		for _, ev := range moveEvents {
			switch ev.Type {
			case "charge":
				charged = true
			case "park_visited":
				park = true
			case "victory":
				victory = true
			}
		}
		result.Step = &StepInfo{
			Idx:           1,
			Dir:           direction,
			From:          prevPos,
			To:            newPos,
			TileChar:      tileChar,
			TileType:      tileType,
			BatteryBefore: prevBattery,
			BatteryAfter:  state.Battery,
			Success:       true,
			Charged:       charged,
			Park:          park,
			Victory:       victory,
		}
	} else {
		// Attempted target
		attemptedX, attemptedY := prevPos.X, prevPos.Y
		switch strings.ToLower(direction) {
		case "up":
			attemptedY--
		case "down":
			attemptedY++
		case "left":
			attemptedX--
		case "right":
			attemptedX++
		}
		gridH := len(state.Grid)
		var tileChar, tileType string
		passable := false
		if attemptedX < 0 || attemptedY < 0 || attemptedY >= gridH || (gridH > 0 && attemptedX >= len(state.Grid[0])) {
			tileChar = "B"
			tileType = "boundary"
		} else {
			cell := state.Grid[attemptedY][attemptedX]
			tileChar, tileType = mapCellToCharAndType(cell)
			passable = cell.Type != engine.Water && cell.Type != engine.Building
		}
		result.AttemptedTo = &AttemptInfo{X: attemptedX, Y: attemptedY, TileChar: tileChar, TileType: tileType, Passable: passable}
	}

	// Enrich state with decision aids
	state.LocalView3x3 = buildLocal3x3(state)
	state.BatteryRisk = riskCode(engine.AnalyzeBatteryRisk(state))

	// Auto-save session after move
	if err := s.sessions.Save(sessionID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after move: %v\n", sessionID, err)
	}

	return result, nil
}

// BulkMove executes multiple moves in sequence
func (s *gameServiceImpl) BulkMove(ctx context.Context, sessionID string, moves []string, reset bool) (*BulkMoveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Update last accessed
	s.sessions.UpdateLastAccessed(sessionID)

	// Initialize result and capture start snapshot
	state := sess.Engine.GetState()
	startPos := state.PlayerPos
	startBattery := state.Battery
	startScore := state.Score

	result := &BulkMoveResult{
		RequestedMoves: len(moves),
		TotalMoves:     len(moves), // backward-compat: mirrors requested_moves
		Events:         make([]GameEvent, 0),
		Success:        true,
		StartPos:       startPos,
		StartBattery:   startBattery,
		GameOver:       state.GameOver,
		Message:        state.Message,
	}

	// Handle reset
	if reset {
		sess.Engine.Reset()
		result.Events = append(result.Events, GameEvent{
			Type:      "reset",
			Message:   "Game reset to initial state",
			Timestamp: time.Now(),
		})
	}

	// Limit moves to prevent abuse
	if len(moves) > engine.MaxBulkMoves {
		result.Truncated = true
		result.Limit = engine.MaxBulkMoves
		moves = moves[:engine.MaxBulkMoves]
	}

	// Execute moves
	for i, move := range moves {
		if sess.Engine.IsGameOver() {
			result.StoppedReason = "game_over"
			result.StopReasonCode = "game_over"
			result.StoppedOnMove = result.MovesExecuted + 1
			break
		}

		prevPos := sess.Engine.GetPlayerPosition()
		prevState := sess.Engine.GetState()
		prevBattery := prevState.Battery
		success := sess.Engine.Move(move)

		if !success {
			result.Success = false
			result.StoppedReason = fmt.Sprintf("move %d blocked: %s", i+1, move)
			result.StoppedOnMove = i + 1

			// Determine attempted target and reason code
			attemptedX, attemptedY := prevPos.X, prevPos.Y
			switch strings.ToLower(move) {
			case "up":
				attemptedY--
			case "down":
				attemptedY++
			case "left":
				attemptedX--
			case "right":
				attemptedX++
			}

			st := sess.Engine.GetState()
			gridH := len(st.Grid)
			var tileChar, tileType string
			passable := false
			if attemptedX < 0 || attemptedY < 0 || attemptedY >= gridH || (gridH > 0 && attemptedX >= len(st.Grid[0])) {
				tileChar = "B" // treat boundary as wall-like
				tileType = "boundary"
				result.StopReasonCode = "blocked_boundary"
			} else {
				cell := st.Grid[attemptedY][attemptedX]
				tileChar, tileType = mapCellToCharAndType(cell)
				passable = cell.Type != engine.Water && cell.Type != engine.Building
				if !passable {
					if cell.Type == engine.Water {
						result.StopReasonCode = "blocked_water"
					} else if cell.Type == engine.Building {
						result.StopReasonCode = "blocked_building"
					}
				} else if prevBattery <= 0 {
					result.StopReasonCode = "out_of_battery"
				} else if st.GameOver {
					result.StopReasonCode = "game_over"
				}
			}
			result.AttemptedTo = &AttemptInfo{
				X:        attemptedX,
				Y:        attemptedY,
				TileChar: tileChar,
				TileType: tileType,
				Passable: passable,
			}
			break
		}

		result.MovesExecuted++
		newPos := sess.Engine.GetPlayerPosition()

		// Collect events for this move
		events := s.extractMoveEvents(sess, prevPos, newPos, move)
		result.Events = append(result.Events, events...)

		// Build step info for this executed move
		currState := sess.Engine.GetState()
		batteryAfter := currState.Battery
		tileChar, tileType := "", ""
		if newPos.Y >= 0 && newPos.Y < len(currState.Grid) && newPos.X >= 0 && newPos.X < len(currState.Grid[0]) {
			tileChar, tileType = mapCellToCharAndType(currState.Grid[newPos.Y][newPos.X])
		}
		charged := false
		park := false
		victory := false
		for _, ev := range events {
			switch ev.Type {
			case "charge":
				charged = true
			case "park_visited":
				park = true
			case "victory":
				victory = true
			}
		}
		step := StepInfo{
			Idx:           i + 1,
			Dir:           move,
			From:          prevPos,
			To:            newPos,
			TileChar:      tileChar,
			TileType:      tileType,
			BatteryBefore: prevBattery,
			BatteryAfter:  batteryAfter,
			Success:       true,
			Charged:       charged,
			Park:          park,
			Victory:       victory,
		}
		result.Steps = append(result.Steps, step)
	}

	result.GameState = sess.Engine.GetState()
	// Ensure backward-compat mirror
	result.TotalMoves = len(moves)

	// Finalize snapshots
	endState := result.GameState
	result.EndPos = endState.PlayerPos
	result.EndBattery = endState.Battery
	result.ScoreDelta = endState.Score - startScore
	result.GameOver = endState.GameOver
	result.Message = endState.Message

	// If we ended due to game over without explicit stop reason code
	if result.GameOver && result.StopReasonCode == "" {
		if endState.Victory {
			result.StopReasonCode = "victory"
			result.GameOverCode = "victory"
		} else if endState.Battery == 0 {
			// Determine stranded vs out_of_battery by checking if we executed a move to 0 battery
			if result.MovesExecuted > 0 {
				// Last executed step battery_after should equal endState.Battery
				last := result.Steps[len(result.Steps)-1]
				if last.BatteryAfter == 0 {
					// Stranded if not on charger
					currCell := endState.Grid[endState.PlayerPos.Y][endState.PlayerPos.X]
					if currCell.Type != engine.Home && currCell.Type != engine.Supercharger {
						result.StopReasonCode = "stranded"
						result.GameOverCode = "stranded"
					} else {
						result.StopReasonCode = "game_over"
						result.GameOverCode = "game_over"
					}
				} else {
					result.StopReasonCode = "game_over"
					result.GameOverCode = "game_over"
				}
			} else {
				// No executed moves, battery must have been 0 at start
				result.StopReasonCode = "out_of_battery"
				result.GameOverCode = "out_of_battery"
			}
		} else {
			result.StopReasonCode = "game_over"
			result.GameOverCode = "game_over"
		}
	}

	// Decision aids
	result.PossibleMoves = sess.Engine.GetPossibleMoves()
	result.LocalView3x3 = buildLocal3x3(endState)
	result.BatteryRisk = riskCode(engine.AnalyzeBatteryRisk(endState))

	// Also expose decision aids on the returned state for parity
	endState.LocalView3x3 = result.LocalView3x3
	endState.BatteryRisk = result.BatteryRisk

	// Auto-save session after bulk moves
	if err := s.sessions.Save(sessionID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after bulk moves: %v\n", sessionID, err)
	}

	return result, nil
}

// Reset resets a game session to initial state
func (s *gameServiceImpl) Reset(ctx context.Context, sessionID string) (*engine.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	s.sessions.UpdateLastAccessed(sessionID)
	state := sess.Engine.Reset()
	// Enrich state with decision aids
	state.LocalView3x3 = buildLocal3x3(state)
	state.BatteryRisk = riskCode(engine.AnalyzeBatteryRisk(state))

	// Auto-save session after reset
	if err := s.sessions.Save(sessionID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after reset: %v\n", sessionID, err)
	}

	return state, nil
}

// GetGameState retrieves the current game state
func (s *gameServiceImpl) GetGameState(ctx context.Context, sessionID string) (*engine.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	s.sessions.UpdateLastAccessed(sessionID)
	state := sess.Engine.GetState()
	// Enrich state with decision aids
	state.LocalView3x3 = buildLocal3x3(state)
	state.BatteryRisk = riskCode(engine.AnalyzeBatteryRisk(state))
	return state, nil
}

// GetMoveHistory returns paginated move history
func (s *gameServiceImpl) GetMoveHistory(ctx context.Context, sessionID string, opts HistoryOptions) (*HistoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	history := sess.Engine.GetMoveHistory()
	total := len(history)

	// Apply defaults
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	// Calculate pagination
	totalPages := (total + opts.Limit - 1) / opts.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	start := (opts.Page - 1) * opts.Limit
	end := start + opts.Limit
	if end > total {
		end = total
	}

	// Get the slice of moves
	var moves []engine.MoveHistoryEntry
	if opts.Order == "desc" {
		// Reverse order (most recent first)
		for i := total - 1 - start; i >= 0 && i >= total-end; i-- {
			moves = append(moves, history[i])
		}
	} else {
		// Normal chronological order
		if start < total {
			moves = history[start:end]
		}
	}

	// Ensure moves is not nil
	if moves == nil {
		moves = []engine.MoveHistoryEntry{}
	}

	return &HistoryResponse{
		Moves:       moves,
		TotalMoves:  total,
		Page:        opts.Page,
		PageSize:    opts.Limit,
		TotalPages:  totalPages,
		HasNext:     opts.Page < totalPages,
		HasPrevious: opts.Page > 1,
	}, nil
}

// ListConfigs returns available game configurations
func (s *gameServiceImpl) ListConfigs(ctx context.Context) ([]*ConfigInfo, error) {
	return s.configs.ListConfigs()
}

// LoadConfig loads a specific game configuration
func (s *gameServiceImpl) LoadConfig(ctx context.Context, configName string) (*engine.GameConfig, error) {
	return s.configs.LoadConfig(configName)
}

// SaveConfig saves a game configuration to disk
func (s *gameServiceImpl) SaveConfig(ctx context.Context, configName string, config *engine.GameConfig) error {
	return s.configs.SaveConfig(configName, config)
}

// extractMoveEvents generates events from a move
func (s *gameServiceImpl) extractMoveEvents(sess *Session, prevPos, newPos engine.Position, direction string) []GameEvent {
	events := []GameEvent{}
	state := sess.Engine.GetState()

	// Basic move event
	events = append(events, GameEvent{
		Type:      "move",
		Message:   fmt.Sprintf("Moved %s to (%d,%d)", direction, newPos.X, newPos.Y),
		Timestamp: time.Now(),
		Position:  newPos,
	})

	// Check if position actually changed (might be blocked)
	if prevPos.X == newPos.X && prevPos.Y == newPos.Y {
		return events // Move was blocked, no additional events
	}

	// Check for special cell events
	if newPos.Y >= 0 && newPos.Y < len(state.Grid) &&
		newPos.X >= 0 && newPos.X < len(state.Grid[0]) {
		cell := state.Grid[newPos.Y][newPos.X]

		switch cell.Type {
		case engine.Home, engine.Supercharger:
			events = append(events, GameEvent{
				Type:      "charge",
				Message:   fmt.Sprintf("Battery charged to %d/%d", state.Battery, state.MaxBattery),
				Timestamp: time.Now(),
				Position:  newPos,
			})
		case engine.Park:
			if cell.Visited {
				events = append(events, GameEvent{
					Type:      "park_visited",
					Message:   fmt.Sprintf("Park %s visited! Score: %d", cell.ID, state.Score),
					Timestamp: time.Now(),
					Position:  newPos,
				})
			}
		}
	}

	// Check for game over events
	if state.GameOver {
		if state.Victory {
			events = append(events, GameEvent{
				Type:      "victory",
				Message:   "Victory! All parks visited!",
				Timestamp: time.Now(),
			})
		} else {
			events = append(events, GameEvent{
				Type:      "game_over",
				Message:   state.Message,
				Timestamp: time.Now(),
			})
		}
	}

	return events
}

// Helpers for BulkMoveResult enrichment
func mapCellToCharAndType(cell engine.Cell) (string, string) {
	switch cell.Type {
	case engine.Road:
		return "R", "road"
	case engine.Home:
		return "H", "home"
	case engine.Park:
		if cell.Visited {
			return "✓", "park_visited"
		}
		return "P", "park"
	case engine.Supercharger:
		return "S", "supercharger"
	case engine.Water:
		return "W", "water"
	case engine.Building:
		return "B", "building"
	default:
		return ".", "unknown"
	}
}

func buildLocal3x3(state *engine.GameState) []string {
	if state == nil {
		return nil
	}
	px, py := state.PlayerPos.X, state.PlayerPos.Y
	lines := make([]string, 0, 3)
	for dy := -1; dy <= 1; dy++ {
		var row strings.Builder
		for dx := -1; dx <= 1; dx++ {
			x, y := px+dx, py+dy
			if dx == 0 && dy == 0 {
				row.WriteString("T")
				continue
			}
			// out of bounds → treat as building wall
			if y < 0 || y >= len(state.Grid) || x < 0 || x >= len(state.Grid[0]) {
				row.WriteString("B")
				continue
			}
			ch, _ := mapCellToCharAndType(state.Grid[y][x])
			row.WriteString(ch)
		}
		lines = append(lines, row.String())
	}
	return lines
}

func riskCode(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "critical"):
		return "CRITICAL"
	case strings.Contains(t, "danger"):
		return "DANGER"
	case strings.Contains(t, "caution"):
		return "CAUTION"
	case strings.Contains(t, "low"):
		return "LOW"
	case strings.Contains(t, "warning"):
		return "WARNING"
	case strings.Contains(t, "safe"):
		return "SAFE"
	default:
		return "UNKNOWN"
	}
}
