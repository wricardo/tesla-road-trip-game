package service

import (
	"context"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

// CreateSessionOptions controls optional per-session behavior at creation time.
type CreateSessionOptions struct {
	FogEnabled   bool
	FogRadius    int
	GridPassword string
	// MoveDelayMs sets the per-step delay (and websocket broadcast) for bulkMove. Single moves are not delayed.
	// nil leaves the session default unchanged.
	MoveDelayMs *int
}

const DefaultMoveDelayMs = 300

// BulkMoveOptions paces a bulk move for spectators.
type BulkMoveOptions struct {
	// StepDelay pauses between moves. The service lock is released while waiting.
	StepDelay time.Duration
	// OnStep, when set, receives a snapshot after every attempted move. It runs
	// without the service lock held.
	OnStep func(*engine.GameState)
}

// GameService defines all game-related operations
type GameService interface {
	// Session Management
	CreateSession(ctx context.Context, configName string, opts ...CreateSessionOptions) (*SessionInfo, error)
	GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
	ListSessions(ctx context.Context) ([]*SessionInfo, error)
	DeleteSession(ctx context.Context, sessionID string) error
	UpdateSessionDisplayName(ctx context.Context, sessionID, displayName string) (*SessionInfo, error)

	// Game Operations
	Move(ctx context.Context, sessionID, direction string, reset bool) (*MoveResult, error)
	BulkMove(ctx context.Context, sessionID string, moves []string, reset bool, opts BulkMoveOptions) (*BulkMoveResult, error)
	Reset(ctx context.Context, sessionID string) (*engine.GameState, error)

	// Game State
	GetGameState(ctx context.Context, sessionID string) (*engine.GameState, error)
	GetMoveHistory(ctx context.Context, sessionID string, opts HistoryOptions) (*HistoryResponse, error)

	// Map management
	ListMaps(ctx context.Context) ([]*MapInfo, error)
	LoadMap(ctx context.Context, mapName string) (*engine.GameConfig, error)
	SaveMap(ctx context.Context, mapName string, config *engine.GameConfig) error
	DeleteMap(ctx context.Context, mapName string) error
}

// SessionManager defines session storage operations
type SessionManager interface {
	Create(id string, config *engine.GameConfig) (*Session, error)
	Get(id string) (*Session, error)
	GetOrCreate(id string, config *engine.GameConfig) (*Session, error)
	List() []*Session
	Delete(id string) error
	UpdateLastAction(id string) error
	UpdateDisplayName(id, displayName string) error
	Save(id string) error
}

// ConfigManager handles game configuration loading
type ConfigManager interface {
	LoadConfig(name string) (*engine.GameConfig, error)
	ListConfigs() ([]*MapInfo, error)
	GetDefault() *engine.GameConfig
	SaveConfig(name string, config *engine.GameConfig) error
	DeleteConfig(name string) error
}

// Session represents an active game session
type Session struct {
	ID           string
	DisplayName  string
	Engine       *engine.GameEngine
	Config       *engine.GameConfig
	CreatedAt    time.Time
	LastActionAt time.Time
	FogEnabled   bool
	FogRadius    int
	GridPassword string
	MoveDelayMs  int
}
