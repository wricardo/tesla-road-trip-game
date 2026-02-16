package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/wricardo/mcp-training/statefullgame/game/engine"
	"github.com/wricardo/mcp-training/statefullgame/game/service"
	"github.com/wricardo/mcp-training/statefullgame/transport/websocket"
)

// MockGameService implements service.GameService for testing
type MockGameService struct {
	// Session Management
	CreateSessionFunc func(ctx context.Context, configName string) (*service.SessionInfo, error)
	GetSessionFunc    func(ctx context.Context, sessionID string) (*service.SessionInfo, error)
	ListSessionsFunc  func(ctx context.Context) ([]*service.SessionInfo, error)
	DeleteSessionFunc func(ctx context.Context, sessionID string) error

	// Game Operations
	MoveFunc     func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error)
	BulkMoveFunc func(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error)
	ResetFunc    func(ctx context.Context, sessionID string) (*engine.GameState, error)

	// Game State
	GetGameStateFunc   func(ctx context.Context, sessionID string) (*engine.GameState, error)
	GetMoveHistoryFunc func(ctx context.Context, sessionID string, opts service.HistoryOptions) (*service.HistoryResponse, error)

	// Configuration
	ListConfigsFunc func(ctx context.Context) ([]*service.ConfigInfo, error)
	LoadConfigFunc  func(ctx context.Context, configName string) (*engine.GameConfig, error)
}

// Session Management
func (m *MockGameService) CreateSession(ctx context.Context, configName string) (*service.SessionInfo, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, configName)
	}
	return &service.SessionInfo{
		ID:         "test-session",
		ConfigName: configName,
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MockGameService) GetSession(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx, sessionID)
	}
	return &service.SessionInfo{
		ID:         sessionID,
		ConfigName: "test-config",
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MockGameService) ListSessions(ctx context.Context) ([]*service.SessionInfo, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc(ctx)
	}
	return []*service.SessionInfo{}, nil
}

func (m *MockGameService) DeleteSession(ctx context.Context, sessionID string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, sessionID)
	}
	return nil
}

// Game Operations
func (m *MockGameService) Move(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
	if m.MoveFunc != nil {
		return m.MoveFunc(ctx, sessionID, direction, reset)
	}
	return &service.MoveResult{
		Success:   true,
		GameState: &engine.GameState{},
	}, nil
}

func (m *MockGameService) BulkMove(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error) {
	if m.BulkMoveFunc != nil {
		return m.BulkMoveFunc(ctx, sessionID, moves, reset)
	}
	return &service.BulkMoveResult{
		Success:   true,
		GameState: &engine.GameState{},
	}, nil
}

func (m *MockGameService) Reset(ctx context.Context, sessionID string) (*engine.GameState, error) {
	if m.ResetFunc != nil {
		return m.ResetFunc(ctx, sessionID)
	}
	return &engine.GameState{}, nil
}

// Game State
func (m *MockGameService) GetGameState(ctx context.Context, sessionID string) (*engine.GameState, error) {
	if m.GetGameStateFunc != nil {
		return m.GetGameStateFunc(ctx, sessionID)
	}
	return &engine.GameState{}, nil
}

func (m *MockGameService) GetMoveHistory(ctx context.Context, sessionID string, opts service.HistoryOptions) (*service.HistoryResponse, error) {
	if m.GetMoveHistoryFunc != nil {
		return m.GetMoveHistoryFunc(ctx, sessionID, opts)
	}
	return &service.HistoryResponse{
		Moves:      []engine.MoveHistoryEntry{},
		TotalMoves: 0,
		Page:       opts.Page,
		PageSize:   opts.Limit,
		TotalPages: 1,
	}, nil
}

// Configuration
func (m *MockGameService) ListConfigs(ctx context.Context) ([]*service.ConfigInfo, error) {
	if m.ListConfigsFunc != nil {
		return m.ListConfigsFunc(ctx)
	}
	return []*service.ConfigInfo{}, nil
}

