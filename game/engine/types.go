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
	Type    CellType `json:"type"`
	Visited bool     `json:"visited,omitempty"` // For parks
	ID      string   `json:"id,omitempty"`      // Unique ID for parks
}

// Position represents x,y coordinates
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// GameConfig represents the game configuration from JSON
type GameConfig struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	GridSize          int               `json:"grid_size"`
	MaxBattery        int               `json:"max_battery"`
	StartingBattery   int               `json:"starting_battery"`
	Layout            []string          `json:"layout"`
	Legend            map[string]string `json:"legend"`
	WallCrashEndsGame bool              `json:"wall_crash_ends_game"`
	Messages          struct {
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
	} `json:"messages"`
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
	ConfigName   string             `json:"config_name"`
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
