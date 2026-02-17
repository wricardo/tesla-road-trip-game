package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
)

var (
	ErrConfigNotFound = errors.New("configuration not found")
	ErrInvalidConfig  = errors.New("invalid configuration")
)

// Manager handles game configuration loading and caching
type Manager struct {
	configDir     string
	defaultConfig *engine.GameConfig
	configs       map[string]*engine.GameConfig
	mu            sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager(configDir string) (*Manager, error) {
	// Ensure config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("config directory does not exist: %s", configDir)
	}

	m := &Manager{
		configDir: configDir,
		configs:   make(map[string]*engine.GameConfig),
	}

	// Load default config
	if err := m.loadDefaultConfig(); err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	return m, nil
}

// LoadConfig loads a configuration by name
func (m *Manager) LoadConfig(name string) (*engine.GameConfig, error) {
	m.mu.RLock()
	// Check cache first
	if config, exists := m.configs[name]; exists {
		m.mu.RUnlock()
		return config, nil
	}
	m.mu.RUnlock()

	// Load from file
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if config, exists := m.configs[name]; exists {
		return config, nil
	}

	// Add .json extension if not present
	filename := name
	if !strings.HasSuffix(filename, ".json") {
		filename = name + ".json"
	}

	configPath := filepath.Join(m.configDir, filename)

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config
	var config engine.GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if err := engine.ValidateGameConfig(&config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Cache the config
	m.configs[name] = &config
	return &config, nil
}

// ListConfigs returns information about all available configurations
func (m *Manager) ListConfigs() ([]*service.ConfigInfo, error) {
	entries, err := os.ReadDir(m.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []*service.ConfigInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Remove .json extension for config name
		name := strings.TrimSuffix(entry.Name(), ".json")

		// Try to load the config to get details
		config, err := m.LoadConfig(name)
		if err != nil {
			// Skip invalid configs
			continue
		}

		configs = append(configs, &service.ConfigInfo{
			Filename:    entry.Name(),
			ConfigID:    name, // This is the identifier to use for session creation
			Name:        config.Name,
			Description: config.Description,
			GridSize:    config.GridSize,
			MaxBattery:  config.MaxBattery,
		})
	}

	return configs, nil
}

// GetDefault returns the default configuration
func (m *Manager) GetDefault() *engine.GameConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultConfig
}

// SetDefault sets the default configuration by name
func (m *Manager) SetDefault(name string) error {
	config, err := m.LoadConfig(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultConfig = config
	return nil
}

// RefreshCache reloads all cached configurations from disk
func (m *Manager) RefreshCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear cache
	m.configs = make(map[string]*engine.GameConfig)

	// Reload default config
	return m.loadDefaultConfig()
}

// loadDefaultConfig loads the default configuration
func (m *Manager) loadDefaultConfig() error {
	// Try to load classic.json as default
	config, err := m.LoadConfig("classic")
	if err != nil {
		// Try to load the first available config
		configs, listErr := m.ListConfigs()
		if listErr != nil || len(configs) == 0 {
			// Create a minimal default config
			m.defaultConfig = m.createMinimalConfig()
			return nil
		}

		// Use the first available config
		config, err = m.LoadConfig(strings.TrimSuffix(configs[0].Filename, ".json"))
		if err != nil {
			m.defaultConfig = m.createMinimalConfig()
			return nil
		}
	}

	m.defaultConfig = config
	return nil
}

// SaveConfig saves a configuration to disk
func (m *Manager) SaveConfig(name string, config *engine.GameConfig) error {
	// Validate config before saving
	if err := engine.ValidateGameConfig(config); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Add .json extension if not present
	filename := name
	if !strings.HasSuffix(filename, ".json") {
		filename = name + ".json"
	}

	configPath := filepath.Join(m.configDir, filename)

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.configs[name] = config
	m.mu.Unlock()

	return nil
}

// createMinimalConfig creates a minimal valid configuration
func (m *Manager) createMinimalConfig() *engine.GameConfig {
	return &engine.GameConfig{
		Name:            "default",
		Description:     "Default minimal configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 10,
		Layout: []string{
			"RRPRR",
			"RRRHR",
			"RRSRR",
			"RRRHR",
			"RRPRR",
		},
		WallCrashEndsGame: false,
	}
}
