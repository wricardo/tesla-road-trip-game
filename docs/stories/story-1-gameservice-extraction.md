# Story 1: Extract GameService Layer

## Story Overview
**Title:** Extract GameService Layer
**Status:** Draft
**Priority:** High
**Estimated:** 3 days
**Epic:** Architectural Refactoring - Separation of Concerns

## Story Description
Create `game/service/` package with clean interfaces consolidating all business logic from HTTP handlers and MCP implementations. This foundational service layer will become the single source of truth for all game operations, eliminating the current duplication across main.go, mcp_server.go, and mcp_embedded.go.

## Acceptance Criteria
- [ ] GameService interface defined with all game operations
- [ ] Service implementation using existing engine package
- [ ] All business logic moved from handlers to service
- [ ] Unit tests achieving 90%+ coverage
- [ ] Zero breaking changes to existing APIs

## Technical Design

### 1. GameService Interface Definition

Create `game/service/game_service.go`:

```go
package service

import (
    "context"
    "github.com/wricardo/mcp-training/statefullgame/game/engine"
)

// GameService defines all game-related operations
type GameService interface {
    // Session Management
    CreateSession(ctx context.Context, configName string) (*SessionInfo, error)
    GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
    ListSessions(ctx context.Context) ([]*SessionInfo, error)
    DeleteSession(ctx context.Context, sessionID string) error

    // Game Operations
    Move(ctx context.Context, sessionID, direction string, reset bool) (*MoveResult, error)
    BulkMove(ctx context.Context, sessionID string, moves []string, reset bool) (*BulkMoveResult, error)
    Reset(ctx context.Context, sessionID string) (*GameState, error)

    // Game State
    GetGameState(ctx context.Context, sessionID string) (*GameState, error)
    GetMoveHistory(ctx context.Context, sessionID string, opts HistoryOptions) (*HistoryResponse, error)

    // Configuration
    ListConfigs(ctx context.Context) ([]*ConfigInfo, error)
    LoadConfig(ctx context.Context, configName string) (*engine.GameConfig, error)
}

// Supporting Types
type SessionInfo struct {
    ID             string                `json:"id"`
    ConfigName     string                `json:"config_name"`
    CreatedAt      time.Time            `json:"created_at"`
    LastAccessedAt time.Time            `json:"last_accessed_at"`
    GameState      *engine.GameState    `json:"game_state"`
    GameConfig     *engine.GameConfig   `json:"game_config"`
}

type MoveResult struct {
    Success   bool                `json:"success"`
    GameState *engine.GameState   `json:"game_state"`
    Message   string              `json:"message"`
    Events    []GameEvent         `json:"events,omitempty"`
}

type BulkMoveResult struct {
    MovesExecuted int              `json:"moves_executed"`
    TotalMoves    int              `json:"total_moves"`
    Success       bool             `json:"success"`
    GameState     *engine.GameState `json:"game_state"`
    Events        []GameEvent      `json:"events"`
    StoppedReason string           `json:"stopped_reason,omitempty"`
}

type GameEvent struct {
    Type      string    `json:"type"` // "move", "charge", "park_visited", "game_over", "victory"
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
    Position  engine.Position `json:"position,omitempty"`
}

type HistoryOptions struct {
    Page    int    `json:"page"`
    Limit   int    `json:"limit"`
    Order   string `json:"order"` // "asc" or "desc"
}

type HistoryResponse struct {
    Moves       []engine.MoveHistoryEntry `json:"moves"`
    TotalMoves  int                       `json:"total_moves"`
    Page        int                       `json:"page"`
    PageSize    int                       `json:"page_size"`
    TotalPages  int                       `json:"total_pages"`
    HasNext     bool                      `json:"has_next"`
    HasPrevious bool                      `json:"has_previous"`
}

type ConfigInfo struct {
    Filename    string `json:"filename"`
    Name        string `json:"name"`
    Description string `json:"description"`
    GridSize    int    `json:"grid_size"`
    MaxBattery  int    `json:"max_battery"`
}
```

### 2. Service Implementation

Create `game/service/game_service_impl.go`:

