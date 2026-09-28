package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
)

// MockSessionManager implements service.SessionManager for testing
type MockSessionManager struct {
	sessions map[string]*service.Session
}

func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: make(map[string]*service.Session),
	}
}

func (m *MockSessionManager) Create(id string, config *engine.GameConfig) (*service.Session, error) {
	// Generate ID if empty (mimics real session manager behavior)
	if id == "" {
		id = fmt.Sprintf("test_%d", len(m.sessions)+1)
	}

	if _, exists := m.sessions[id]; exists {
		return nil, errors.New("session already exists")
	}

	eng, err := engine.NewEngine(config)
	if err != nil {
		return nil, err
	}

	session := &service.Session{
		ID:           id,
		Engine:       eng,
		Config:       config,
		CreatedAt:    time.Now(),
		LastActionAt: time.Now(),
	}

	m.sessions[id] = session
	return session, nil
}

func (m *MockSessionManager) Get(id string) (*service.Session, error) {
	session, exists := m.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (m *MockSessionManager) GetOrCreate(id string, config *engine.GameConfig) (*service.Session, error) {
	if session, exists := m.sessions[id]; exists {
		return session, nil
	}
	return m.Create(id, config)
}

func (m *MockSessionManager) List() []*service.Session {
	result := make([]*service.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session)
	}
	return result
}

func (m *MockSessionManager) Delete(id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionManager) UpdateLastAction(id string) error {
	if session, exists := m.sessions[id]; exists {
		session.LastActionAt = time.Now()
		return nil
	}
	return errors.New("session not found")
}

func (m *MockSessionManager) Save(id string) error {
	if _, exists := m.sessions[id]; !exists {
		return errors.New("session not found")
	}
	// Mock save - in real implementation this would persist to disk
	return nil
}

func (m *MockSessionManager) UpdateDisplayName(id, displayName string) error {
	session, exists := m.sessions[id]
	if !exists {
		return errors.New("session not found")
	}
	session.DisplayName = displayName
	return nil
}

// MockConfigManager implements service.ConfigManager for testing
type MockConfigManager struct {
	configs map[string]*engine.GameConfig
}

func NewMockConfigManager() *MockConfigManager {
	// Create a default test config
	defaultConfig := &engine.GameConfig{
		Name:            "test",
		Description:     "Test configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 10,
		Layout: []string{
			"RRPRR",
			"RWRWR",
			"RRRHR",
			"RWRWR",
			"RRPRR",
		},
		Legend: map[string]string{
			"R": "road",
			"H": "home",
			"P": "park",
			"S": "supercharger",
			"W": "water",
			"B": "building",
		},
	}

	return &MockConfigManager{
		configs: map[string]*engine.GameConfig{
			"test":    defaultConfig,
			"default": defaultConfig,
		},
	}
}

func (m *MockConfigManager) LoadConfig(name string) (*engine.GameConfig, error) {
	config, exists := m.configs[name]
	if !exists {
		return nil, errors.New("config not found")
	}
	return config, nil
}

func (m *MockConfigManager) ListConfigs() ([]*service.MapInfo, error) {
	result := make([]*service.MapInfo, 0, len(m.configs))
	for name, config := range m.configs {
		result = append(result, &service.MapInfo{
			Filename:    name + ".json",
			MapID:       name,
			Name:        config.Name,
			Description: config.Description,
			GridSize:    config.GridSize,
			MaxBattery:  config.MaxBattery,
		})
	}
	return result, nil
}

func (m *MockConfigManager) GetDefault() *engine.GameConfig {
	return m.configs["default"]
}

func (m *MockConfigManager) SaveConfig(name string, config *engine.GameConfig) error {
	m.configs[name] = config
	return nil
}

func (m *MockConfigManager) DeleteConfig(name string) error {
	delete(m.configs, name)
	return nil
}

// Test cases
func TestGameService_CreateSession(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	tests := []struct {
		name       string
		configName string
		wantErr    bool
	}{
		{
			name:       "create with default config",
			configName: "",
			wantErr:    false,
		},
		{
			name:       "create with specific config",
			configName: "test",
			wantErr:    false,
		},
		{
			name:       "create with invalid config",
			configName: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := svc.CreateSession(ctx, tt.configName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && session == nil {
				t.Error("CreateSession() returned nil session")
			}
		})
	}
}

