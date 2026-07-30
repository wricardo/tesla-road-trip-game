package mcp

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

type mockGameService struct {
	moveFunc     func(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error)
	bulkMoveFunc func(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error)
	resetFunc    func(ctx context.Context, sessionID string) (*engine.GameState, error)
}

func (m *mockGameService) CreateSession(ctx context.Context, configName string) (*service.SessionInfo, error) {
	return &service.SessionInfo{}, nil
}
func (m *mockGameService) GetSession(ctx context.Context, sessionID string) (*service.SessionInfo, error) {
	return &service.SessionInfo{}, nil
}
func (m *mockGameService) ListSessions(ctx context.Context) ([]*service.SessionInfo, error) {
	return []*service.SessionInfo{}, nil
}
func (m *mockGameService) DeleteSession(ctx context.Context, sessionID string) error { return nil }
func (m *mockGameService) UpdateSessionDisplayName(ctx context.Context, sessionID, displayName string) (*service.SessionInfo, error) {
	return &service.SessionInfo{}, nil
}
func (m *mockGameService) Move(ctx context.Context, sessionID, direction string, reset bool) (*service.MoveResult, error) {
	if m.moveFunc != nil {
		return m.moveFunc(ctx, sessionID, direction, reset)
	}
	return &service.MoveResult{Success: true, GameState: &engine.GameState{}}, nil
}
func (m *mockGameService) BulkMove(ctx context.Context, sessionID string, moves []string, reset bool) (*service.BulkMoveResult, error) {
	if m.bulkMoveFunc != nil {
		return m.bulkMoveFunc(ctx, sessionID, moves, reset)
	}
	return &service.BulkMoveResult{Success: true, GameState: &engine.GameState{}}, nil
}
func (m *mockGameService) Reset(ctx context.Context, sessionID string) (*engine.GameState, error) {
	if m.resetFunc != nil {
		return m.resetFunc(ctx, sessionID)
	}
	return &engine.GameState{}, nil
}
func (m *mockGameService) GetGameState(ctx context.Context, sessionID string) (*engine.GameState, error) {
	return &engine.GameState{}, nil
}
func (m *mockGameService) GetMoveHistory(ctx context.Context, sessionID string, opts service.HistoryOptions) (*service.HistoryResponse, error) {
	return &service.HistoryResponse{}, nil
}
func (m *mockGameService) ListMaps(ctx context.Context) ([]*service.MapInfo, error) {
	return []*service.MapInfo{}, nil
}
func (m *mockGameService) LoadMap(ctx context.Context, mapName string) (*engine.GameConfig, error) {
	return &engine.GameConfig{}, nil
}
func (m *mockGameService) SaveMap(ctx context.Context, mapName string, config *engine.GameConfig) error {
	return nil
}
func (m *mockGameService) DeleteMap(ctx context.Context, mapName string) error { return nil }

func expectState(t *testing.T, ch <-chan *engine.GameState, want *engine.GameState) {
	t.Helper()
	select {
	case got := <-ch:
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected broadcast state: got %+v want %+v", got, want)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for broadcast state")
	}
}

func TestHandleMoveBroadcastsToSessionAndLobby(t *testing.T) {
	sessionID := "session-1"
	want := &engine.GameState{Battery: 42, Score: 3, PlayerPos: engine.Position{X: 2, Y: 1}}
	mockSvc := &mockGameService{
		moveFunc: func(ctx context.Context, gotSessionID, direction string, reset bool) (*service.MoveResult, error) {
			if gotSessionID != sessionID {
				t.Fatalf("unexpected session ID: got %q", gotSessionID)
			}
			return &service.MoveResult{Success: true, GameState: want}, nil
		},
	}
	hub := websocket.NewHub()
	server := NewServer(mockSvc, hub)

	sessionCtx, cancelSession := context.WithCancel(context.Background())
	defer cancelSession()
	lobbyCtx, cancelLobby := context.WithCancel(context.Background())
	defer cancelLobby()
	sessionSub := hub.SubscribeSession(sessionCtx, sessionID)
	lobbySub := hub.SubscribeLobby(lobbyCtx)

	_, err := server.handleMove(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"session_id": sessionID,
				"direction":  "right",
			},
		},
	})
	if err != nil {
		t.Fatalf("handleMove returned error: %v", err)
	}

	expectState(t, sessionSub, want)
	expectState(t, lobbySub, want)
}

func TestHandleBulkMoveBroadcastsToSession(t *testing.T) {
	sessionID := "session-2"
	want := &engine.GameState{Battery: 30, Score: 7, PlayerPos: engine.Position{X: 4, Y: 5}}
	mockSvc := &mockGameService{
		bulkMoveFunc: func(ctx context.Context, gotSessionID string, moves []string, reset bool) (*service.BulkMoveResult, error) {
			if gotSessionID != sessionID {
				t.Fatalf("unexpected session ID: got %q", gotSessionID)
			}
			return &service.BulkMoveResult{Success: true, GameState: want}, nil
		},
	}
	hub := websocket.NewHub()
	server := NewServer(mockSvc, hub)

	sessionCtx, cancelSession := context.WithCancel(context.Background())
	defer cancelSession()
	sessionSub := hub.SubscribeSession(sessionCtx, sessionID)

	_, err := server.handleBulkMove(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"session_id": sessionID,
				"moves":      []interface{}{"up", "left"},
			},
		},
	})
	if err != nil {
		t.Fatalf("handleBulkMove returned error: %v", err)
	}

	expectState(t, sessionSub, want)
}