func (m *MockGameService) LoadConfig(ctx context.Context, configName string) (*engine.GameConfig, error) {
	if m.LoadConfigFunc != nil {
		return m.LoadConfigFunc(ctx, configName)
	}
	return &engine.GameConfig{
		Name:        configName,
		Description: "Test config",
	}, nil
}

// Test helpers
func setupTestServer(mockService *MockGameService) *Server {
	hub := websocket.NewHub()
	go hub.Run()
	return NewServer(mockService, hub)
}

func makeRequest(method, path string, body interface{}) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
}

// Session Management Tests

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Create session with default config",
			requestBody: nil,
			setupMock: func(m *MockGameService) {
				m.CreateSessionFunc = func(ctx context.Context, configName string) (*service.SessionInfo, error) {
					return &service.SessionInfo{
						ID:             "sess-123",
						ConfigName:     "default",
						CreatedAt:      time.Now(),
						LastAccessedAt: time.Now(),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.SessionInfo
				parseResponse(t, w, &resp)
				if resp.ID != "sess-123" {
					t.Errorf("Expected session ID sess-123, got %s", resp.ID)
				}
			},
		},
		{
			name:        "Create session with specific config",
			requestBody: map[string]string{"config_name": "easy"},
			setupMock: func(m *MockGameService) {
				m.CreateSessionFunc = func(ctx context.Context, configName string) (*service.SessionInfo, error) {
					if configName != "easy" {
						t.Errorf("Expected config name 'easy', got %s", configName)
					}
					return &service.SessionInfo{
						ID:         "sess-456",
						ConfigName: configName,
						CreatedAt:  time.Now(),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.SessionInfo
				parseResponse(t, w, &resp)
				if resp.ConfigName != "easy" {
					t.Errorf("Expected config name 'easy', got %s", resp.ConfigName)
				}
			},
		},
		{
			name:        "Handle service error",
			requestBody: nil,
			setupMock: func(m *MockGameService) {
				m.CreateSessionFunc = func(ctx context.Context, configName string) (*service.SessionInfo, error) {
					return nil, fmt.Errorf("service error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "service error" {
					t.Errorf("Expected error message 'service error', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("POST", "/api/sessions", tt.requestBody)

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestListSessions(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "List multiple sessions",
			setupMock: func(m *MockGameService) {
				m.ListSessionsFunc = func(ctx context.Context) ([]*service.SessionInfo, error) {
					return []*service.SessionInfo{
						{ID: "sess-1", ConfigName: "easy"},
						{ID: "sess-2", ConfigName: "hard"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				if resp["count"].(float64) != 2 {
					t.Errorf("Expected count 2, got %v", resp["count"])
				}
				sessions := resp["sessions"].([]interface{})
				if len(sessions) != 2 {
					t.Errorf("Expected 2 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name: "Handle empty session list",
			setupMock: func(m *MockGameService) {
				m.ListSessionsFunc = func(ctx context.Context) ([]*service.SessionInfo, error) {
					return []*service.SessionInfo{}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				if resp["count"].(float64) != 0 {
					t.Errorf("Expected count 0, got %v", resp["count"])
				}
			},
		},
		{
			name: "Handle service error",
			setupMock: func(m *MockGameService) {
				m.ListSessionsFunc = func(ctx context.Context) ([]*service.SessionInfo, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "database error" {
					t.Errorf("Expected error 'database error', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("GET", "/api/sessions", nil)

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Get existing session",
			sessionID: "sess-123",
			setupMock: func(m *MockGameService) {
				m.GetSessionFunc = func(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
					if sessionID != "sess-123" {
						return nil, fmt.Errorf("session not found")
					}
					return &service.SessionInfo{
						ID:         sessionID,
						ConfigName: "test-config",
						CreatedAt:  time.Now(),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.SessionInfo
				parseResponse(t, w, &resp)
				if resp.ID != "sess-123" {
					t.Errorf("Expected session ID sess-123, got %s", resp.ID)
				}
			},
		},
		{
			name:      "Session not found",
			sessionID: "nonexistent",
			setupMock: func(m *MockGameService) {
				m.GetSessionFunc = func(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
					return nil, fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "session not found" {
					t.Errorf("Expected error 'session not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("GET", "/api/sessions/"+tt.sessionID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleGetSession(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestDeleteSession(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Delete existing session",
			sessionID: "sess-123",
			setupMock: func(m *MockGameService) {
				m.DeleteSessionFunc = func(ctx context.Context, sessionID string) error {
					if sessionID != "sess-123" {
						return fmt.Errorf("session not found")
					}
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["message"] != "Session sess-123 deleted" {
					t.Errorf("Unexpected message: %s", resp["message"])
				}
			},
		},
		{
			name:      "Delete non-existent session",
			sessionID: "nonexistent",
			setupMock: func(m *MockGameService) {
				m.DeleteSessionFunc = func(ctx context.Context, sessionID string) error {
					return fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "session not found" {
					t.Errorf("Expected error 'session not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("DELETE", "/api/sessions/"+tt.sessionID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleDeleteSession(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

// Game Operations Tests

func TestMove(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		requestBody    map[string]interface{}
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Valid move up",
			sessionID:   "sess-123",
			requestBody: map[string]interface{}{"direction": "up"},
			setupMock: func(m *MockGameService) {
				m.MoveFunc = func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
					if direction != "up" {
						t.Errorf("Expected direction 'up', got %s", direction)
					}
					return &service.MoveResult{
						Success: true,
						GameState: &engine.GameState{
							PlayerPos: engine.Position{X: 5, Y: 4},
							Battery:   79,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.MoveResult
				parseResponse(t, w, &resp)
				if !resp.Success {
					t.Error("Expected success to be true")
				}
				if resp.GameState.PlayerPos.Y != 4 {
					t.Errorf("Expected Y position 4, got %d", resp.GameState.PlayerPos.Y)
				}
			},
		},
		{
			name:        "Move with reset",
			sessionID:   "sess-123",
			requestBody: map[string]interface{}{"direction": "right", "reset": true},
			setupMock: func(m *MockGameService) {
				m.MoveFunc = func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
					if !reset {
						t.Error("Expected reset to be true")
					}
					return &service.MoveResult{
						Success: true,
						GameState: &engine.GameState{
							PlayerPos: engine.Position{X: 1, Y: 0},
							Battery:   100,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.MoveResult
				parseResponse(t, w, &resp)
				if resp.GameState.Battery != 100 {
					t.Errorf("Expected battery 100 after reset, got %d", resp.GameState.Battery)
				}
			},
		},
		{
			name:        "Invalid request body",
			sessionID:   "sess-123",
			requestBody: map[string]interface{}{"invalid": "field"},
			setupMock: func(m *MockGameService) {
				m.MoveFunc = func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
					// Empty direction should cause an error
					if direction == "" {
						return nil, fmt.Errorf("invalid direction")
					}
					return &service.MoveResult{Success: true, GameState: &engine.GameState{}}, nil
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "invalid direction" {
					t.Errorf("Expected error 'invalid direction', got %s", resp["error"])
				}
			},
		},
		{
			name:        "Session not found",
			sessionID:   "nonexistent",
			requestBody: map[string]interface{}{"direction": "up"},
			setupMock: func(m *MockGameService) {
				m.MoveFunc = func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
					return nil, fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "session not found" {
					t.Errorf("Expected error 'session not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("POST", "/api/sessions/"+tt.sessionID+"/move", tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleMove(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestBulkMove(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		requestBody    map[string]interface{}
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Multiple valid moves",
			sessionID:   "sess-123",
			requestBody: map[string]interface{}{"moves": []string{"up", "right", "down"}},
			setupMock: func(m *MockGameService) {
				m.BulkMoveFunc = func(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error) {
					if len(moves) != 3 {
						t.Errorf("Expected 3 moves, got %d", len(moves))
					}
					return &service.BulkMoveResult{
						Success:       true,
						GameState:     &engine.GameState{Battery: 77},
						MovesExecuted: 3,
						TotalMoves:    3,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.BulkMoveResult
				parseResponse(t, w, &resp)
				if resp.MovesExecuted != 3 {
					t.Errorf("Expected 3 moves executed, got %d", resp.MovesExecuted)
				}
			},
		},
		{
			name:        "Bulk move with reset",
			sessionID:   "sess-123",
			requestBody: map[string]interface{}{"moves": []string{"up", "up"}, "reset": true},
			setupMock: func(m *MockGameService) {
				m.BulkMoveFunc = func(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error) {
					if !reset {
						t.Error("Expected reset to be true")
					}
					return &service.BulkMoveResult{
						Success:   true,
						GameState: &engine.GameState{Battery: 98},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.BulkMoveResult
				parseResponse(t, w, &resp)
				if resp.GameState.Battery != 98 {
					t.Errorf("Expected battery 98, got %d", resp.GameState.Battery)
				}
			},
		},
		{
			name:           "Empty moves array",
			sessionID:      "sess-123",
			requestBody:    map[string]interface{}{"moves": []string{}},
			setupMock:      nil,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.BulkMoveResult
				parseResponse(t, w, &resp)
				if resp.MovesExecuted != 0 {
					t.Errorf("Expected 0 moves executed for empty array, got %d", resp.MovesExecuted)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("POST", "/api/sessions/"+tt.sessionID+"/bulk-move", tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleBulkMove(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestReset(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Reset existing session",
			sessionID: "sess-123",
			setupMock: func(m *MockGameService) {
				m.ResetFunc = func(ctx context.Context, sessionID string) (*engine.GameState, error) {
					return &engine.GameState{
						PlayerPos:  engine.Position{X: 0, Y: 0},
						Battery:    100,
						GameOver:   false,
						TotalMoves: 0,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				if resp["message"] != "Game reset successfully" {
					t.Errorf("Expected success message, got %s", resp["message"])
				}
				state := resp["state"].(map[string]interface{})
				if state["battery"].(float64) != 100 {
					t.Error("Expected battery to be reset to 100")
				}
			},
		},
		{
			name:      "Reset non-existent session",
			sessionID: "nonexistent",
			setupMock: func(m *MockGameService) {
				m.ResetFunc = func(ctx context.Context, sessionID string) (*engine.GameState, error) {
					return nil, fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "session not found" {
					t.Errorf("Expected error 'session not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("POST", "/api/sessions/"+tt.sessionID+"/reset", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleReset(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestGetHistory(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		queryParams    string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Default pagination",
			sessionID:   "sess-123",
			queryParams: "",
			setupMock: func(m *MockGameService) {
				m.GetMoveHistoryFunc = func(ctx context.Context, sessionID string, opts service.HistoryOptions) (*service.HistoryResponse, error) {
					if opts.Page != 1 || opts.Limit != 20 {
						t.Errorf("Expected default page=1, limit=20, got page=%d, limit=%d", opts.Page, opts.Limit)
					}
					return &service.HistoryResponse{
						Moves: []engine.MoveHistoryEntry{
							{Action: "up"},
							{Action: "right"},
						},
						TotalMoves: 5,
						Page:       1,
						PageSize:   20,
						TotalPages: 1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.HistoryResponse
				parseResponse(t, w, &resp)
				if resp.PageSize != 20 {
					t.Errorf("Expected page size 20, got %d", resp.PageSize)
				}
			},
		},
		{
			name:        "Custom pagination parameters",
			sessionID:   "sess-123",
			queryParams: "?page=2&limit=10&order=asc",
			setupMock: func(m *MockGameService) {
				m.GetMoveHistoryFunc = func(ctx context.Context, sessionID string, opts service.HistoryOptions) (*service.HistoryResponse, error) {
					if opts.Page != 2 || opts.Limit != 10 || opts.Order != "asc" {
						t.Errorf("Expected page=2, limit=10, order=asc, got page=%d, limit=%d, order=%s",
							opts.Page, opts.Limit, opts.Order)
					}
					return &service.HistoryResponse{
						Page:     2,
						PageSize: 10,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.HistoryResponse
				parseResponse(t, w, &resp)
				if resp.Page != 2 || resp.PageSize != 10 {
					t.Errorf("Expected page 2 with size 10, got page %d with size %d",
						resp.Page, resp.PageSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/sessions/"+tt.sessionID+"/history"+tt.queryParams, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleGetHistory(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestGetGameState(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Get existing game state",
			sessionID: "sess-123",
			setupMock: func(m *MockGameService) {
				m.GetGameStateFunc = func(ctx context.Context, sessionID string) (*engine.GameState, error) {
					return &engine.GameState{
						PlayerPos:  engine.Position{X: 5, Y: 3},
						Battery:    75,
						Score:      150,
						GameOver:   false,
						TotalMoves: 25,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp engine.GameState
				parseResponse(t, w, &resp)
				if resp.Battery != 75 || resp.Score != 150 {
					t.Errorf("Expected battery=75, score=150, got battery=%d, score=%d", resp.Battery, resp.Score)
				}
			},
		},
		{
			name:      "Session not found",
			sessionID: "nonexistent",
			setupMock: func(m *MockGameService) {
				m.GetGameStateFunc = func(ctx context.Context, sessionID string) (*engine.GameState, error) {
					return nil, fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "session not found" {
					t.Errorf("Expected error 'session not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("GET", "/api/sessions/"+tt.sessionID+"/state", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			server.handleGetGameState(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestListConfigs(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "List available configs",
			setupMock: func(m *MockGameService) {
				m.ListConfigsFunc = func(ctx context.Context) ([]*service.ConfigInfo, error) {
					return []*service.ConfigInfo{
						{Name: "easy", Description: "Easy mode"},
						{Name: "hard", Description: "Hard mode"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp []*service.ConfigInfo
				parseResponse(t, w, &resp)
				if len(resp) != 2 {
					t.Errorf("Expected 2 configs, got %d", len(resp))
				}
			},
		},
		{
			name: "Handle service error",
			setupMock: func(m *MockGameService) {
				m.ListConfigsFunc = func(ctx context.Context) ([]*service.ConfigInfo, error) {
					return nil, fmt.Errorf("config error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "config error" {
					t.Errorf("Expected error 'config error', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("GET", "/api/configs", nil)

			server.handleListConfigs(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		configName     string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "Get existing config",
			configName: "easy",
			setupMock: func(m *MockGameService) {
				m.LoadConfigFunc = func(ctx context.Context, configName string) (*engine.GameConfig, error) {
					if configName != "easy" {
						return nil, fmt.Errorf("config not found")
					}
					return &engine.GameConfig{
						Name:        "easy",
						Description: "Easy mode configuration",
						GridSize:    10,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp engine.GameConfig
				parseResponse(t, w, &resp)
				if resp.Name != "easy" {
					t.Errorf("Expected config name 'easy', got %s", resp.Name)
				}
			},
		},
		{
			name:       "Strip .json extension",
			configName: "medium.json",
			setupMock: func(m *MockGameService) {
				m.LoadConfigFunc = func(ctx context.Context, configName string) (*engine.GameConfig, error) {
					if configName != "medium" {
						t.Errorf("Expected config name 'medium' (without .json), got %s", configName)
					}
					return &engine.GameConfig{Name: "medium"}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "Config not found",
			configName: "nonexistent",
			setupMock: func(m *MockGameService) {
				m.LoadConfigFunc = func(ctx context.Context, configName string) (*engine.GameConfig, error) {
					return nil, fmt.Errorf("config not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				parseResponse(t, w, &resp)
				if resp["error"] != "config not found" {
					t.Errorf("Expected error 'config not found', got %s", resp["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := makeRequest("GET", "/api/configs/"+tt.configName, nil)
			req = mux.SetURLVars(req, map[string]string{"name": tt.configName})

			server.handleGetConfig(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestUnifiedSessions(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockGameService)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Get all sessions",
			queryParams: "",
			setupMock: func(m *MockGameService) {
				m.ListSessionsFunc = func(ctx context.Context) ([]*service.SessionInfo, error) {
					return []*service.SessionInfo{
						{
							ID:         "sess-1",
							ConfigName: "easy",
							GameState: &engine.GameState{
								Battery: 80,
							},
							GameConfig: &engine.GameConfig{
								Layout: []string{"PPR", "RRR", "RRP"},
							},
						},
						{
							ID:         "sess-2",
							ConfigName: "easy",
							GameState: &engine.GameState{
								Battery: 60,
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				if resp["config_name"] != "easy" {
					t.Errorf("Expected config_name 'easy', got %v", resp["config_name"])
				}
				if resp["total_parks"].(float64) != 3 {
					t.Errorf("Expected 3 total parks, got %v", resp["total_parks"])
				}
				sessions := resp["sessions"].([]interface{})
				if len(sessions) != 2 {
					t.Errorf("Expected 2 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name:        "Filter by session IDs",
			queryParams: "?sessionIds=sess-1,sess-3",
			setupMock: func(m *MockGameService) {
				m.GetSessionFunc = func(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
					if sessionID == "sess-1" {
						return &service.SessionInfo{
							ID:         "sess-1",
							ConfigName: "easy",
							GameState:  &engine.GameState{},
						}, nil
					}
					if sessionID == "sess-3" {
						return &service.SessionInfo{
							ID:         "sess-3",
							ConfigName: "hard",
							GameState:  &engine.GameState{},
						}, nil
					}
					return nil, fmt.Errorf("not found")
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				sessions := resp["sessions"].([]interface{})
				if len(sessions) != 2 {
					t.Errorf("Expected 2 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name:        "Filter by config name",
			queryParams: "?configName=medium",
			setupMock: func(m *MockGameService) {
				m.ListSessionsFunc = func(ctx context.Context) ([]*service.SessionInfo, error) {
					return []*service.SessionInfo{
						{ID: "sess-1", ConfigName: "easy"},
						{ID: "sess-2", ConfigName: "medium"},
						{ID: "sess-3", ConfigName: "medium"},
						{ID: "sess-4", ConfigName: "hard"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				parseResponse(t, w, &resp)
				sessions := resp["sessions"].([]interface{})
				if len(sessions) != 2 {
					t.Errorf("Expected 2 medium sessions, got %d", len(sessions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/sessions/unified"+tt.queryParams, nil)

			server.handleUnifiedSessions(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, w)
			}
		})
	}
}

func TestWebSocket(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockGameService)
		expectedStatus int
	}{
		{
			name:           "Missing session parameter",
			queryParams:    "",
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Invalid session",
			queryParams: "?session=invalid",
			setupMock: func(m *MockGameService) {
				m.GetSessionFunc = func(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
					return nil, fmt.Errorf("session not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "Valid session",
			queryParams: "?session=sess-123",
			setupMock: func(m *MockGameService) {
				m.GetSessionFunc = func(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
					return &service.SessionInfo{
						ID:         sessionID,
						ConfigName: "test",
					}, nil
				}
			},
			expectedStatus: http.StatusSwitchingProtocols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockGameService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			server := setupTestServer(mockService)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/ws"+tt.queryParams, nil)

			// For WebSocket upgrade test, we need proper headers
			if tt.expectedStatus == http.StatusSwitchingProtocols {
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
				req.Header.Set("Sec-WebSocket-Version", "13")
			}

			server.handleWebSocket(w, req)

			// WebSocket upgrade fails in unit tests due to httptest.ResponseRecorder limitations
			if tt.expectedStatus == http.StatusSwitchingProtocols {
				// Can't test actual WebSocket upgrade with httptest.ResponseRecorder
				// It doesn't implement http.Hijacker interface
				// We accept 500 error in this case as it indicates the upgrade was attempted
				if w.Code == http.StatusInternalServerError {
					return
				}
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
