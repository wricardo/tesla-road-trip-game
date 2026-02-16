package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateGameConfig validates a game configuration for correctness and playability
func ValidateGameConfig(config *GameConfig) error {
	// Validate required fields
	if config.Name == "" {
		return fmt.Errorf("config validation: name is required")
	}
	if config.Description == "" {
		return fmt.Errorf("config validation: description is required")
	}

	// Validate grid size
	if config.GridSize < MinGridSize || config.GridSize > MaxGridSize {
		return fmt.Errorf("config validation: grid_size must be between %d and %d, got %d", MinGridSize, MaxGridSize, config.GridSize)
	}

	// Validate battery settings
	if config.MaxBattery < MinBattery || config.MaxBattery > MaxBattery {
		return fmt.Errorf("config validation: max_battery must be between %d and %d, got %d", MinBattery, MaxBattery, config.MaxBattery)
	}
	if config.StartingBattery < MinBattery || config.StartingBattery > config.MaxBattery {
		return fmt.Errorf("config validation: starting_battery must be between %d and max_battery (%d), got %d",
			MinBattery, config.MaxBattery, config.StartingBattery)
	}

	// Validate layout
	if len(config.Layout) != config.GridSize {
		return fmt.Errorf("config validation: layout must have %d rows to match grid_size, got %d",
			config.GridSize, len(config.Layout))
	}

	hasHome := false
	parkCount := 0
	for i, row := range config.Layout {
		if len(row) != config.GridSize {
			return fmt.Errorf("config validation: row %d must have %d characters to match grid_size, got %d",
				i+1, config.GridSize, len(row))
		}

		// Validate characters and count important cells
		for j, char := range row {
			switch char {
			case 'R', 'S', 'W', 'B': // Valid characters
			case 'H':
				hasHome = true
			case 'P':
				parkCount++
			default:
				return fmt.Errorf("config validation: invalid character '%c' at row %d, col %d", char, i+1, j+1)
			}
		}
	}

	if !hasHome {
		return fmt.Errorf("config validation: layout must contain at least one home (H) cell")
	}
	if parkCount == 0 {
		return fmt.Errorf("config validation: layout must contain at least one park (P) cell")
	}

	// Validate legend
	requiredLegend := map[string]string{
		"R": "road",
		"H": "home",
		"P": "park",
		"S": "supercharger",
		"W": "water",
		"B": "building",
	}
	for key, expectedValue := range requiredLegend {
		if value, ok := config.Legend[key]; !ok || value != expectedValue {
			return fmt.Errorf("config validation: legend['%s'] must be '%s', got '%s'", key, expectedValue, value)
		}
	}

	// Validate messages
	if config.Messages.Welcome == "" {
		return fmt.Errorf("config validation: messages.welcome is required")
	}
	if config.Messages.Victory == "" {
		return fmt.Errorf("config validation: messages.victory is required")
	}
	if config.Messages.OutOfBattery == "" {
		return fmt.Errorf("config validation: messages.out_of_battery is required")
	}

	// Validate wall crash message if feature is enabled
	if config.WallCrashEndsGame && config.Messages.HitWall == "" {
		return fmt.Errorf("config validation: messages.hit_wall is required when wall_crash_ends_game is true")
	}

	// Validate format strings
	if !strings.Contains(config.Messages.ParkVisited, "%d") {
		return fmt.Errorf("config validation: messages.park_visited must contain %%d for score")
	}
	if !strings.Contains(config.Messages.Victory, "%d") {
		return fmt.Errorf("config validation: messages.victory must contain %%d for park count")
	}
	if config.Messages.BatteryStatus != "" && !strings.Contains(config.Messages.BatteryStatus, "%d") {
		return fmt.Errorf("config validation: messages.battery_status must contain %%d for battery values")
	}

	// Validate winnability - check that all parks are reachable from chargers
	type Point struct {
		X, Y int
	}

	var chargers []Point
	var parks []Point

	// Find all chargers (S and H) and parks
	for y, row := range config.Layout {
		for x, cell := range row {
			switch cell {
			case 'S', 'H':
				chargers = append(chargers, Point{x, y})
			case 'P':
				parks = append(parks, Point{x, y})
			}
		}
	}

	// Check if all parks are reachable from at least one charger
	for _, park := range parks {
		minDistToCharger := UnreachableDistance
		for _, charger := range chargers {
			// Manhattan distance
			dist := abs(park.X-charger.X) + abs(park.Y-charger.Y)
			if dist < minDistToCharger {
				minDistToCharger = dist
			}
		}
		if minDistToCharger > config.MaxBattery {
			return fmt.Errorf("config validation: park at (%d, %d) is unreachable - nearest charger is %d moves away but max battery is %d",
				park.X+1, park.Y+1, minDistToCharger, config.MaxBattery)
		}
	}

	return nil
}

