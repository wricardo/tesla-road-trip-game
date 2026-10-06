package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

// gameServiceImpl implements the GameService interface
type gameServiceImpl struct {
	sessions SessionManager
	configs  ConfigManager
	mu       sync.RWMutex
}

// getMapID returns the map_id for a given map name, used for consistent API responses
func (s *gameServiceImpl) getMapID(mapName string) string {
	availableMaps, err := s.configs.ListConfigs()
	if err == nil {
		for _, m := range availableMaps {
			if m.Name == mapName {
				return m.MapID
			}
		}
	}
	// Fallback: return as-is or "default"
	if mapName == "" {
		return "default"
	}
	return mapName
}

// mapIDForDisplayName returns the map ID whose display name matches name
// (case-insensitive), or "" if none does.
func (s *gameServiceImpl) mapIDForDisplayName(name string) string {
	availableMaps, err := s.configs.ListConfigs()
	if err != nil {
		return ""
	}
	for _, m := range availableMaps {
		if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(name)) {
			return m.MapID
		}
	}
	return ""
}

// NewGameService creates a new game service instance
func NewGameService(sessions SessionManager, configs ConfigManager) GameService {
	return &gameServiceImpl{
		sessions: sessions,
		configs:  configs,
	}
}

// CreateSession creates a new game session
func (s *gameServiceImpl) CreateSession(ctx context.Context, mapName string, opts ...CreateSessionOptions) (*SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load configuration
	var config *engine.GameConfig
	var err error
	if mapName != "" {
		config, err = s.configs.LoadConfig(mapName)
		if err != nil {
			// Accept a display name ("Classic Layout") as well as the map ID ("classic").
			if id := s.mapIDForDisplayName(mapName); id != "" {
				mapName = id
				config, err = s.configs.LoadConfig(id)
			}
		}
		if err != nil {
			// Provide helpful error message with available options
			if strings.Contains(err.Error(), "configuration not found") || strings.Contains(err.Error(), "invalid map name") {
				availableMaps, listErr := s.configs.ListConfigs()
				if listErr == nil && len(availableMaps) > 0 {
					var mapIDs []string
					for _, m := range availableMaps {
						mapIDs = append(mapIDs, m.MapID)
					}
					return nil, fmt.Errorf("map '%s' not found (use a mapId or display name from the maps query). Available map IDs: %v", mapName, mapIDs)
				}
				return nil, fmt.Errorf("map '%s' not found; list maps with the maps query", mapName)
			}
			return nil, fmt.Errorf("failed to load map %s: %w", mapName, err)
		}
	} else {
		config = s.configs.GetDefault()
	}

	var createOpts CreateSessionOptions
	if len(opts) > 0 {
		createOpts = opts[0]
	}
	if err := validateCreateSessionOptions(createOpts); err != nil {
		return nil, err
	}
	// Fog needs a password for the full grid. If the caller gave none, generate one and
	// hand it back on this response only.
	generatedPassword := ""
	if createOpts.FogEnabled && strings.TrimSpace(createOpts.GridPassword) == "" {
		pw, err := generateGridPassword()
		if err != nil {
			return nil, fmt.Errorf("failed to generate grid password: %w", err)
		}
		createOpts.GridPassword = pw
		generatedPassword = pw
	}
	if createOpts.FogRadius <= 0 {
		createOpts.FogRadius = 1
	}

	// Let session manager generate a proper 4-character ID
	session, err := s.sessions.Create("", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	// Attach fog settings to session metadata.
	session.FogEnabled = createOpts.FogEnabled
	session.FogRadius = createOpts.FogRadius
	session.GridPassword = createOpts.GridPassword
	if createOpts.MoveDelayMs != nil {
		session.MoveDelayMs = *createOpts.MoveDelayMs
	} else if session.MoveDelayMs == 0 {
		session.MoveDelayMs = DefaultMoveDelayMs
	}
	applySessionVisibilityMeta(session, session.Engine.GetState())
	if err := s.sessions.Save(session.ID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after creation: %v\n", session.ID, err)
	}

	// Determine the map identifier to return - prefer the input mapName if provided,
	// otherwise look up the map_id by display name
	mapID := mapName
	if mapID == "" {
		mapID = s.getMapID(config.Name)
	}

	return &SessionInfo{
		ID:                    session.ID,
		MapName:               mapID,
		CreatedAt:             session.CreatedAt,
		LastActionAt:          session.LastActionAt,
		FogEnabled:            session.FogEnabled,
		GameState:             sessionSnapshot(session),
		GameMap:               session.Config,
		GeneratedGridPassword: generatedPassword,
	}, nil
}

func validateCreateSessionOptions(opts CreateSessionOptions) error {
	if opts.MoveDelayMs != nil && *opts.MoveDelayMs < 0 {
		return fmt.Errorf("move delay must be >= 0")
	}
	if !opts.FogEnabled {
		// Fog settings without fog would silently create a fully visible session.
		if opts.GridPassword != "" || opts.FogRadius > 1 {
			return fmt.Errorf("fogRadius/gridPassword only apply to fog sessions: also pass fogEnabled: true")
		}
		return nil
	}
	if opts.FogRadius < 1 {
		return fmt.Errorf("fog radius must be >= 1 when fog mode is enabled")
	}
	return nil
}

// gridPasswordAlphabet has 32 characters (no 0/o/1/l look-alikes), so a random byte maps to it without bias.
const gridPasswordAlphabet = "abcdefghijkmnpqrstuvwxyz23456789"

func generateGridPassword() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = gridPasswordAlphabet[int(b)%len(gridPasswordAlphabet)]
	}
	return string(buf), nil
}