func TestHandleResetBroadcastsToSession(t *testing.T) {
	sessionID := "session-3"
	want := &engine.GameState{Battery: 50, MaxBattery: 50, PlayerPos: engine.Position{X: 0, Y: 0}}
	mockSvc := &mockGameService{
		resetFunc: func(ctx context.Context, gotSessionID string) (*engine.GameState, error) {
			if gotSessionID != sessionID {
				t.Fatalf("unexpected session ID: got %q", gotSessionID)
			}
			return want, nil
		},
	}
	hub := websocket.NewHub()
	server := NewServer(mockSvc, hub)

	sessionCtx, cancelSession := context.WithCancel(context.Background())
	defer cancelSession()
	sessionSub := hub.SubscribeSession(sessionCtx, sessionID)

	_, err := server.handleReset(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"session_id": sessionID,
			},
		},
	})
	if err != nil {
		t.Fatalf("handleReset returned error: %v", err)
	}

	expectState(t, sessionSub, want)
}

func TestHandleMoveWithoutHubDoesNotPanic(t *testing.T) {
	mockSvc := &mockGameService{
		moveFunc: func(ctx context.Context, gotSessionID, direction string, reset bool) (*service.MoveResult, error) {
			return &service.MoveResult{Success: true, GameState: &engine.GameState{Battery: 1}}, nil
		},
	}
	server := NewServer(mockSvc)

	if _, err := server.handleMove(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"session_id": "session-nil-hub",
				"direction":  "up",
			},
		},
	}); err != nil {
		t.Fatalf("handleMove returned error: %v", err)
	}
}

func TestFilterGameState_DefaultDropsHistory(t *testing.T) {
	state := &engine.GameState{
		Grid: [][]engine.Cell{{{Type: engine.Road}}},
		MoveHistory: []engine.MoveHistoryEntry{
			{MoveNumber: 1, Action: "left"},
			{MoveNumber: 2, Action: "right"},
		},
		CurrentMoves: []engine.MoveHistoryEntry{
			{MoveNumber: 2, Action: "right"},
		},
		CurrentMovesCount: 1,
	}
	opts := &service.ResponseOptions{
		IncludeGrid: false,
	}

	filtered := filterGameState(state, opts)

	if filtered == nil {
		t.Fatal("expected filtered state")
	}
	if filtered.Grid != nil {
		t.Fatalf("expected grid to be omitted by default")
	}
	if filtered.MoveHistory != nil {
		t.Fatalf("expected move history to be omitted by default")
	}
	if filtered.CurrentMoves != nil {
		t.Fatalf("expected current moves to be omitted by default")
	}
	if filtered.CurrentMovesCount != 0 {
		t.Fatalf("expected current moves count to reset to 0, got %d", filtered.CurrentMovesCount)
	}
}

func TestFilterGameState_HistoryLimitApplied(t *testing.T) {
	state := &engine.GameState{
		MoveHistory: []engine.MoveHistoryEntry{
			{MoveNumber: 1, Action: "a1"},
			{MoveNumber: 2, Action: "a2"},
			{MoveNumber: 3, Action: "a3"},
		},
		CurrentMoves: []engine.MoveHistoryEntry{
			{MoveNumber: 10, Action: "c1"},
			{MoveNumber: 11, Action: "c2"},
			{MoveNumber: 12, Action: "c3"},
		},
		CurrentMovesCount: 3,
	}
	opts := &service.ResponseOptions{
		IncludeGrid:    true,
		IncludeHistory: true,
		HistoryLimit:   2,
	}

	filtered := filterGameState(state, opts)

	if filtered == nil {
		t.Fatal("expected filtered state")
	}
	if len(filtered.MoveHistory) != 2 || filtered.MoveHistory[0].MoveNumber != 2 || filtered.MoveHistory[1].MoveNumber != 3 {
		t.Fatalf("expected last 2 move history entries, got %+v", filtered.MoveHistory)
	}
	if len(filtered.CurrentMoves) != 2 || filtered.CurrentMoves[0].MoveNumber != 11 || filtered.CurrentMoves[1].MoveNumber != 12 {
		t.Fatalf("expected last 2 current move entries, got %+v", filtered.CurrentMoves)
	}
	if filtered.CurrentMovesCount != 2 {
		t.Fatalf("expected current move count 2, got %d", filtered.CurrentMovesCount)
	}
}

func TestHandleMove_DefaultOutputOmitsHistoryFields(t *testing.T) {
	mockSvc := &mockGameService{
		moveFunc: func(ctx context.Context, gotSessionID, direction string, reset bool) (*service.MoveResult, error) {
			return &service.MoveResult{
				Success: true,
				GameState: &engine.GameState{
					MoveHistory: []engine.MoveHistoryEntry{{MoveNumber: 1}},
					CurrentMoves: []engine.MoveHistoryEntry{
						{MoveNumber: 1},
					},
					CurrentMovesCount: 1,
				},
			}, nil
		},
	}
	server := NewServer(mockSvc)

	result, err := server.handleMove(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"session_id": "session-default-omit",
				"direction":  "left",
			},
		},
	})
	if err != nil {
		t.Fatalf("handleMove returned error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected result content")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if strings.Contains(text, "move_history") || strings.Contains(text, "current_moves") || strings.Contains(text, "current_moves_count") {
		t.Fatalf("expected history fields omitted from default output, got: %s", text)
	}
}