// LoadGameConfig loads a game configuration from a JSON file
func LoadGameConfig(filename string) (*GameConfig, error) {
	// Support CONFIG_DIR environment variable for alternative config directory
	configPath := filename
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		// If filename starts with "configs/", replace with CONFIG_DIR
		if strings.HasPrefix(filename, "configs/") {
			configPath = filepath.Join(configDir, strings.TrimPrefix(filename, "configs/"))
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate the loaded configuration
	if err := ValidateGameConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfigByName loads a game configuration by name from the configs directory
func LoadConfigByName(configName string) (*GameConfig, error) {
	// Add .json extension if not present
	if !strings.HasSuffix(configName, ".json") {
		configName = configName + ".json"
	}

	configPath := filepath.Join("configs", configName)

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file '%s' not found", configName)
	}

	// Load and parse the config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %v", configName, err)
	}

	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %v", configName, err)
	}

	// Validate the config
	if err := ValidateGameConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config '%s': %v", configName, err)
	}

	return &config, nil
}

// InitGameStateFromConfig creates a new game state using the provided configuration
func InitGameStateFromConfig(config *GameConfig) *GameState {
	if config == nil {
		// Use default config if not provided
		config = &GameConfig{
			GridSize:        15,
			MaxBattery:      10,
			StartingBattery: 10,
			Layout: []string{
				"BBBWBBBPBBBWBBB",
				"BRRRRRRRRRRRRRB",
				"BRBBBRRSRBBBRPB",
				"BRBPBRRRRRBPBRB",
				"BRBRBBBRBBBRBBB",
				"BRRRRRRRRRRRRRB",
				"BBBBRWWWWWBBBBB",
				"PRRRRHHHHHRRRRP",
				"BBBBRWWWWWBBBBB",
				"BRRRRRRRRRRRRRB",
				"BRBRBBBRBBBRBBB",
				"BRBPBRRRRRBPBRB",
				"BRBBBRRSRBBBRPB",
				"BRRRRRRRRRRRRRB",
				"BBBWBBBPBBBWBBB",
			},
		}
		config.Messages.Welcome = "Welcome! Drive your Tesla to collect parks. Watch your battery!"
		config.Messages.HomeCharge = "Home sweet home! Battery fully charged!"
		config.Messages.SuperchargerCharge = "Supercharger! Battery fully charged!"
		config.Messages.ParkVisited = "Park visited! Score: %d"
		config.Messages.ParkAlreadyVisited = "Already visited this park"
		config.Messages.Victory = "Victory! All %d parks visited!"
		config.Messages.OutOfBattery = "Out of battery! Game Over!"
		config.Messages.Stranded = "Stranded with no battery! Game Over!"
		config.Messages.CantMove = "Can't move there!"
		config.Messages.BatteryStatus = "Battery: %d/%d"
	}

	// Create grid based on config
	gridSize := config.GridSize
	grid := make([][]Cell, gridSize)
	for i := range grid {
		grid[i] = make([]Cell, gridSize)
	}

	parkCount := 0
	var homePos Position

	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if y < len(config.Layout) && x < len(config.Layout[y]) {
				switch config.Layout[y][x] {
				case 'R':
					grid[y][x] = Cell{Type: Road}
				case 'H':
					grid[y][x] = Cell{Type: Home}
					homePos = Position{X: x, Y: y}
				case 'P':
					parkID := fmt.Sprintf("park_%d", parkCount)
					grid[y][x] = Cell{Type: Park, ID: parkID}
					parkCount++
				case 'S':
					grid[y][x] = Cell{Type: Supercharger}
				case 'W':
					grid[y][x] = Cell{Type: Water}
				case 'B':
					grid[y][x] = Cell{Type: Building}
				}
			}
		}
	}

	return &GameState{
		Grid:              grid,
		PlayerPos:         homePos,
		Battery:           config.StartingBattery,
		MaxBattery:        config.MaxBattery,
		Score:             0,
		VisitedParks:      make(map[string]bool),
		Message:           config.Messages.Welcome,
		GameOver:          false,
		Victory:           false,
		ConfigName:        config.Name,
		MoveHistory:       []MoveHistoryEntry{},
		TotalMoves:        0,
		CurrentMoves:      []MoveHistoryEntry{},
		CurrentMovesCount: 0,
	}
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
