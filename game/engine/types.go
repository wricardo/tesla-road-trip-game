package engine

// CellType represents different types of grid cells
type CellType string

const (
	Road         CellType = "road"
	Home         CellType = "home"
	Park         CellType = "park"
	Supercharger CellType = "supercharger"
	Water        CellType = "water"
	Building     CellType = "building"

	// Validation constants
	MinGridSize         = 5
	MaxGridSize         = 50
	MinBattery          = 1
	MaxBattery          = 100
	MaxBulkMoves        = 50
	UnreachableDistance = 999999
	WebSocketBufferSize = 256
)

// Cell represents a single grid cell
type Cell struct {
	Type              CellType `json:"type"`
	Visited           bool     `json:"visited,omitempty"`
	ID                string   `json:"id,omitempty"`
	AllowedDirections []string `json:"allowed_directions,omitempty"` // empty = unrestricted
}

// CellConfig defines properties for a custom layout char (used in GameConfig.CellConfigs)
type CellConfig struct {
	Type              string   `json:"type"`
	AllowedDirections []string `json:"allowed_directions,omitempty"` // empty = unrestricted
}

// Position represents x,y coordinates
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// GameMessages contains the built-in gameplay copy.
type GameMessages struct {
	Welcome            string
	HomeCharge         string
	SuperchargerCharge string
	ParkVisited        string
	ParkAlreadyVisited string
	Victory            string
	OutOfBattery       string
	Stranded           string
	CantMove           string
	BatteryStatus      string
	HitWall            string
}

// DefaultMessages is the canonical gameplay copy. Maps do not customize messages.
var DefaultMessages = GameMessages{
	Welcome:            "Welcome! Drive your Tesla to collect parks. Watch your battery!",
	HomeCharge:         "Home sweet home! Battery fully charged!",
	SuperchargerCharge: "Supercharger! Battery fully charged!",
	ParkVisited:        "Park visited! Score: %d",
	ParkAlreadyVisited: "Already visited this park",
	Victory:            "Victory! All %d parks visited!",
	OutOfBattery:       "Out of battery! Game Over!",
	Stranded:           "Stranded with no battery! Game Over!",
	CantMove:           "Can't move there!",
	BatteryStatus:      "Battery: %d/%d",
	HitWall:            "You crashed into a wall! Game Over!",
}

// GameConfig represents the game configuration from JSON
type GameConfig struct {
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	GridSize          int                   `json:"grid_size"`
	MaxBattery        int                   `json:"max_battery"`
	StartingBattery   int                   `json:"starting_battery"`
	Layout            []string              `json:"layout"`
	Legend            map[string]string     `json:"legend"`
	CellConfigs       map[string]CellConfig `json:"cell_configs,omitempty"` // extra chars with direction constraints
	WallCrashEndsGame bool                  `json:"wall_crash_ends_game"`
}

// SurroundingCell represents a cell with its absolute position
type SurroundingCell struct {
	X    int      `json:"x"`
	Y    int      `json:"y"`
	Type CellType `json:"type"`
}

// GameState represents the complete game state
type GameState struct {
	Grid         [][]Cell           `json:"grid"`
	PlayerPos    Position           `json:"player_pos"`
	Battery      int                `json:"battery"`
	MaxBattery   int                `json:"max_battery"`
	Score        int                `json:"score"`
	VisitedParks map[string]bool    `json:"visited_parks"`
	Message      string             `json:"message"`
	GameOver     bool               `json:"game_over"`
	Victory      bool               `json:"victory"`
	MapName      string             `json:"map_name"`
	MoveHistory  []MoveHistoryEntry `json:"move_history"`
	TotalMoves   int                `json:"total_moves"`
	LocalView    []SurroundingCell  `json:"local_view,omitempty"` // 8 surrounding cells

	// CurrentMoves tracks only the moves since the last reset. It mirrors MoveHistory entries
	// but gets cleared on reset while MoveHistory remains cumulative.
	CurrentMoves      []MoveHistoryEntry `json:"current_moves"`
	CurrentMovesCount int                `json:"current_moves_count"`

	// Computed helper views (not required for core game logic)
	LocalView3x3 []string `json:"local_view_3x3,omitempty"`
	BatteryRisk  string   `json:"battery_risk,omitempty"`
}

// MoveHistoryEntry represents a single move in the game history
type MoveHistoryEntry struct {
	Action       string   `json:"action"`
	FromPosition Position `json:"from_position"`
	ToPosition   Position `json:"to_position"`
	Battery      int      `json:"battery"`
	Timestamp    int64    `json:"timestamp"`
	Success      bool     `json:"success"`
	MoveNumber   int      `json:"move_number"`
}
