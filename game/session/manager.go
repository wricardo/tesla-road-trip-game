package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrInvalidSessionID     = errors.New("invalid session ID")
)

// Manager handles game session lifecycle
type Manager struct {
	sessions    map[string]*service.Session
	persistence SessionPersistence
	mu          sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*service.Session),
	}
}

// NewManagerWithPersistence creates a new session manager with persistence
func NewManagerWithPersistence(persistence SessionPersistence) *Manager {
	return &Manager{
		sessions:    make(map[string]*service.Session),
		persistence: persistence,
	}
}

// Create creates a new session with the given ID and configuration
func (m *Manager) Create(id string, config *engine.GameConfig) (*service.Session, error) {
	if id == "" {
		id = m.generateSessionID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists (case-insensitive)
	if m.sessionExists(id) {
		return nil, ErrSessionAlreadyExists
	}

	// Create game engine
	eng, err := engine.NewEngine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	// Create session
	session := &service.Session{
		ID:             id,
		Engine:         eng,
		Config:         config,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	m.sessions[strings.ToLower(id)] = session

	// Auto-save if persistence is enabled
	if m.persistence != nil {
		if err := m.persistence.Save(session); err != nil {
			// Log error but don't fail the creation
			fmt.Printf("Warning: Failed to persist session %s: %v\n", id, err)
		}
	}

	return session, nil
}

// Get retrieves a session by ID (case-insensitive)
func (m *Manager) Get(id string) (*service.Session, error) {
	m.mu.RLock()
	session, exists := m.sessions[strings.ToLower(id)]
	if !exists {
		// Try exact match for backward compatibility
		session, exists = m.sessions[id]
	}
	m.mu.RUnlock()

	if exists {
		return session, nil
	}

	// Try loading from persistence if not in memory
	if m.persistence != nil && m.persistence.Exists(id) {
		session, err := m.persistence.Load(id)
		if err != nil {
			return nil, fmt.Errorf("failed to load persisted session: %w", err)
		}

		// Add to memory cache
		m.mu.Lock()
		m.sessions[strings.ToLower(id)] = session
		m.mu.Unlock()

		return session, nil
	}

	return nil, ErrSessionNotFound
}

// GetOrCreate gets an existing session or creates a new one
func (m *Manager) GetOrCreate(id string, config *engine.GameConfig) (*service.Session, error) {
	// Try to get existing session first
	session, err := m.Get(id)
	if err == nil {
		return session, nil
	}

	// Create new session if not found
	if errors.Is(err, ErrSessionNotFound) {
		return m.Create(id, config)
	}

	return nil, err
}

// List returns all active sessions
func (m *Manager) List() []*service.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*service.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session)
	}

	return result
}

// Delete removes a session
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowerID := strings.ToLower(id)
	inMemory := false

	if _, exists := m.sessions[lowerID]; exists {
		delete(m.sessions, lowerID)
		inMemory = true
	} else if _, exists := m.sessions[id]; exists {
		delete(m.sessions, id)
		inMemory = true
	}

	// Delete from persistence if it exists
	if m.persistence != nil && m.persistence.Exists(id) {
		if err := m.persistence.Delete(id); err != nil {
			return fmt.Errorf("failed to delete persisted session: %w", err)
		}
		return nil
	}

	// If not in persistence and not in memory, it doesn't exist
	if !inMemory {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteFromMemory removes a session from memory only (not from persistence)
func (m *Manager) DeleteFromMemory(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowerID := strings.ToLower(id)

	if _, exists := m.sessions[lowerID]; exists {
		delete(m.sessions, lowerID)
		return nil
	}

	if _, exists := m.sessions[id]; exists {
		delete(m.sessions, id)
		return nil
	}

	return ErrSessionNotFound
}

// UpdateLastAccessed updates the last accessed time for a session
func (m *Manager) UpdateLastAccessed(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[strings.ToLower(id)]
	if !exists {
		// Try exact match for backward compatibility
		session, exists = m.sessions[id]
		if !exists {
			return ErrSessionNotFound
		}
	}

	session.LastAccessedAt = time.Now()

	// Auto-save if persistence is enabled
	if m.persistence != nil {
		if err := m.persistence.Save(session); err != nil {
			fmt.Printf("Warning: Failed to persist session %s after access update: %v\n", id, err)
		}
	}

	return nil
}

// Save saves a specific session to persistence
func (m *Manager) Save(id string) error {
	if m.persistence == nil {
		return nil // No persistence configured
	}

	m.mu.RLock()
	session, exists := m.sessions[strings.ToLower(id)]
	if !exists {
		// Try exact match for backward compatibility
		session, exists = m.sessions[id]
		if !exists {
			m.mu.RUnlock()
			return ErrSessionNotFound
		}
	}
	m.mu.RUnlock()

	return m.persistence.Save(session)
}

// CleanupExpiredSessions removes sessions that haven't been accessed in the given duration
func (m *Manager) CleanupExpiredSessions(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, session := range m.sessions {
		if session.LastAccessedAt.Before(cutoff) {
			delete(m.sessions, id)
			removed++
		}
	}

	return removed
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// generateSessionID generates a random 4-character session ID
func (m *Manager) generateSessionID() string {
	// Generate 2 random bytes (4 hex characters)
	bytes := make([]byte, 2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// sessionExists checks if a session exists (case-insensitive)
func (m *Manager) sessionExists(id string) bool {
	lowerID := strings.ToLower(id)
	if _, exists := m.sessions[lowerID]; exists {
		return true
	}
	// Also check exact match for backward compatibility
	_, exists := m.sessions[id]
	return exists
}

// LoadPersistedSessions loads all persisted sessions into memory
func (m *Manager) LoadPersistedSessions() error {
	if m.persistence == nil {
		return nil // No persistence configured
	}

	sessionIDs, err := m.persistence.ListAll()
	if err != nil {
		return fmt.Errorf("failed to list persisted sessions: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	loadedCount := 0
	for _, id := range sessionIDs {
		// Skip if already loaded in memory
		if _, exists := m.sessions[strings.ToLower(id)]; exists {
			continue
		}

		session, err := m.persistence.Load(id)
		if err != nil {
			fmt.Printf("Warning: Failed to load persisted session %s: %v\n", id, err)
			continue
		}

		m.sessions[strings.ToLower(id)] = session
		loadedCount++
	}

	if loadedCount > 0 {
		fmt.Printf("Loaded %d persisted sessions from storage\n", loadedCount)
	}

	return nil
}

// SaveAllSessions saves all in-memory sessions to persistence
func (m *Manager) SaveAllSessions() error {
	if m.persistence == nil {
		return nil // No persistence configured
	}

	m.mu.RLock()
	sessions := make([]*service.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	errorCount := 0
	for _, session := range sessions {
		if err := m.persistence.Save(session); err != nil {
			fmt.Printf("Warning: Failed to save session %s: %v\n", session.ID, err)
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("failed to save %d sessions", errorCount)
	}

	return nil
}