func applySessionVisibilityMeta(session *Session, state *engine.GameState) {
	if session == nil || state == nil {
		return
	}
	radius := session.FogRadius
	if radius <= 0 {
		radius = 1
	}
	state.FogEnabled = session.FogEnabled
	state.FogRadius = radius
	state.GridPassword = session.GridPassword
	delay := session.MoveDelayMs
	if delay < 0 {
		delay = 0
	}
	state.MoveDelayMs = delay
}

// sessionSnapshot returns a private deep copy of the session's current state with
// visibility metadata applied. Service methods return snapshots instead of the
// live engine state so callers can read and enrich them after the service lock
// is released without racing concurrent moves.
func sessionSnapshot(session *Session) *engine.GameState {
	state := session.Engine.GetState().Clone()
	applySessionVisibilityMeta(session, state)
	return state
}

// GetSession retrieves session information
func (s *gameServiceImpl) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	state := sessionSnapshot(session)

	return &SessionInfo{
		ID:           session.ID,
		DisplayName:  session.DisplayName,
		MapName:      s.getMapID(session.Config.Name),
		CreatedAt:    session.CreatedAt,
		LastActionAt: session.LastActionAt,
		FogEnabled:   session.FogEnabled,
		GameState:    state,
		GameMap:      session.Config,
	}, nil
}

