// Command validate provides a small CLI that validates game configuration JSON
// files in the ../configs directory. It checks:
//   - JSON structure and required fields
//   - Grid consistency and allowed characters (R, H, P, S, W, B)
//   - Presence of at least one home (H) and one park (P)
//   - Battery constraints (starting <= max and both positive)
//   - Required message keys
//   - Connectivity: all parks are reachable from at least one home via passable cells
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config mirrors the JSON schema for a game configuration.
type Config struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	GridSize        int               `json:"grid_size"`
	Layout          []string          `json:"layout"`
	MaxBattery      int               `json:"max_battery"`
	StartingBattery int               `json:"starting_battery"`
	Messages        map[string]string `json:"messages"`
	WallCrashEnds   bool              `json:"wall_crash_ends_game"`
	Legend          map[string]string `json:"legend"`
}

// ValidationResult captures the outcome of validating a single file.
// If Valid is true, Errors contains informational messages; otherwise it
// accumulates the validation errors that were found.
type ValidationResult struct {
	File   string
	Valid  bool
	Errors []string
}

// validateConfig loads and validates a single configuration JSON file.
// It performs structural checks, grid/legend validation, message presence,
// and reachability analysis for parks.
func validateConfig(filePath string) ValidationResult {
	result := ValidationResult{
		File:   filepath.Base(filePath),
		Valid:  true,
		Errors: []string{},
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read file: %v", err))
		return result
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
		return result
	}

	// Validate grid
	if len(config.Layout) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Layout is empty")
	}

	gridWidth := -1
	homeCount := 0
	parkCount := 0
	superchargerCount := 0
	validChars := map[rune]bool{
		'R': true, // Road
		'H': true, // Home
		'P': true, // Park
		'S': true, // Supercharger
		'W': true, // Water
		'B': true, // Building
	}

	for i, row := range config.Layout {
		if gridWidth == -1 {
			gridWidth = len(row)
		} else if len(row) != gridWidth {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Inconsistent grid width at row %d: expected %d, got %d", i+1, gridWidth, len(row)))
		}

		for j, char := range row {
			if !validChars[char] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Invalid character '%c' at position [%d,%d]", char, i+1, j+1))
			}
			switch char {
			case 'H':
				homeCount++
			case 'P':
				parkCount++
			case 'S':
				superchargerCount++
			}
		}
	}

	// Validate game elements
	if homeCount == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Must have at least 1 home (H) cell")
	}

	if parkCount == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Must have at least 1 park (P)")
	}

	// Validate battery settings
	if config.MaxBattery <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("max_battery must be positive, got %d", config.MaxBattery))
	}

	if config.StartingBattery <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("starting_battery must be positive, got %d", config.StartingBattery))
	}

	if config.StartingBattery > config.MaxBattery {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("starting_battery (%d) cannot exceed max_battery (%d)", config.StartingBattery, config.MaxBattery))
	}

	// Validate messages
	requiredMessages := []string{
		"welcome",
		"park_visited",
		"victory",
		"out_of_battery",
		"supercharger_charge",
		"home_charge",
		"battery_status",
		"cant_move",
	}
	for _, msg := range requiredMessages {
		if _, exists := config.Messages[msg]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required message: %s", msg))
		}
	}

	// Connectivity validation - check if all parks are reachable from homes
	if result.Valid {
		reachabilityResult := validateConnectivity(config.Layout, homeCount, parkCount)
		if !reachabilityResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, reachabilityResult.Errors...)
		} else {
			result.Errors = append(result.Errors, reachabilityResult.Errors...)
		}
	}

	// Add informational data
	if result.Valid {
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Name: %s", config.Name))
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Grid: %dx%d", len(config.Layout), gridWidth))
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Home cells: %d", homeCount))
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Parks: %d", parkCount))
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Superchargers: %d", superchargerCount))
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Battery: %d/%d", config.StartingBattery, config.MaxBattery))
	}

	return result
}

// validateConnectivity ensures all parks are reachable from a home using
// 4-directional movement over passable cells (R, H, P, S). It reports any
// unreachable parks and returns an aggregated ValidationResult.
func validateConnectivity(layout []string, homeCount, parkCount int) ValidationResult {
	result := ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	if len(layout) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Cannot validate connectivity: empty layout")
		return result
	}

	height := len(layout)
	width := len(layout[0])

	// Find all homes, parks, and passable cells
	var homes [][]int
	var parks [][]int

	for y := 0; y < height; y++ {
		for x := 0; x < width && x < len(layout[y]); x++ {
			cell := rune(layout[y][x])
			switch cell {
			case 'H':
				homes = append(homes, []int{x, y})
			case 'P':
				parks = append(parks, []int{x, y})
			}
		}
	}

	if len(homes) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "No home positions found for connectivity test")
		return result
	}

	if len(parks) == 0 {
		// Already validated elsewhere, but just in case
		result.Valid = false
		result.Errors = append(result.Errors, "No parks found for connectivity test")
		return result
	}

	// Flood fill from first home to find all reachable cells
	visited := make(map[string]bool)
	queue := [][]int{homes[0]}

	// Helper function to check if a cell is passable
	isPassable := func(x, y int) bool {
		if x < 0 || y < 0 || y >= height || x >= width || x >= len(layout[y]) {
			return false
		}
		cell := rune(layout[y][x])
		return cell == 'R' || cell == 'H' || cell == 'P' || cell == 'S'
	}

	// Flood fill algorithm
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		x, y := current[0], current[1]
		key := fmt.Sprintf("%d,%d", x, y)

		if visited[key] {
			continue
		}
		visited[key] = true

		// Check all 4 directions
		directions := [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
		for _, dir := range directions {
			nx, ny := x+dir[0], y+dir[1]
			nkey := fmt.Sprintf("%d,%d", nx, ny)

			if !visited[nkey] && isPassable(nx, ny) {
				queue = append(queue, []int{nx, ny})
			}
		}
	}

	// Check if all parks are reachable
	unreachableParks := []string{}
	for _, park := range parks {
		px, py := park[0], park[1]
		key := fmt.Sprintf("%d,%d", px, py)
		if !visited[key] {
			unreachableParks = append(unreachableParks, fmt.Sprintf("Park at (%d,%d)", px, py))
		}
	}

	if len(unreachableParks) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Connectivity failure: %d/%d parks unreachable from home", len(unreachableParks), len(parks)))
		for _, park := range unreachableParks {
			result.Errors = append(result.Errors, fmt.Sprintf("Unreachable: %s", park))
		}
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("✓ Connectivity: All %d parks reachable from home", len(parks)))
	}

	return result
}

// main scans ../configs for *.json files and validates each one, printing a
// concise report and exiting with non-zero status if any are invalid.
func main() {
	configDir := "../configs"
	files, err := filepath.Glob(filepath.Join(configDir, "*.json"))
	if err != nil {
		fmt.Printf("Error finding config files: %v\n", err)
		os.Exit(1)
	}

	allValid := true
	for _, file := range files {
		result := validateConfig(file)

		fmt.Printf("\n%s %s\n", strings.Repeat("=", 20), result.File)

		if result.Valid {
			fmt.Println("✅ VALID")
			for _, info := range result.Errors {
				fmt.Println("  " + info)
			}
		} else {
			fmt.Println("❌ INVALID")
			allValid = false
			for _, err := range result.Errors {
				if !strings.HasPrefix(err, "✓") {
					fmt.Println("  ❌ " + err)
				}
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 40))
	if allValid {
		fmt.Println("✅ All configurations are valid!")
	} else {
		fmt.Println("❌ Some configurations have errors")
		os.Exit(1)
	}
}