```go
package service

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/wricardo/mcp-training/statefullgame/game/engine"
    "github.com/wricardo/mcp-training/statefullgame/game/session"
)

type gameServiceImpl struct {
    sessions   session.Manager
    configs    ConfigManager
    mu         sync.RWMutex
}

// NewGameService creates a new game service instance
func NewGameService(sessions session.Manager, configs ConfigManager) GameService {
    return &gameServiceImpl{
        sessions: sessions,
        configs:  configs,
    }
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
    sess.LastAccessedAt = time.Now()

    // Handle reset if requested
    if reset {
        sess.Engine.Reset()
        s.recordEvent(sess, GameEvent{
            Type:      "reset",
            Message:   "Game reset to initial state",
            Timestamp: time.Now(),
        })
    }

    // Execute move
    prevPos := sess.Engine.GetPlayerPosition()
    success := sess.Engine.Move(direction)
    newPos := sess.Engine.GetPlayerPosition()

    // Build result
    result := &MoveResult{
        Success:   success,
        GameState: sess.Engine.GetState(),
        Message:   sess.Engine.GetState().Message,
    }

    // Record events
    if success {
        result.Events = s.extractMoveEvents(sess, prevPos, newPos, direction)
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
    sess.LastAccessedAt = time.Now()

    // Handle reset
    if reset {
        sess.Engine.Reset()
    }

    // Limit moves to prevent abuse
    if len(moves) > engine.MaxBulkMoves {
        moves = moves[:engine.MaxBulkMoves]
    }

    result := &BulkMoveResult{
        TotalMoves: len(moves),
        Events:     make([]GameEvent, 0),
        Success:    true,
    }

    // Execute moves
    for i, move := range moves {
        if sess.Engine.IsGameOver() {
            result.StoppedReason = "game_over"
            break
        }

        prevPos := sess.Engine.GetPlayerPosition()
        success := sess.Engine.Move(move)

        if !success {
            result.Success = false
            result.StoppedReason = fmt.Sprintf("move %d blocked: %s", i+1, move)
            break
        }

        result.MovesExecuted++
        newPos := sess.Engine.GetPlayerPosition()

        // Collect events for this move
        events := s.extractMoveEvents(sess, prevPos, newPos, move)
        result.Events = append(result.Events, events...)
    }

    result.GameState = sess.Engine.GetState()
    return result, nil
}

// extractMoveEvents generates events from a move
func (s *gameServiceImpl) extractMoveEvents(sess *session.Session, prevPos, newPos engine.Position, direction string) []GameEvent {
    events := []GameEvent{}
    state := sess.Engine.GetState()

    // Basic move event
    events = append(events, GameEvent{
        Type:      "move",
        Message:   fmt.Sprintf("Moved %s to (%d,%d)", direction, newPos.X, newPos.Y),
        Timestamp: time.Now(),
        Position:  newPos,
    })

    // Check for special cell events
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

// GetMoveHistory returns paginated move history
func (s *gameServiceImpl) GetMoveHistory(ctx context.Context, sessionID string, opts HistoryOptions) (*HistoryResponse, error) {
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
        // Reverse order
        for i := total - 1 - start; i >= 0 && i >= total - end; i-- {
            moves = append(moves, history[i])
        }
    } else {
        if start < total {
            moves = history[start:end]
        }
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

// Additional methods implementation...
// (CreateSession, GetSession, ListSessions, Reset, GetGameState, etc.)
```

### 3. Migration Strategy

#### Phase 1: Create Service Package (Day 1)
1. Create directory structure: `game/service/`
2. Define interfaces and types
3. Implement core methods with existing engine
4. Write comprehensive unit tests

#### Phase 2: Update HTTP Handlers (Day 2)
Replace existing handler logic with service calls:

**Before (main.go):**
```go
func handleMove(w http.ResponseWriter, r *http.Request) {
    session, err := getSessionFromRequest(r)
    // ... lots of business logic ...
    success := session.GameState.MovePlayer(direction, session.GameConfig)
    // ... more business logic ...
    broadcastSessionState(session)
    json.NewEncoder(w).Encode(session.GameState)
}
```

**After (main.go):**
```go
func handleMove(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("sessionId")
    var req MoveRequest
    json.NewDecoder(r.Body).Decode(&req)

    result, err := gameService.Move(r.Context(), sessionID, req.Direction, req.Reset)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    hub.Broadcast(sessionID, result.GameState)
    json.NewEncoder(w).Encode(result)
}
```