func TestGameService_CreateSession_FogOptions(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	tests := []struct {
		name    string
		opts    service.CreateSessionOptions
		wantErr bool
	}{
		{
			name: "fog disabled accepts defaults",
			opts: service.CreateSessionOptions{},
		},
		{
			name: "fog enabled with valid radius and password",
			opts: service.CreateSessionOptions{FogEnabled: true, FogRadius: 2, GridPassword: "secret"},
		},
		{
			name:    "fog enabled requires radius",
			opts:    service.CreateSessionOptions{FogEnabled: true, FogRadius: 0, GridPassword: "secret"},
			wantErr: true,
		},
		{
			name:    "fog enabled requires password",
			opts:    service.CreateSessionOptions{FogEnabled: true, FogRadius: 2},
			wantErr: true,
		},
		{
			name: "move delay can be disabled",
			opts: service.CreateSessionOptions{MoveDelayMs: intPtr(0)},
		},
		{
			name:    "move delay rejects negative values",
			opts:    service.CreateSessionOptions{MoveDelayMs: intPtr(-1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := svc.CreateSession(ctx, "test", tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if session == nil || session.GameState == nil {
				t.Fatalf("expected session/game state")
			}
			if tt.opts.FogEnabled {
				if !session.GameState.FogEnabled {
					t.Fatalf("expected fog enabled on game state")
				}
				if session.GameState.FogRadius != tt.opts.FogRadius {
					t.Fatalf("expected fog radius %d, got %d", tt.opts.FogRadius, session.GameState.FogRadius)
				}
			}
			expectedDelay := service.DefaultMoveDelayMs
			if tt.opts.MoveDelayMs != nil {
				expectedDelay = *tt.opts.MoveDelayMs
			}
			if session.GameState.MoveDelayMs != expectedDelay {
				t.Fatalf("expected move delay %d, got %d", expectedDelay, session.GameState.MoveDelayMs)
			}
		})
	}
}

func intPtr(v int) *int { return &v }

func TestGameService_Move(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	// Create a session first
	sessionInfo, err := svc.CreateSession(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		direction string
		reset     bool
		wantErr   bool
	}{
		{
			name:      "valid move up",
			sessionID: sessionInfo.ID,
			direction: "up",
			reset:     false,
			wantErr:   false,
		},
		{
			name:      "valid move with reset",
			sessionID: sessionInfo.ID,
			direction: "right",
			reset:     true,
			wantErr:   false,
		},
		{
			name:      "invalid session",
			sessionID: "nonexistent",
			direction: "up",
			reset:     false,
			wantErr:   true,
		},
		{
			name:      "invalid direction",
			sessionID: sessionInfo.ID,
			direction: "diagonal",
			reset:     false,
			wantErr:   false, // Won't error but success will be false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Move(ctx, tt.sessionID, tt.direction, tt.reset)
			if (err != nil) != tt.wantErr {
				t.Errorf("Move() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Move() returned nil result")
			}
		})
	}

	// Additional checks: StepInfo on success and AttemptInfo on failure
	// Reset to ensure consistent start
	_, _ = svc.Reset(ctx, sessionInfo.ID)

	// Successful move from Home (3,2) to left (2,2) which is road
	res1, err := svc.Move(ctx, sessionInfo.ID, "left", false)
	if err != nil {
		t.Fatalf("Move left failed unexpectedly: %v", err)
	}
	if res1.Step == nil || !res1.Success {
		t.Errorf("Expected success with StepInfo, got success=%v step=%v", res1.Success, res1.Step)
	} else {
		if res1.Step.Dir != "left" || res1.Step.TileChar == "" {
			t.Errorf("Invalid StepInfo: %+v", res1.Step)
		}
	}

	// Failing move: from new position (2,2) attempt up to (2,1) which is R (passable) — move to (2,1) first
	_, _ = svc.Move(ctx, sessionInfo.ID, "up", false)
	// Now at (2,1), attempt right to (3,1) which is W (water) and should fail
	res2, err := svc.Move(ctx, sessionInfo.ID, "right", false)
	if err != nil {
		t.Fatalf("Move right failed with error: %v", err)
	}
	if res2.Success {
		t.Errorf("Expected failure moving into water, got success")
	}
	if res2.AttemptedTo == nil || res2.AttemptedTo.TileChar != "W" || res2.AttemptedTo.Passable {
		t.Errorf("Expected AttemptedTo with water impassable, got %+v", res2.AttemptedTo)
	}
}

func TestGameService_BulkMove(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	// Create a session
	sessionInfo, err := svc.CreateSession(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		moves     []string
		reset     bool
		wantErr   bool
	}{
		{
			name:      "valid bulk moves",
			sessionID: sessionInfo.ID,
			moves:     []string{"up", "right", "down", "left"},
			reset:     false,
			wantErr:   false,
		},
		{
			name:      "bulk moves with reset",
			sessionID: sessionInfo.ID,
			moves:     []string{"up", "up"},
			reset:     true,
			wantErr:   false,
		},
		{
			name:      "empty moves",
			sessionID: sessionInfo.ID,
			moves:     []string{},
			reset:     false,
			wantErr:   false,
		},
		{
			name:      "invalid session",
			sessionID: "nonexistent",
			moves:     []string{"up"},
			reset:     false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.BulkMove(ctx, tt.sessionID, tt.moves, tt.reset)
			if (err != nil) != tt.wantErr {
				t.Errorf("BulkMove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("BulkMove() returned nil result")
			}
			if !tt.wantErr && result != nil {
				if result.TotalMoves != len(tt.moves) {
					t.Errorf("BulkMove() TotalMoves = %v, want %v", result.TotalMoves, len(tt.moves))
				}
			}
		})
	}

	// Additional bulk diagnostics: steps, stop_reason_code, attempted_to
	// Reset to start from Home (3,2)
	_, _ = svc.Reset(ctx, sessionInfo.ID)
	// Sequence: left (ok), right (ok, back to home), up (blocked by water)
	res3, err := svc.BulkMove(ctx, sessionInfo.ID, []string{"left", "right", "up"}, false)
	if err != nil {
		t.Fatalf("BulkMove diagnostics failed with error: %v", err)
	}
	if res3.MovesExecuted != 2 {
		t.Errorf("Expected 2 executed moves, got %d", res3.MovesExecuted)
	}
	if len(res3.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(res3.Steps))
	}
	if res3.StopReasonCode == "" || res3.AttemptedTo == nil || res3.AttemptedTo.TileChar != "W" {
		t.Errorf("Expected stop_reason_code and attempted_to=W, got code=%s attempted=%+v", res3.StopReasonCode, res3.AttemptedTo)
	}
}

