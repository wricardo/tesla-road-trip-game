package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

func createTestConfigDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func createValidConfig() *engine.GameConfig {
	return &engine.GameConfig{
		Name:            "Test Config",
		Description:     "Test configuration",
		GridSize:        5,
		MaxBattery:      10,
		StartingBattery: 8,
		Layout: []string{
			"BBBBB",
			"BRHPB",
			"BRRSB",
			"BPPPB",
			"BBBBB",
		},
		Legend: map[string]string{
			"R": "road", "H": "home", "P": "park",
			"S": "supercharger", "W": "water", "B": "building",
		},
		WallCrashEndsGame: false,
		Messages: struct {
			Welcome            string `json:"welcome"`
			HomeCharge         string `json:"home_charge"`
			SuperchargerCharge string `json:"supercharger_charge"`
			ParkVisited        string `json:"park_visited"`
			ParkAlreadyVisited string `json:"park_already_visited"`
			Victory            string `json:"victory"`
			OutOfBattery       string `json:"out_of_battery"`
			Stranded           string `json:"stranded"`
			CantMove           string `json:"cant_move"`
			BatteryStatus      string `json:"battery_status"`
			HitWall            string `json:"hit_wall"`
		}{
			Welcome:            "Welcome!",
			HomeCharge:         "Home charged!",
			SuperchargerCharge: "Supercharged!",
			ParkVisited:        "Park visited! Score: %d",
			ParkAlreadyVisited: "Already visited",
			Victory:            "Victory! All %d parks!",
			OutOfBattery:       "No battery!",
			Stranded:           "Stranded!",
			CantMove:           "Can't move!",
			BatteryStatus:      "Battery: %d/%d",
			HitWall:            "Hit wall!",
		},
	}
}

func writeConfigFile(t *testing.T, dir, name string, config *engine.GameConfig) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	filename := name
	if filepath.Ext(filename) == "" {
		filename = name + ".json"
	}

	path := filepath.Join(dir, filename)
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
}

