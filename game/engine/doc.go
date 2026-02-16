// Package engine provides the core game logic for the Tesla Road Trip Game.
//
// The engine package implements the game mechanics including:
//   - Grid-based movement and collision detection
//   - Battery management and charging mechanics
//   - Park collection and victory conditions
//   - Game state management and persistence
//   - Configuration loading and validation
//
// Core Types:
//
// The Engine interface defines the main contract for game operations,
// implemented by GameEngine. GameState represents the current game state,
// while GameConfig defines the game rules and layout loaded from JSON files.
//
// Usage:
//
//	config, err := engine.LoadConfigByName("easy")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	gameEngine, err := engine.NewEngine(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Move the player
//	success := gameEngine.Move("up")
//	state := gameEngine.GetState()
//
// Game Rules:
//
// Players control a Tesla vehicle on a grid, collecting parks while managing
// battery. Movement costs battery, which can be recharged at home tiles or
// superchargers. The game ends in victory when all parks are collected, or
// in defeat when the battery is depleted without reaching a charger.
package engine
