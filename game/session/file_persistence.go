package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wricardo/mcp-training/statefullgame/game/engine"
	"github.com/wricardo/mcp-training/statefullgame/game/service"
)

// FilePersistence implements SessionPersistence using file system storage
type FilePersistence struct {
	sessionsDir   string
	configManager service.ConfigManager
}

// NewFilePersistence creates a new file-based session persistence layer
func NewFilePersistence(sessionsDir string, configManager service.ConfigManager) (*FilePersistence, error) {
	// Create sessions directory if it doesn't exist
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &FilePersistence{
		sessionsDir:   sessionsDir,
		configManager: configManager,
	}, nil
}

// Save persists a session to a JSON file
func (fp *FilePersistence) Save(session *service.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Get config ID from display name
	configID, err := fp.getConfigIDFromName(session.Config.Name)
	if err != nil {
		return fmt.Errorf("failed to get config ID: %w", err)
	}

	// Create persisted data structure
	data := PersistedSessionData{
		ID:             session.ID,
		ConfigName:     configID, // Store config ID, not display name
		CreatedAt:      session.CreatedAt,
		LastAccessedAt: session.LastAccessedAt,
		GameState:      session.Engine.GetState(),
	}

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Write to file
	filePath := fp.getFilePath(session.ID)
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load retrieves a session from a JSON file
func (fp *FilePersistence) Load(id string) (*service.Session, error) {
	filePath := fp.getFilePath(id)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, ErrSessionNotFound
	}

	// Read file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Unmarshal JSON
	var data PersistedSessionData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Load the game configuration
	gameConfig, err := fp.configManager.LoadConfig(data.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("failed to load config '%s': %w", data.ConfigName, err)
	}

	// Create game engine with configuration
	gameEngine, err := engine.NewEngine(gameConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create game engine: %w", err)
	}

	// Restore game state
	gameStateJSON, err := json.Marshal(data.GameState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal game state: %w", err)
	}

	var gameState engine.GameState
	if err := json.Unmarshal(gameStateJSON, &gameState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game state: %w", err)
	}

	// Set the restored state to the engine
	if err := gameEngine.SetState(&gameState); err != nil {
		return nil, fmt.Errorf("failed to set game state: %w", err)
	}

	// Create session
	session := &service.Session{
		ID:             data.ID,
		Engine:         gameEngine,
		Config:         gameConfig,
		CreatedAt:      data.CreatedAt,
		LastAccessedAt: data.LastAccessedAt,
	}

	return session, nil
}

// Delete removes a session file
func (fp *FilePersistence) Delete(id string) error {
	filePath := fp.getFilePath(id)

	// Check if file exists
	if !fp.Exists(id) {
		return ErrSessionNotFound
	}

	// Remove file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove session file: %w", err)
	}

	return nil
}

// ListAll returns all persisted session IDs
func (fp *FilePersistence) ListAll() ([]string, error) {
	entries, err := os.ReadDir(fp.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessionIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			// Remove .json extension to get session ID
			sessionID := strings.TrimSuffix(name, ".json")
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	return sessionIDs, nil
}

// Exists checks if a session file exists
func (fp *FilePersistence) Exists(id string) bool {
	filePath := fp.getFilePath(id)
	_, err := os.Stat(filePath)
	return err == nil
}

// getFilePath returns the full file path for a session ID
func (fp *FilePersistence) getFilePath(id string) string {
	return filepath.Join(fp.sessionsDir, fmt.Sprintf("%s.json", id))
}

// getConfigIDFromName returns the config ID (filename without extension) from display name
func (fp *FilePersistence) getConfigIDFromName(displayName string) (string, error) {
	configs, err := fp.configManager.ListConfigs()
	if err != nil {
		return "", fmt.Errorf("failed to list configs: %w", err)
	}

	for _, config := range configs {
		if config.Name == displayName {
			return config.ConfigID, nil
		}
	}

	// If not found, assume the displayName is already the config ID
	return displayName, nil
}