// ListSessions returns all active sessions
func (s *gameServiceImpl) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := s.sessions.List()
	result := make([]*SessionInfo, 0, len(sessions))

	for _, sess := range sessions {
		state := sessionSnapshot(sess)
		result = append(result, &SessionInfo{
			ID:           sess.ID,
			DisplayName:  sess.DisplayName,
			MapName:      s.getMapID(sess.Config.Name),
			CreatedAt:    sess.CreatedAt,
			LastActionAt: sess.LastActionAt,
			FogEnabled:   sess.FogEnabled,
			GameState:    state,
			GameMap:      sess.Config,
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

// UpdateSessionDisplayName sets the display name for a session.
func (s *gameServiceImpl) UpdateSessionDisplayName(ctx context.Context, sessionID, displayName string) (*SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sessions.UpdateDisplayName(sessionID, displayName); err != nil {
		return nil, fmt.Errorf("update display name: %w", err)
	}

	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found after update: %w", err)
	}
	state := sessionSnapshot(session)

	return &SessionInfo{
		ID:           session.ID,
		DisplayName:  session.DisplayName,
		MapName:      s.getMapID(session.Config.Name),
		CreatedAt:    session.CreatedAt,
		LastActionAt: session.LastActionAt,
		FogEnabled:   session.FogEnabled,
		GameState:    state,
		GameMap:      session.Config,
	}, nil
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

	// Update last action time
	s.sessions.UpdateLastAction(sessionID)

	// Handle reset if requested
	if reset {
		sess.Engine.Reset()
	}

	// A finished game rejects moves outright; don't echo the stale end-of-game
	// message, which reads like this move hit something.
	if sess.Engine.IsGameOver() {
		state := decoratedSnapshot(sess)
		return &MoveResult{Success: false, GameState: state, Message: alreadyOverMessage(state)}, nil
	}

	// Execute move
	prevPos := sess.Engine.GetPlayerPosition()
	prevBattery := sess.Engine.GetState().Battery
	success := sess.Engine.Move(direction)
	state := decoratedSnapshot(sess)

	result := &MoveResult{
		Success:   success,
		GameState: state,
		Message:   state.Message,
	}
	if success {
		step := executedStep(1, direction, prevPos, prevBattery, state)
		result.Step = &step
	} else {
		result.AttemptedTo, _ = rejectedMove(state, prevPos, direction)
	}

	// Auto-save session after move
	if err := s.sessions.Save(sessionID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after move: %v\n", sessionID, err)
	}

	return result, nil
}

// BulkMove executes up to engine.MaxBulkMoves moves in order. It stops at the first
// rejected move or when the game ends. opts.StepDelay paces the run for spectators;
// the service lock is released between steps so other sessions stay responsive.
func (s *gameServiceImpl) BulkMove(ctx context.Context, sessionID string, moves []string, reset bool, opts BulkMoveOptions) (*BulkMoveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	s.sessions.UpdateLastAction(sessionID)

	if reset {
		sess.Engine.Reset()
	}
	result := &BulkMoveResult{RequestedMoves: len(moves), Success: true}
	if len(moves) > engine.MaxBulkMoves {
		result.Truncated = true
		result.Limit = engine.MaxBulkMoves
		moves = moves[:engine.MaxBulkMoves]
	}
	result.TotalMoves = len(moves)

	start := sess.Engine.GetState()
	result.StartPos = start.PlayerPos
	result.StartBattery = start.Battery
	startScore := start.Score

	var cancelErr error
	if sess.Engine.IsGameOver() {
		result.Success = false
		result.StopReasonCode = "already_over"
		if len(moves) > 0 {
			result.StoppedOnMove = 1
		}
	} else {
		for i, move := range moves {
			if i > 0 && opts.StepDelay > 0 {
				s.mu.Unlock()
				cancelErr = sleepCtx(ctx, opts.StepDelay)
				s.mu.Lock()
				if cancelErr != nil {
					break
				}
			}
			done := bulkStep(sess, i, move, result)
			if opts.OnStep != nil {
				snap := decoratedSnapshot(sess)
				s.mu.Unlock()
				opts.OnStep(snap)
				s.mu.Lock()
			}
			if done {
				break
			}
		}
	}

	end := decoratedSnapshot(sess)
	result.GameState = end
	result.EndPos = end.PlayerPos
	result.EndBattery = end.Battery
	result.ScoreDelta = end.Score - startScore
	result.GameOver = end.GameOver
	result.GameOverCode = gameOverCode(end)
	result.Message = end.Message
	switch {
	case result.StopReasonCode == "already_over":
		result.Message = alreadyOverMessage(end)
		result.StoppedReason = result.Message
	case result.StopReasonCode == "" && end.GameOver:
		// The last executed move ended the game: victory, stranded, or another end.
		result.StopReasonCode = result.GameOverCode
		result.StoppedReason = end.Message
	}
	result.PossibleMoves = sess.Engine.GetPossibleMoves()
	result.LocalView3x3 = end.LocalView3x3
	result.BatteryRisk = end.BatteryRisk

	if err := s.sessions.Save(sessionID); err != nil {
		fmt.Printf("Warning: Failed to persist session %s after bulk moves: %v\n", sessionID, err)
	}
	if cancelErr != nil {
		return nil, cancelErr
	}
	return result, nil
}

// bulkStep runs moves[i] and records it on result. It reports whether the run must stop.
func bulkStep(sess *Session, i int, move string, result *BulkMoveResult) bool {
	if sess.Engine.IsGameOver() {
		// Another request ended the game while this run waited between steps.
		result.Success = false
		result.StoppedOnMove = i + 1
		result.StopReasonCode = "game_over"
		result.StoppedReason = fmt.Sprintf("game ended before move %d", i+1)
		return true
	}
	prevPos := sess.Engine.GetPlayerPosition()
	prevBattery := sess.Engine.GetState().Battery
	if !sess.Engine.Move(move) {
		result.Success = false
		result.StoppedOnMove = i + 1
		result.StoppedReason = fmt.Sprintf("move %d blocked: %s", i+1, move)
		result.AttemptedTo, result.StopReasonCode = rejectedMove(sess.Engine.GetState(), prevPos, move)
		return true
	}
	st := sess.Engine.GetState()
	result.MovesExecuted++
	result.Steps = append(result.Steps, executedStep(i+1, move, prevPos, prevBattery, st))
	return st.GameOver
}

// executedStep describes a move that succeeded; st is the state after the move.
func executedStep(idx int, dir string, from engine.Position, batteryBefore int, st *engine.GameState) StepInfo {
	step := StepInfo{
		Idx:           idx,
		Dir:           dir,
		From:          from,
		To:            st.PlayerPos,
		BatteryBefore: batteryBefore,
		BatteryAfter:  st.Battery,
		Success:       true,
		Victory:       st.Victory,
	}
	x, y := st.PlayerPos.X, st.PlayerPos.Y
	if y >= 0 && y < len(st.Grid) && x >= 0 && x < len(st.Grid[y]) {
		cell := st.Grid[y][x]
		step.TileChar, step.TileType = mapCellToCharAndType(cell)
		step.Charged = cell.Type == engine.Home || cell.Type == engine.Supercharger
		step.Park = cell.Type == engine.Park
	}
	return step
}

// rejectedMove describes the cell a rejected move aimed at and classifies the
// rejection; st is the state after the engine refused the move.
func rejectedMove(st *engine.GameState, from engine.Position, dir string) (*AttemptInfo, string) {
	x, y := from.X, from.Y
	switch strings.ToLower(dir) {
	case "up":
		y--
	case "down":
		y++
	case "left":
		x--
	case "right":
		x++
	}
	attempt := &AttemptInfo{X: x, Y: y, TileChar: "B", TileType: "boundary"}
	inside := y >= 0 && y < len(st.Grid) && x >= 0 && x < len(st.Grid[y])
	var cellType engine.CellType
	if inside {
		cell := st.Grid[y][x]
		cellType = cell.Type
		attempt.TileChar, attempt.TileType = mapCellToCharAndType(cell)
		attempt.Passable = cell.Type != engine.Water && cell.Type != engine.Building
	}
	switch {
	case !st.GameOver:
		// Rejected without ending the game: a one-way road.
		return attempt, "blocked_direction"
	case !inside:
		return attempt, "blocked_boundary"
	case cellType == engine.Water:
		return attempt, "blocked_water"
	case cellType == engine.Building:
		return attempt, "blocked_building"
	default:
		return attempt, "out_of_battery"
	}
}

// gameOverCode classifies how a finished game ended; empty while it is running.
func gameOverCode(st *engine.GameState) string {
	switch {
	case !st.GameOver:
		return ""
	case st.Victory:
		return "victory"
	case strings.HasPrefix(st.Message, engine.DefaultMessages.HitWall):
		return "crashed"
	case st.Message == engine.DefaultMessages.Stranded:
		return "stranded"
	case st.Message == engine.DefaultMessages.OutOfBattery:
		return "out_of_battery"
	default:
		return "game_over"
	}
}

// decoratedSnapshot is sessionSnapshot plus the per-state decision aids.
func decoratedSnapshot(sess *Session) *engine.GameState {
	state := sessionSnapshot(sess)
	state.LocalView3x3 = buildLocal3x3(state)
	state.BatteryRisk = riskCode(engine.AnalyzeBatteryRisk(state))
	return state
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// alreadyOverMessage explains why a move was refused on a finished game.
func alreadyOverMessage(state *engine.GameState) string {
	why := "you won"
	if !state.Victory {
		why = "it ended: " + state.Message
	}
	return "Game is already over (" + why + "). No move was made — call reset (or pass reset: true) to play again."
}

// Reset resets a game session to initial state
func (s *gameServiceImpl) Reset(ctx context.Context, sessionID string) (*engine.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	s.sessions.UpdateLastAction(sessionID)
	sess.Engine.Reset()
	state := decoratedSnapshot(sess)

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

	state := decoratedSnapshot(sess)
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

// ListMaps returns available game maps
func (s *gameServiceImpl) ListMaps(ctx context.Context) ([]*MapInfo, error) {
	return s.configs.ListConfigs()
}

// LoadMap loads a specific game map configuration
func (s *gameServiceImpl) LoadMap(ctx context.Context, mapName string) (*engine.GameConfig, error) {
	return s.configs.LoadConfig(mapName)
}

// SaveMap saves a game map configuration to disk
func (s *gameServiceImpl) SaveMap(ctx context.Context, mapName string, config *engine.GameConfig) error {
	return s.configs.SaveConfig(mapName, config)
}

// DeleteMap removes a map configuration from disk
func (s *gameServiceImpl) DeleteMap(ctx context.Context, mapName string) error {
	return s.configs.DeleteConfig(mapName)
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
