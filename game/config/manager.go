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
	// ErrMapExists is returned by CreateCustom when the name is already taken by
	// a built-in or another user map; creation never overwrites.
	ErrMapExists = errors.New("map already exists")
)

// Manager handles game configuration loading and caching.
//
// Maps live in two places:
//   - configDir: the shipped/built-in maps (read-only for API users).
//   - customDir: maps created through the API, each with an edit-token sidecar.
//
// A custom map can never shadow a built-in one: creation fails when the name is
// taken in either directory.
type Manager struct {
	configDir     string
	customDir     string
	defaultConfig *engine.GameConfig
	configs       map[string]*engine.GameConfig
	mu            sync.RWMutex
}

// NewManager creates a new configuration manager. User-created maps are stored
// in CUSTOM_MAPS_DIR when set, otherwise in "<configDir>/custom".
func NewManager(configDir string) (*Manager, error) {
	// Ensure config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("config directory does not exist: %s", configDir)
	}

	customDir := strings.TrimSpace(os.Getenv("CUSTOM_MAPS_DIR"))
	if customDir == "" {
		customDir = filepath.Join(configDir, "custom")
	}

	if err := os.MkdirAll(customDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create custom maps directory %s: %w", customDir, err)
	}

	m := &Manager{
		configDir: configDir,
		customDir: customDir,
		configs:   make(map[string]*engine.GameConfig),
	}

	// Load default config
	if err := m.loadDefaultConfig(); err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	return m, nil
}

// CustomDir returns the directory holding user-created maps.
func (m *Manager) CustomDir() string { return m.customDir }

// IsBuiltin reports whether name is a shipped map (i.e. not user-created).
func (m *Manager) IsBuiltin(name string) bool {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), ".json")
	if base == "" || base == "." || base == ".." {
		return false
	}
	return fileExists(filepath.Join(m.configDir, base+".json"))
}

// isCustom reports whether name resolves to a user-created map file.
func (m *Manager) isCustom(name string) bool {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), ".json")
	if base == "" || base == "." || base == ".." {
		return false
	}
	return fileExists(filepath.Join(m.customDir, base+".json"))
}

// configPath resolves the on-disk location of an existing map, preferring the
// built-in directory. Returns "" when the map exists in neither directory.
func (m *Manager) configPath(name string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), ".json")
	if base == "" || base == "." || base == ".." {
		return ""
	}
	for _, dir := range []string{m.configDir, m.customDir} {
		p := filepath.Join(dir, base+".json")
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// LoadConfig loads a configuration by name
func (m *Manager) LoadConfig(name string) (*engine.GameConfig, error) {
	if err := validateConfigName(name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

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

	configPath := m.configPath(filename)
	if configPath == "" {
		configPath = filepath.Join(m.configDir, filename)
	}

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

// ListConfigs returns information about all available maps (implements ConfigManager interface)
func (m *Manager) ListConfigs() ([]*service.MapInfo, error) {
	var maps []*service.MapInfo
	seen := make(map[string]bool)

	// Built-in maps first, so a stale custom file with the same name can never
	// masquerade as one.
	for _, dir := range []struct {
		path    string
		builtin bool
	}{{m.configDir, true}, {m.customDir, false}} {
		entries, err := os.ReadDir(dir.path)
		if err != nil {
			if dir.builtin {
				return nil, fmt.Errorf("failed to read config directory: %w", err)
			}
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".json")
			if seen[name] {
				continue
			}

			config, err := m.LoadConfig(name)
			if err != nil {
				continue
			}
			seen[name] = true

			maps = append(maps, &service.MapInfo{
				Filename:    entry.Name(),
				MapID:       name,
				Name:        config.Name,
				Description: config.Description,
				GridSize:    config.GridSize,
				MaxBattery:  config.MaxBattery,
				Builtin:     dir.builtin,
			})
		}
	}

	return maps, nil
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

// SaveConfig overwrites an existing map. Built-in maps are protected: callers
// must go through CreateCustom for new maps, so a stray save can never clobber a
// shipped map by accident.
func (m *Manager) SaveConfig(name string, config *engine.GameConfig) error {
	if err := validateConfigName(name); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Validate config before saving
	if err := engine.ValidateGameConfig(config); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if !m.isCustom(name) {
		if m.IsBuiltin(name) {
			return fmt.Errorf("%w: %q is a built-in map", ErrReadOnlyMap, name)
		}
		return fmt.Errorf("%w: %q does not exist; use CreateCustom", ErrConfigNotFound, name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeCustomLocked(name, config)
}

// SaveBuiltin overwrites a built-in map file. This is the admin-only escape
// hatch; everything else goes through SaveConfig, which refuses built-ins.
func (m *Manager) SaveBuiltin(name string, config *engine.GameConfig) error {
	if err := validateConfigName(name); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if err := engine.ValidateGameConfig(config); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if !m.IsBuiltin(name) {
		return fmt.Errorf("%w: %q is not a built-in map", ErrConfigNotFound, name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	base := strings.TrimSuffix(name, ".json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(m.configDir, base+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	m.configs[base] = config
	m.configs[name] = config
	return nil
}

// DeleteBuiltin removes a built-in map file. Admin-only, like SaveBuiltin.
func (m *Manager) DeleteBuiltin(name string) error {
	if err := validateConfigName(name); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if !m.IsBuiltin(name) {
		return fmt.Errorf("%w: %q is not a built-in map", ErrConfigNotFound, name)
	}

	base := strings.TrimSuffix(name, ".json")
	if err := os.Remove(filepath.Join(m.configDir, base+".json")); err != nil {
		return fmt.Errorf("failed to delete map: %w", err)
	}
	m.mu.Lock()
	delete(m.configs, base)
	delete(m.configs, name)
	m.mu.Unlock()
	return nil
}

// DeleteConfig removes a user-created map config file and evicts it from cache.
// Built-in maps can never be deleted through this path.
func (m *Manager) DeleteConfig(name string) error {
	if err := validateConfigName(name); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if !m.isCustom(name) {
		if m.IsBuiltin(name) {
			return fmt.Errorf("%w: %q is a built-in map", ErrReadOnlyMap, name)
		}
		return fmt.Errorf("map %q not found", name)
	}

	filename := name
	if !strings.HasSuffix(filename, ".json") {
		filename = name + ".json"
	}
	configPath := filepath.Join(m.customDir, filename)
	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("map %q not found", name)
		}
		return fmt.Errorf("failed to delete map: %w", err)
	}
	os.Remove(configPath + editTokenSuffix)

	m.mu.Lock()
	delete(m.configs, name)
	delete(m.configs, strings.TrimSuffix(name, ".json"))
	m.mu.Unlock()
	return nil
}

func validateConfigName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	base := strings.TrimSuffix(name, ".json")
	if base == "" || base == "." || base == ".." || filepath.IsAbs(name) || strings.ContainsAny(name, `/\\`) || strings.Contains(base, "..") {
		return fmt.Errorf("invalid map name %q", name)
	}
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("invalid map name %q: use letters, numbers, hyphen, or underscore", name)
	}
	return nil
}

func (m *Manager) createMinimalConfig() *engine.GameConfig {
	cfg := &engine.GameConfig{
		Name:            "default",
		Description:     "Default minimal configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 10,
		Legend:          map[string]string{"R": "road", "H": "home", "P": "park", "S": "supercharger", "W": "water", "B": "building"},
		Layout: []string{
			"RRPRR",
			"RRRHR",
			"RRSRR",
			"RRRHR",
			"RRPRR",
		},
	}
	return cfg
}
