package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig_ValidConfig(t *testing.T) {
	// Create a valid test config
	validConfig := `{
		"name": "Test Config",
		"description": "Test configuration",
		"grid_size": 5,
		"layout": [
			"BBBBB",
			"BRHPB",
			"BRRRB",
			"BPPPB",
			"BBBBB"
		],
		"max_battery": 10,
		"starting_battery": 8,
		"messages": {
			"welcome": "Welcome!",
			"park_visited": "Park visited!",
			"victory": "Victory!",
			"out_of_battery": "Out of battery!",
			"supercharger_charge": "Charged!",
			"home_charge": "Home charged!",
			"battery_status": "Battery: %d/%d",
			"cant_move": "Can't move!"
		},
		"wall_crash_ends_game": false,
		"legend": {
			"R": "road",
			"H": "home",
			"P": "park",
			"S": "supercharger",
			"W": "water",
			"B": "building"
		}
	}`

	// Write to temp file
	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(validConfig)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if !result.Valid {
		t.Errorf("Expected valid config, but got errors: %v", result.Errors)
	}

	if result.File != filepath.Base(tmpfile.Name()) {
		t.Errorf("Expected file name %s, got %s", filepath.Base(tmpfile.Name()), result.File)
	}
}

func TestValidateConfig_InvalidJSON(t *testing.T) {
	// Create invalid JSON
	invalidJSON := `{"name": "test", invalid json}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(invalidJSON))
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if result.Valid {
		t.Error("Expected invalid config due to bad JSON")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Invalid JSON") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Invalid JSON' error")
	}
}

func TestValidateConfig_MissingFile(t *testing.T) {
	result := validateConfig("/non/existent/file.json")
	if result.Valid {
		t.Error("Expected invalid result for missing file")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Failed to read file") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Failed to read file' error")
	}
}

func TestValidateConfig_EmptyLayout(t *testing.T) {
	config := `{
		"name": "Test",
		"description": "Test",
		"grid_size": 5,
		"layout": [],
		"max_battery": 10,
		"starting_battery": 8,
		"messages": {
			"welcome": "Welcome!",
			"park_visited": "Park visited!",
			"victory": "Victory!",
			"out_of_battery": "Out of battery!",
			"supercharger_charge": "Charged!",
			"home_charge": "Home charged!",
			"battery_status": "Battery: %d/%d",
			"cant_move": "Can't move!"
		},
		"wall_crash_ends_game": false,
		"legend": {}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(config))
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if result.Valid {
		t.Error("Expected invalid config due to empty layout")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Layout is empty") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Layout is empty' error")
	}
}

func TestValidateConfig_NoHome(t *testing.T) {
	config := `{
		"name": "Test",
		"description": "Test",
		"grid_size": 3,
		"layout": [
			"RRR",
			"RPR",
			"RRR"
		],
		"max_battery": 10,
		"starting_battery": 8,
		"messages": {
			"welcome": "Welcome!",
			"park_visited": "Park visited!",
			"victory": "Victory!",
			"out_of_battery": "Out of battery!",
			"supercharger_charge": "Charged!",
			"home_charge": "Home charged!",
			"battery_status": "Battery: %d/%d",
			"cant_move": "Can't move!"
		},
		"wall_crash_ends_game": false,
		"legend": {}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(config))
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if result.Valid {
		t.Error("Expected invalid config due to no home")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Must have at least 1 home") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Must have at least 1 home' error")
	}
}

func TestValidateConfig_NoParks(t *testing.T) {
	config := `{
		"name": "Test",
		"description": "Test",
		"grid_size": 3,
		"layout": [
			"RRR",
			"RHR",
			"RRR"
		],
		"max_battery": 10,
		"starting_battery": 8,
		"messages": {
			"welcome": "Welcome!",
			"park_visited": "Park visited!",
			"victory": "Victory!",
			"out_of_battery": "Out of battery!",
			"supercharger_charge": "Charged!",
			"home_charge": "Home charged!",
			"battery_status": "Battery: %d/%d",
			"cant_move": "Can't move!"
		},
		"wall_crash_ends_game": false,
		"legend": {}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(config))
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if result.Valid {
		t.Error("Expected invalid config due to no parks")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Must have at least 1 park") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Must have at least 1 park' error")
	}
}

func TestValidateConfig_InvalidBattery(t *testing.T) {
	config := `{
		"name": "Test",
		"description": "Test",
		"grid_size": 3,
		"layout": [
			"RRR",
			"RHP",
			"RRR"
		],
		"max_battery": -5,
		"starting_battery": 10,
		"messages": {
			"welcome": "Welcome!",
			"park_visited": "Park visited!",
			"victory": "Victory!",
			"out_of_battery": "Out of battery!",
			"supercharger_charge": "Charged!",
			"home_charge": "Home charged!",
			"battery_status": "Battery: %d/%d",
			"cant_move": "Can't move!"
		},
		"wall_crash_ends_game": false,
		"legend": {}
	}`

	tmpfile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(config))
	tmpfile.Close()

	result := validateConfig(tmpfile.Name())
	if result.Valid {
		t.Error("Expected invalid config due to invalid battery settings")
	}

	foundMaxBattery := false
	foundStartingBattery := false
	for _, err := range result.Errors {
		if contains(err, "max_battery must be positive") {
			foundMaxBattery = true
		}
		if contains(err, "starting_battery") && contains(err, "cannot exceed") {
			foundStartingBattery = true
		}
	}
	if !foundMaxBattery {
		t.Error("Expected 'max_battery must be positive' error")
	}
	if !foundStartingBattery {
		t.Error("Expected 'starting_battery cannot exceed max_battery' error")
	}
}

func TestValidateConnectivity_ValidLayout(t *testing.T) {
	layout := []string{
		"BBBBB",
		"BRHPB",
		"BRRRB",
		"BPPPB",
		"BBBBB",
	}

	result := validateConnectivity(layout, 1, 4)
	if !result.Valid {
		t.Errorf("Expected valid connectivity, but got errors: %v", result.Errors)
	}
}

func TestValidateConnectivity_UnreachablePark(t *testing.T) {
	layout := []string{
		"BBBBB",
		"BRHBB",
		"BRWBB",
		"BBPBB",
		"BBBBB",
	}

	result := validateConnectivity(layout, 1, 1)
	if result.Valid {
		t.Error("Expected invalid connectivity due to unreachable park")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Connectivity failure") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Connectivity failure' error")
	}
}

func TestValidateConnectivity_EmptyLayout(t *testing.T) {
	result := validateConnectivity([]string{}, 0, 0)
	if result.Valid {
		t.Error("Expected invalid result for empty layout")
	}

	found := false
	for _, err := range result.Errors {
		if contains(err, "Cannot validate connectivity: empty layout") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Cannot validate connectivity: empty layout' error")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
