// Command analyze prints quick, human-readable heuristics about configuration
// files in the project's configs directory. It summarizes dimensions, battery
// settings, counts of chargers and parks, and highlights unreachable locations
// based on Manhattan distance vs. max battery.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// AnalysisConfig is a light struct for reading config files used by analysis.
type AnalysisConfig struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	GridSize          int               `json:"grid_size"`
	MaxBattery        int               `json:"max_battery"`
	StartingBattery   int               `json:"starting_battery"`
	Layout            []string          `json:"layout"`
	Legend            map[string]string `json:"legend"`
	WallCrashEndsGame bool              `json:"wall_crash_ends_game"`
	Messages          map[string]string `json:"messages"`
}

// AnalysisPoint denotes a grid coordinate used during analysis output.
type AnalysisPoint struct {
	X, Y int
}

func main() {
	configs := []string{
		"challenge.json",
		"classic.json",
		"easy.json",
		"easy_circuit.json",
		"easy_gardens.json",
		"hard_expedition.json",
		"medium_maze.json",
	}

	for _, configFile := range configs {
		fmt.Printf("\n=== Analyzing %s ===\n", configFile)
		analyzeConfig(filepath.Join("configs", configFile))
	}
}

func analyzeConfig(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var config AnalysisConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	fmt.Printf("Name: %s\n", config.Name)
	fmt.Printf("Grid Size: %d x %d\n", config.GridSize, len(config.Layout))
	fmt.Printf("Max Battery: %d\n", config.MaxBattery)
	fmt.Printf("Starting Battery: %d\n", config.StartingBattery)

	// Find all chargers (S and H)
	var chargers []AnalysisPoint
	var parks []AnalysisPoint
	var homePos AnalysisPoint
	foundHome := false

	for y, row := range config.Layout {
		for x, cell := range row {
			switch cell {
			case 'S':
				chargers = append(chargers, AnalysisPoint{x, y})
			case 'H':
				chargers = append(chargers, AnalysisPoint{x, y})
				if !foundHome {
					homePos = AnalysisPoint{x, y}
					foundHome = true
				}
			case 'P':
				parks = append(parks, AnalysisPoint{x, y})
			}
		}
	}

	fmt.Printf("Home Position: (%d, %d)\n", homePos.X, homePos.Y)
	fmt.Printf("Total Chargers (S+H): %d\n", len(chargers))
	fmt.Printf("Total Parks: %d\n", len(parks))

	// Check reachability from any position
	maxReachableDistance := config.MaxBattery
	unreachablePoints := []AnalysisPoint{}

	for y, row := range config.Layout {
		for x, cell := range row {
			if cell == 'R' || cell == 'P' || cell == 'S' || cell == 'H' {
				// This is a traversable cell
				minDistToCharger := 999999
				for _, charger := range chargers {
					dist := abs(x-charger.X) + abs(y-charger.Y)
					if dist < minDistToCharger {
						minDistToCharger = dist
					}
				}
				if minDistToCharger > maxReachableDistance {
					unreachablePoints = append(unreachablePoints, AnalysisPoint{x, y})
				}
			}
		}
	}

	if len(unreachablePoints) > 0 {
		fmt.Printf("⚠️  WARNING: %d points are unreachable from any charger!\n", len(unreachablePoints))
		fmt.Printf("   Max battery: %d, but some points are further than this from all chargers\n", config.MaxBattery)
		for i, p := range unreachablePoints {
			if i < 5 { // Show first 5 unreachable points
				fmt.Printf("   Unreachable: (%d, %d) - '%c'\n", p.X, p.Y, config.Layout[p.Y][p.X])
			}
		}
		if len(unreachablePoints) > 5 {
			fmt.Printf("   ... and %d more\n", len(unreachablePoints)-5)
		}
	} else {
		fmt.Printf("✅ All traversable points are within reach of at least one charger\n")
	}

	// Check if all parks are reachable
	unreachableParks := []AnalysisPoint{}
	for _, park := range parks {
		minDistToCharger := 999999
		for _, charger := range chargers {
			dist := abs(park.X-charger.X) + abs(park.Y-charger.Y)
			if dist < minDistToCharger {
				minDistToCharger = dist
			}
		}
		if minDistToCharger > maxReachableDistance {
			unreachableParks = append(unreachableParks, park)
		}
	}

	if len(unreachableParks) > 0 {
		fmt.Printf("⚠️  CRITICAL: %d parks are unreachable from any charger!\n", len(unreachableParks))
		for _, p := range unreachableParks {
			fmt.Printf("   Unreachable Park: (%d, %d)\n", p.X, p.Y)
		}
	} else {
		fmt.Printf("✅ All parks are within reach of at least one charger\n")
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