func TestGameService_GetMoveHistory(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	// Create a session and make some moves
	sessionInfo, err := svc.CreateSession(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Make some moves to generate history
	moves := []string{"up", "right", "down", "left"}
	_, err = svc.BulkMove(ctx, sessionInfo.ID, moves, false)
	if err != nil {
		t.Fatalf("Failed to make moves: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		opts      service.HistoryOptions
		wantErr   bool
	}{
		{
			name:      "default options",
			sessionID: sessionInfo.ID,
			opts:      service.HistoryOptions{},
			wantErr:   false,
		},
		{
			name:      "with pagination",
			sessionID: sessionInfo.ID,
			opts: service.HistoryOptions{
				Page:  1,
				Limit: 2,
				Order: "asc",
			},
			wantErr: false,
		},
		{
			name:      "descending order",
			sessionID: sessionInfo.ID,
			opts: service.HistoryOptions{
				Page:  1,
				Limit: 10,
				Order: "desc",
			},
			wantErr: false,
		},
		{
			name:      "invalid session",
			sessionID: "nonexistent",
			opts:      service.HistoryOptions{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetMoveHistory(ctx, tt.sessionID, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMoveHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("GetMoveHistory() returned nil result")
			}
			if !tt.wantErr && result != nil {
				if result.Moves == nil {
					t.Error("GetMoveHistory() returned nil moves slice")
				}
			}
		})
	}
}

func TestGameService_ListSessions(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_, err := svc.CreateSession(ctx, "test")
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// List sessions
	sessionList, err := svc.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessionList) != 3 {
		t.Errorf("ListSessions() returned %d sessions, want 3", len(sessionList))
	}
}

func TestGameService_Reset(t *testing.T) {
	ctx := context.Background()
	sessions := NewMockSessionManager()
	configs := NewMockConfigManager()
	svc := service.NewGameService(sessions, configs)

	// Create a session
	sessionInfo, err := svc.CreateSession(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Make some moves
	_, err = svc.Move(ctx, sessionInfo.ID, "up", false)
	if err != nil {
		t.Fatalf("Failed to move: %v", err)
	}

	// Reset the game
	state, err := svc.Reset(ctx, sessionInfo.ID)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if state == nil {
		t.Error("Reset() returned nil state")
	}

	// Verify player is back at starting position
	// (This would depend on your specific game logic)
}

// TestGameService_ReturnedStateIsSnapshot guards against handing out the live
// engine state: reads run under a read lock and callers use the result after
// the lock is released, so both must operate on private copies. Run with -race.
func TestGameService_ReturnedStateIsSnapshot(t *testing.T) {
	ctx := context.Background()
	svc := service.NewGameService(NewMockSessionManager(), NewMockConfigManager())
	info, err := svc.CreateSession(ctx, "test")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	before, err := svc.GetGameState(ctx, info.ID)
	if err != nil {
		t.Fatalf("GetGameState: %v", err)
	}
	moves := before.TotalMoves
	if _, err := svc.Move(ctx, info.ID, "up", false); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if before.TotalMoves != moves {
		t.Fatal("previously returned state changed after a later move")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			dir := []string{"up", "down", "left", "right"}[i%4]
			if _, err := svc.Move(ctx, info.ID, dir, i%50 == 0); err != nil {
				t.Errorf("Move: %v", err)
				return
			}
		}
	}()
	for i := 0; i < 200; i++ {
		st, err := svc.GetGameState(ctx, info.ID)
		if err != nil {
			t.Fatalf("GetGameState: %v", err)
		}
		_ = len(st.MoveHistory) + st.Battery // read after the service lock is released
		sess, err := svc.GetSession(ctx, info.ID)
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		_ = sess.GameState.PlayerPos
	}
	<-done
}

func TestGameService_BulkMove_WrongWayReportsBlockedDirection(t *testing.T) {
	ctx := context.Background()
	configs := NewMockConfigManager()
	configs.configs["oneway"] = &engine.GameConfig{
		Name:            "oneway",
		GridSize:        5,
		MaxBattery:      5,
		StartingBattery: 5,
		Layout:          []string{"BBBBB", "BH>PB", "BBBBB", "BBBBB", "BBBBB"},
		Legend:          map[string]string{"R": "road", "H": "home", "P": "park", "S": "supercharger", "B": "building", "W": "water"},
		CellConfigs:     map[string]engine.CellConfig{">": {Type: "road", AllowedDirections: []string{"east"}}},
	}
	svc := service.NewGameService(NewMockSessionManager(), configs)
	sess, err := svc.CreateSession(ctx, "oneway")
	if err != nil {
		t.Fatal(err)
	}
	// RIGHT onto '>' is fine; LEFT back out of it violates the one-way rule.
	res, err := svc.BulkMove(ctx, sess.ID, []string{"right", "left"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MovesExecuted != 1 || res.StopReasonCode != "blocked_direction" {
		t.Fatalf("want 1 move then blocked_direction, got %d moves, code %q (%s)", res.MovesExecuted, res.StopReasonCode, res.StoppedReason)
	}
}

func TestGameService_CreateSession_FogSettingsRequireFogEnabled(t *testing.T) {
	svc := service.NewGameService(NewMockSessionManager(), NewMockConfigManager())
	for name, opts := range map[string]service.CreateSessionOptions{
		"password without fog": {GridPassword: "c0d3"},
		"radius without fog":   {FogRadius: 2},
	} {
		if _, err := svc.CreateSession(context.Background(), "test", opts); err == nil {
			t.Errorf("%s: expected error, got a non-fog session", name)
		}
	}
}

func TestGameService_CreateSession_AcceptsDisplayName(t *testing.T) {
	configs := NewMockConfigManager()
	configs.configs["classic"] = configs.configs["test"]
	svc := service.NewGameService(NewMockSessionManager(), configs)
	sess, err := svc.CreateSession(context.Background(), strings.ToUpper(configs.configs["classic"].Name))
	if err != nil {
		t.Fatalf("display name rejected: %v", err)
	}
	if sess.GameMap != configs.configs["classic"] {
		t.Fatalf("display name resolved to the wrong map")
	}
}

func TestGameService_MovesAfterGameOverReportAlreadyOver(t *testing.T) {
	ctx := context.Background()
	configs := NewMockConfigManager()
	// Start next to the only park: one move wins and ends the game.
	configs.configs["win"] = &engine.GameConfig{
		Name: "win", GridSize: 5, MaxBattery: 5, StartingBattery: 5,
		Layout: []string{"BBBBB", "BHPBB", "BBBBB", "BBBBB", "BBBBB"},
		Legend: map[string]string{"R": "road", "H": "home", "P": "park", "S": "supercharger", "B": "building", "W": "water"},
	}
	svc := service.NewGameService(NewMockSessionManager(), configs)
	sess, err := svc.CreateSession(ctx, "win")
	if err != nil {
		t.Fatal(err)
	}
	if res, _ := svc.Move(ctx, sess.ID, "right", false); !res.GameState.Victory {
		t.Fatalf("setup: expected victory, got %q", res.Message)
	}

	mv, err := svc.Move(ctx, sess.ID, "left", false)
	if err != nil {
		t.Fatal(err)
	}
	if mv.Success || !strings.Contains(mv.Message, "already over") || mv.AttemptedTo != nil {
		t.Fatalf("move after game over: success=%v message=%q attempted=%v", mv.Success, mv.Message, mv.AttemptedTo)
	}

	bm, err := svc.BulkMove(ctx, sess.ID, []string{"left", "left"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if bm.Success || bm.StopReasonCode != "already_over" || bm.MovesExecuted != 0 || !strings.Contains(bm.Message, "reset") {
		t.Fatalf("bulkMove after game over: success=%v code=%q moves=%d message=%q", bm.Success, bm.StopReasonCode, bm.MovesExecuted, bm.Message)
	}
}
