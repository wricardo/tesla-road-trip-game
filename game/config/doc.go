// Package config provides configuration management for the Tesla Road Trip Game.
//
// The config package handles:
//   - Loading game configurations from JSON files
//   - Configuration validation and verification
//   - Default configuration management
//   - Configuration discovery and listing
//
// Configuration Format:
//
// Game configurations are stored as JSON files in the configs directory.
// Each configuration defines:
//   - Grid layout using character mapping (R=road, H=home, P=park, etc.)
//   - Battery parameters (max capacity, starting amount)
//   - Game messages for various events
//   - Victory and game-over conditions
//
// Available Configurations:
//
// The package supports multiple difficulty levels and maze types:
//   - classic: Original 10x10 grid with balanced challenge
//   - easy: Smaller grid with more charging stations
//   - medium_maze: Complex maze requiring strategic navigation
//   - challenge: Large grid with limited charging infrastructure
//
// Usage:
//
//	manager := config.NewManager("configs")
//
//	// Load specific configuration
//	gameConfig, err := manager.LoadConfig("easy")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get default configuration
//	defaultConfig := manager.GetDefault()
//
//	// List available configurations
//	configs, err := manager.ListConfigs()
//
// Validation:
//
// All configurations are validated for:
//   - Proper grid dimensions and layout
//   - Valid cell types and legend mappings
//   - Reachable parks and charging stations
//   - Required message templates
//   - Battery parameter constraints
package config
