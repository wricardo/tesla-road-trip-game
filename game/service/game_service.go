package service

import (
	"context"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
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
	Reset(ctx context.Context, sessionID string) (*engine.GameState, error)

	// Game State
	GetGameState(ctx context.Context, sessionID string) (*engine.GameState, error)
	GetMoveHistory(ctx context.Context, sessionID string, opts HistoryOptions) (*HistoryResponse, error)

	// Configuration
	ListConfigs(ctx context.Context) ([]*ConfigInfo, error)
	LoadConfig(ctx context.Context, configName string) (*engine.GameConfig, error)
	SaveConfig(ctx context.Context, configName string, config *engine.GameConfig) error
}

// SessionManager defines session storage operations
type SessionManager interface {
	Create(id string, config *engine.GameConfig) (*Session, error)
	Get(id string) (*Session, error)
	GetOrCreate(id string, config *engine.GameConfig) (*Session, error)
	List() []*Session
	Delete(id string) error
	UpdateLastAccessed(id string) error
	Save(id string) error
}

// ConfigManager handles game configuration loading
type ConfigManager interface {
	LoadConfig(name string) (*engine.GameConfig, error)
	ListConfigs() ([]*ConfigInfo, error)
	GetDefault() *engine.GameConfig
	SaveConfig(name string, config *engine.GameConfig) error
}

// Session represents an active game session
type Session struct {
	ID             string
	Engine         *engine.GameEngine
	Config         *engine.GameConfig
	CreatedAt      time.Time
	LastAccessedAt time.Time
}