#### Phase 3: Update MCP Handlers (Day 3)
Similar simplification for MCP handlers:

**Before (mcp_embedded.go):**
```go
func (s *EmbeddedMCPServer) handleMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ... session management ...
    // ... business logic ...
    success := session.GameState.MovePlayer(direction, session.GameConfig)
    // ... formatting logic ...
}
```

**After (mcp_embedded.go):**
```go
func (s *EmbeddedMCPServer) handleMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := parseArgs(request)
    result, err := s.gameService.Move(ctx, args.SessionID, args.Direction, args.Reset)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(formatMoveResult(result)), nil
}
```

## Testing Strategy

### Unit Tests
Create `game/service/game_service_test.go`:

```go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestGameService_Move(t *testing.T) {
    tests := []struct {
        name      string
        sessionID string
        direction string
        reset     bool
        wantErr   bool
    }{
        {
            name:      "valid move up",
            sessionID: "test123",
            direction: "up",
            reset:     false,
            wantErr:   false,
        },
        {
            name:      "move with reset",
            sessionID: "test123",
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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockSessions := new(MockSessionManager)
            mockConfigs := new(MockConfigManager)
            service := NewGameService(mockSessions, mockConfigs)

            // Execute
            result, err := service.Move(context.Background(), tt.sessionID, tt.direction, tt.reset)

            // Assert
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}

func TestGameService_BulkMove(t *testing.T) {
    // Test bulk move scenarios
    // - Success path with multiple moves
    // - Stopping on game over
    // - Stopping on blocked move
    // - Move limit enforcement
}

func TestGameService_ConcurrentMoves(t *testing.T) {
    // Test thread safety with concurrent moves
    // Use goroutines to simulate multiple clients
}
```

### Integration Tests
Test service with real engine:

```go
func TestGameServiceIntegration(t *testing.T) {
    // Create real dependencies
    sessions := session.NewManager()
    configs := config.NewManager("configs/")
    service := NewGameService(sessions, configs)

    // Create session
    sess, err := service.CreateSession(context.Background(), "classic")
    assert.NoError(t, err)

    // Test move sequence
    moves := []string{"up", "right", "right", "down"}
    result, err := service.BulkMove(context.Background(), sess.ID, moves, false)

    assert.NoError(t, err)
    assert.Equal(t, 4, result.MovesExecuted)
}
```

## Implementation Checklist

### Day 1: Foundation
- [ ] Create `game/service/` directory
- [ ] Define GameService interface
- [ ] Define supporting types (MoveResult, BulkMoveResult, etc.)
- [ ] Implement core methods (Move, BulkMove)
- [ ] Write unit tests for core methods

### Day 2: Complete Implementation
- [ ] Implement session management methods
- [ ] Implement configuration methods
- [ ] Implement history methods
- [ ] Add comprehensive error handling
- [ ] Write integration tests

### Day 3: Migration
- [ ] Update HTTP handlers to use service
- [ ] Update MCP handlers to use service
- [ ] Remove duplicate business logic
- [ ] Run full test suite
- [ ] Performance benchmarks

## Success Metrics
- [ ] 90%+ test coverage for service package
- [ ] All existing APIs continue working
- [ ] Performance benchmarks show <5% degradation
- [ ] Code duplication reduced by at least 500 lines
- [ ] All business logic centralized in service layer

## Dependencies
- Requires session.Manager interface (Story 2)
- Can use mock for initial development
- Engine package remains unchanged

## Risks & Mitigations
1. **Risk:** Breaking existing API contracts
   - **Mitigation:** Comprehensive integration tests before deployment

2. **Risk:** Performance degradation from abstraction
   - **Mitigation:** Benchmark critical paths, optimize if needed

3. **Risk:** Concurrent access issues
   - **Mitigation:** Proper mutex usage, race detector testing

## Definition of Done
- [ ] All acceptance criteria met
- [ ] Code review completed
- [ ] Unit tests passing (90%+ coverage)
- [ ] Integration tests passing
- [ ] No performance regression
- [ ] Documentation updated
- [ ] PR approved and merged

## Notes
- This is the foundation for all subsequent refactoring
- Keep interfaces clean and focused
- Don't leak transport concerns into service
- Use context for cancellation support
- Consider adding metrics/logging hooks