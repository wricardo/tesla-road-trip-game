package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateGameConfig validates a game configuration for correctness and
// playability. It includes a winnability proof (an exhaustive search over
// position × battery × park-subset states), which is expensive: call it when a
// map is authored, not when a saved session is reloaded.
func ValidateGameConfig(config *GameConfig) error {
	if err := ValidateGameConfigStructure(config); err != nil {
		return err
	}
	return validateWinnable(config)
}

// ValidateGameConfigStructure performs every check ValidateGameConfig does
// except the winnability search: required fields, grid/battery bounds, row
// widths, legal characters, the legend and the presence of a home and a park.
// It is cheap (linear in the grid) and is what session rehydration needs, since
// a map that was accepted when created does not need to be re-proven on load.
func ValidateGameConfigStructure(config *GameConfig) error {
	// Validate required fields
	if config.Name == "" {
		return fmt.Errorf("config validation: name is required")
	}
	// Description is optional; blank descriptions are allowed for custom maps.

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
				if _, ok := config.CellConfigs[string(char)]; ok {
					continue
				}
				return fmt.Errorf("config validation: invalid character '%c' at row %d, col %d", char, i+1, j+1)
			}
		}
	}

	// Validate CellConfigs
	validDirections := map[string]bool{"north": true, "south": true, "east": true, "west": true}
	for char, cc := range config.CellConfigs {
		if len(char) != 1 {
			return fmt.Errorf("config validation: cell_configs key must be single character, got %q", char)
		}
		if cc.Type != string(Road) {
			return fmt.Errorf("config validation: cell_configs[%q] type must be %q, got %q", char, Road, cc.Type)
		}
		for _, d := range cc.AllowedDirections {
			if !validDirections[d] {
				return fmt.Errorf("config validation: cell_configs[%q] invalid direction %q (must be north/south/east/west)", char, d)
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

	return nil
}

func validateWinnable(config *GameConfig) error {
	type point struct{ x, y int }
	type state struct {
		x, y    int
		battery int
		parks   uint64
	}

	var home *point
	parks := map[point]int{}
	for y, row := range config.Layout {
		for x, cell := range row {
			switch cell {
			case 'H':
				if home == nil {
					home = &point{x: x, y: y}
				}
			case 'P':
				if len(parks) >= 63 {
					return fmt.Errorf("config validation: too many parks to validate winnability: %d", len(parks)+1)
				}
				parks[point{x: x, y: y}] = len(parks)
			}
		}
	}
	if home == nil {
		return fmt.Errorf("config validation: layout must contain at least one home (H) cell")
	}

	// dirName maps (dx,dy) → direction string for AllowedDirections checks
	type dirEntry struct {
		dx, dy int
		name   string
	}
	dirList := []dirEntry{
		{0, -1, "north"},
		{0, 1, "south"},
		{-1, 0, "west"},
		{1, 0, "east"},
	}

	// cellAllowed checks direction constraints from CellConfigs for a raw layout char
	cellAllowed := func(ch rune, dirName string) bool {
		cc, ok := config.CellConfigs[string(ch)]
		if !ok || len(cc.AllowedDirections) == 0 {
			return true
		}
		for _, d := range cc.AllowedDirections {
			if d == dirName {
				return true
			}
		}
		return false
	}

	// The search space is positions × battery levels × park subsets. Encode each
	// state as an integer and track visited states in a bitset rather than a map:
	// a map keyed by a 24-byte struct costs ~50 bytes per state, so a 20M-state
	// map allocated over a gigabyte and got the server OOM-killed. A bitset needs
	// one bit per state, and the size is known up front so an oversized map is
	// rejected before anything is allocated.
	masks := uint64(1) << len(parks)
	total := uint64(config.GridSize) * uint64(config.GridSize) * uint64(config.MaxBattery+1) * masks
	const maxStates = 24_000_000 // ~3 MB bitset, ~192 MB queue in the worst case
	if total > maxStates {
		return fmt.Errorf(
			"config validation: map %q is too complex to verify winnability "+
				"(%d parks, grid %d, max battery %d); reduce the parks or the battery",
			config.Name, len(parks), config.GridSize, config.MaxBattery)
	}

	stride := uint64(config.MaxBattery+1) * masks
	encode := func(x, y, battery int, parkMask uint64) uint64 {
		return (uint64(y*config.GridSize+x))*stride + uint64(battery)*masks + parkMask
	}
	// The queue holds encoded indices rather than structs: 8 bytes per state
	// instead of 32, which is what made the map-based version allocate over a
	// gigabyte for a 900-cell map.
	decode := func(i uint64) (x, y, battery int, parkMask uint64) {
		parkMask = i % masks
		rest := i / masks
		battery = int(rest % uint64(config.MaxBattery+1))
		cell := rest / uint64(config.MaxBattery+1)
		return int(cell % uint64(config.GridSize)), int(cell / uint64(config.GridSize)), battery, parkMask
	}

	visited := make([]uint64, (total+63)/64)
	isSet := func(i uint64) bool { return visited[i>>6]&(1<<(i&63)) != 0 }
	set := func(i uint64) { visited[i>>6] |= 1 << (i & 63) }

	allParks := masks - 1
	startIdx := encode(home.x, home.y, config.StartingBattery, 0)
	// Preallocate modestly: reserving the whole theoretical space up front would
	// mean hundreds of MB for maps that explore far less.
	queue := make([]uint64, 0, 4096)
	queue = append(queue, startIdx)
	set(startIdx)

	for head := 0; head < len(queue); head++ {
		curX, curY, curBattery, curParks := decode(queue[head])
		if curParks == allParks {
			return nil
		}
		if curBattery <= 0 {
			continue
		}

		currentChar := rune(config.Layout[curY][curX])

		for _, dir := range dirList {
			// exit constraint: current cell must allow this direction
			if !cellAllowed(currentChar, dir.name) {
				continue
			}

			nx, ny := curX+dir.dx, curY+dir.dy
			if nx < 0 || ny < 0 || nx >= config.GridSize || ny >= config.GridSize {
				continue
			}

			cell := rune(config.Layout[ny][nx])
			if cell == 'B' || cell == 'W' {
				continue
			}

			// entry constraint: destination cell must allow this direction
			if !cellAllowed(cell, dir.name) {
				continue
			}

			nextBattery := curBattery - 1
			nextParks := curParks
			if cell == 'H' || cell == 'S' {
				nextBattery = config.MaxBattery
			}
			if idx, ok := parks[point{x: nx, y: ny}]; ok {
				nextParks |= 1 << idx
				if nextParks == allParks {
					return nil // collected the last park; the walk is monotonic
				}
			}

			nextIdx := encode(nx, ny, nextBattery, nextParks)
			if isSet(nextIdx) {
				continue
			}
			set(nextIdx)
			queue = append(queue, nextIdx)
		}
	}

	return fmt.Errorf("config validation: parks are unreachable with max_battery %d and starting_battery %d", config.MaxBattery, config.StartingBattery)
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

// LoadConfigByName loads a game configuration by name from the maps directory
func LoadConfigByName(configName string) (*GameConfig, error) {
	// Add .json extension if not present
	if !strings.HasSuffix(configName, ".json") {
		configName = configName + ".json"
	}

	configPath := filepath.Join("maps", configName)

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
			Name:            "default",
			Description:     "Default configuration",
			GridSize:        15,
			MaxBattery:      10,
			StartingBattery: 10,
			Legend:          map[string]string{"R": "road", "H": "home", "P": "park", "S": "supercharger", "W": "water", "B": "building"},
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
				ch := config.Layout[y][x]
				switch ch {
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
				default:
					if cc, ok := config.CellConfigs[string(ch)]; ok {
						cell := Cell{Type: CellType(cc.Type)}
						if len(cc.AllowedDirections) > 0 {
							dirs := make([]string, len(cc.AllowedDirections))
							copy(dirs, cc.AllowedDirections)
							cell.AllowedDirections = dirs
						}
						grid[y][x] = cell
					}
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
		Message:           DefaultMessages.Welcome,
		GameOver:          false,
		Victory:           false,
		MapName:           config.Name,
		MoveHistory:       []MoveHistoryEntry{},
		TotalMoves:        0,
		ResetCount:        0,
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