func TestNewManager(t *testing.T) {
	t.Run("valid directory", func(t *testing.T) {
		dir := createTestConfigDir(t)
		defer os.RemoveAll(dir)

		// Create default config
		defaultConfig := createValidConfig()
		defaultConfig.Name = "Default"
		writeConfigFile(t, dir, "default", defaultConfig)

		manager, err := NewManager(dir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}
		if manager == nil {
			t.Error("Expected manager to be non-nil")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := NewManager("/non/existent/path")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})

	t.Run("missing default config", func(t *testing.T) {
		dir := createTestConfigDir(t)
		defer os.RemoveAll(dir)

		manager, err := NewManager(dir)
		if err != nil {
			t.Errorf("NewManager should succeed even without config files, got error: %v", err)
		}

		// Should have created a minimal default config
		if manager == nil {
			t.Fatal("Expected manager to be created")
		}

		// Should be able to get the default config
		defaultConfig := manager.GetDefault()
		if defaultConfig == nil {
			t.Error("Expected default config to be available")
		}
	})
}

func TestManager_LoadConfig(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	// Create test configs
	defaultConfig := createValidConfig()
	defaultConfig.Name = "Default"
	writeConfigFile(t, dir, "default", defaultConfig)

	easyConfig := createValidConfig()
	easyConfig.Name = "Easy"
	easyConfig.MaxBattery = 20
	writeConfigFile(t, dir, "easy", easyConfig)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("load existing config", func(t *testing.T) {
		config, err := manager.LoadConfig("easy")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if config.Name != "Easy" {
			t.Errorf("Expected config name 'Easy', got '%s'", config.Name)
		}
		if config.MaxBattery != 20 {
			t.Errorf("Expected max battery 20, got %d", config.MaxBattery)
		}
	})

	t.Run("load with .json extension", func(t *testing.T) {
		config, err := manager.LoadConfig("easy.json")
		if err != nil {
			t.Fatalf("Failed to load config with extension: %v", err)
		}
		if config.Name != "Easy" {
			t.Errorf("Expected config name 'Easy', got '%s'", config.Name)
		}
	})

	t.Run("load from cache", func(t *testing.T) {
		// First load
		config1, _ := manager.LoadConfig("easy")

		// Second load should come from cache
		config2, err := manager.LoadConfig("easy")
		if err != nil {
			t.Fatalf("Failed to load config from cache: %v", err)
		}

		// Should be the same pointer (cached)
		if config1 != config2 {
			t.Error("Expected config to be loaded from cache")
		}
	})

	t.Run("load non-existent config", func(t *testing.T) {
		_, err := manager.LoadConfig("non-existent")
		if err != ErrConfigNotFound {
			t.Errorf("Expected ErrConfigNotFound, got %v", err)
		}
	})

	t.Run("load invalid config", func(t *testing.T) {
		// Write invalid config
		invalidData := []byte(`{"name": ""}`) // Missing required fields
		err := os.WriteFile(filepath.Join(dir, "invalid.json"), invalidData, 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		_, err = manager.LoadConfig("invalid")
		if err == nil {
			t.Error("Expected error for invalid config")
		}
	})

	t.Run("load malformed JSON", func(t *testing.T) {
		// Write malformed JSON
		malformedData := []byte(`{"name": "Malformed", invalid json}`)
		err := os.WriteFile(filepath.Join(dir, "malformed.json"), malformedData, 0644)
		if err != nil {
			t.Fatalf("Failed to write malformed config: %v", err)
		}

		_, err = manager.LoadConfig("malformed")
		if err == nil {
			t.Error("Expected error for malformed JSON")
		}
	})
}

func TestManager_GetDefault(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	defaultConfig := createValidConfig()
	defaultConfig.Name = "Default Config"
	writeConfigFile(t, dir, "default", defaultConfig)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	config := manager.GetDefault()
	if config == nil {
		t.Fatal("Expected default config to be non-nil")
	}
	if config.Name != "Default Config" {
		t.Errorf("Expected default config name 'Default Config', got '%s'", config.Name)
	}
}

func TestManager_ListConfigs(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	// Create multiple configs
	configs := []struct {
		filename string
		name     string
	}{
		{"default", "Default"},
		{"easy", "Easy"},
		{"medium", "Medium"},
		{"hard", "Hard"},
	}

	for _, cfg := range configs {
		config := createValidConfig()
		config.Name = cfg.name
		writeConfigFile(t, dir, cfg.filename, config)
	}

	// Also add a non-JSON file that should be ignored
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	configList, err := manager.ListConfigs()
	if err != nil {
		t.Fatalf("Failed to list configs: %v", err)
	}
	if len(configList) != 4 {
		t.Errorf("Expected 4 configs, got %d", len(configList))
	}

	// Verify all configs are listed
	foundConfigs := make(map[string]bool)
	for _, info := range configList {
		foundConfigs[info.Name] = true
	}

	for _, cfg := range configs {
		if !foundConfigs[cfg.name] {
			t.Errorf("Config '%s' not found in list", cfg.name)
		}
	}
}

func TestManager_ReloadConfig(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	// Create initial config
	config := createValidConfig()
	config.Name = "Changeable"
	config.MaxBattery = 10
	writeConfigFile(t, dir, "default", config)
	writeConfigFile(t, dir, "changeable", config)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Load config first time
	loaded, _ := manager.LoadConfig("changeable")
	if loaded.MaxBattery != 10 {
		t.Errorf("Expected initial max battery 10, got %d", loaded.MaxBattery)
	}

	// Modify config file
	config.MaxBattery = 20
	writeConfigFile(t, dir, "changeable", config)

	// Reload config
	err = manager.ReloadConfig("changeable")
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	// Verify updated value
	reloaded, _ := manager.LoadConfig("changeable")
	if reloaded.MaxBattery != 20 {
		t.Errorf("Expected reloaded max battery 20, got %d", reloaded.MaxBattery)
	}
}

func TestManager_ValidateConfig(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	defaultConfig := createValidConfig()
	writeConfigFile(t, dir, "default", defaultConfig)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("valid config", func(t *testing.T) {
		config := createValidConfig()
		err := manager.ValidateConfig(config)
		if err != nil {
			t.Errorf("Expected valid config to pass validation: %v", err)
		}
	})

	t.Run("invalid config - missing name", func(t *testing.T) {
		config := createValidConfig()
		config.Name = ""
		err := manager.ValidateConfig(config)
		if err == nil {
			t.Error("Expected error for config missing name")
		}
	})

	t.Run("invalid config - invalid grid size", func(t *testing.T) {
		config := createValidConfig()
		config.GridSize = 2 // Too small
		err := manager.ValidateConfig(config)
		if err == nil {
			t.Error("Expected error for invalid grid size")
		}
	})

	t.Run("invalid config - no parks", func(t *testing.T) {
		config := createValidConfig()
		config.Layout = []string{
			"BBBBB",
			"BRHBB",
			"BRRSB",
			"BRRRB",
			"BBBBB",
		}
		err := manager.ValidateConfig(config)
		if err == nil {
			t.Error("Expected error for config with no parks")
		}
	})
}

func TestManager_ConcurrentAccess(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	// Create configs
	defaultConfig := createValidConfig()
	writeConfigFile(t, dir, "default", defaultConfig)

	for i := 1; i <= 5; i++ {
		config := createValidConfig()
		config.Name = "Config" + string(rune('0'+i))
		writeConfigFile(t, dir, "config"+string(rune('0'+i)), config)
	}

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test concurrent loading
	var wg sync.WaitGroup
	errors := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			configName := "config" + string(rune('0'+((id%5)+1)))
			_, err := manager.LoadConfig(configName)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Unexpected error during concurrent access: %v", err)
	}

	// Verify cache size
	if manager.Count() < 5 {
		t.Errorf("Expected at least 5 configs in cache, got %d", manager.Count())
	}
}

func TestManager_CachingBehavior(t *testing.T) {
	dir := createTestConfigDir(t)
	defer os.RemoveAll(dir)

	defaultConfig := createValidConfig()
	writeConfigFile(t, dir, "default", defaultConfig)

	testConfig := createValidConfig()
	testConfig.Name = "Test"
	writeConfigFile(t, dir, "test", testConfig)

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Load config multiple times
	for i := 0; i < 10; i++ {
		config, err := manager.LoadConfig("test")
		if err != nil {
			t.Fatalf("Failed to load config on iteration %d: %v", i, err)
		}
		if config.Name != "Test" {
			t.Errorf("Unexpected config name on iteration %d", i)
		}
	}

	// Should have two entries in cache: the default config and the test config
	if manager.Count() != 2 { // Both "default" (or first available) and "test" are cached
		t.Errorf("Expected 2 configs in cache, got %d", manager.Count())
	}
}

// Add missing test-only methods to Manager

func (m *Manager) ReloadConfig(name string) error {
	m.mu.Lock()
	// Remove from cache to force reload
	delete(m.configs, name)
	m.mu.Unlock()

	// Load fresh from disk (without holding the lock)
	_, err := m.LoadConfig(name)
	return err
}

func (m *Manager) ValidateConfig(config *engine.GameConfig) error {
	return engine.ValidateGameConfig(config)
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.configs)
}
