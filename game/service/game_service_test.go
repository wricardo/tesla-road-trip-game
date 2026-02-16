package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/wricardo/mcp-training/statefullgame/game/engine"
	"github.com/wricardo/mcp-training/statefullgame/game/service"
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
		ID:             id,
		Engine:         eng,
		Config:         config,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
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

func (m *MockSessionManager) UpdateLastAccessed(id string) error {
	if session, exists := m.sessions[id]; exists {
		session.LastAccessedAt = time.Now()
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

func (m *MockConfigManager) ListConfigs() ([]*service.ConfigInfo, error) {
	result := make([]*service.ConfigInfo, 0, len(m.configs))
	for name, config := range m.configs {
		result = append(result, &service.ConfigInfo{
			Filename:    name + ".json",
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
