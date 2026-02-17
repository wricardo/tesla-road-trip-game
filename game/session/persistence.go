package session

import (
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/service"
)

// SessionPersistence defines the interface for persisting sessions
type SessionPersistence interface {
	// Save persists a session to storage
	Save(session *service.Session) error

	// Load retrieves a session from storage by ID
	Load(id string) (*service.Session, error)

	// Delete removes a session from storage
	Delete(id string) error

	// ListAll returns all persisted session IDs
	ListAll() ([]string, error)

	// Exists checks if a session exists in storage
	Exists(id string) bool
}

// PersistedSessionData represents the JSON structure for persisted sessions
type PersistedSessionData struct {
	ID             string    `json:"id"`
	ConfigName     string    `json:"config_name"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
	GameState      any       `json:"game_state"` // Will be *engine.GameState when loaded
}
