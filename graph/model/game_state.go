package model

// GridAccess records whether a grid/layout is hidden behind fog-of-war and the
// password required to reveal it. It is carried on the model instance itself so
// resolvers can enforce it without any global registry.
type GridAccess struct {
	FogEnabled   bool
	GridPassword string
}

// GameState is bound in gqlgen.yml instead of being generated so it can carry
// the unexported grid access policy.
type GameState struct {
	Grid              [][]*Cell           `json:"grid"`
	PlayerPos         *Position           `json:"playerPos"`
	Battery           int                 `json:"battery"`
	MaxBattery        int                 `json:"maxBattery"`
	Score             int                 `json:"score"`
	TotalParks        int                 `json:"totalParks"`
	VisitedParks      []*VisitedPark      `json:"visitedParks"`
	Message           string              `json:"message"`
	GameOver          bool                `json:"gameOver"`
	Victory           bool                `json:"victory"`
	MapName           string              `json:"mapName"`
	MoveHistory       []*MoveHistoryEntry `json:"moveHistory"`
	TotalMoves        int                 `json:"totalMoves"`
	ResetCount        int                 `json:"resetCount"`
	NearbyGrid        [][]*Cell           `json:"nearbyGrid"`
	CurrentMoves      []*MoveHistoryEntry `json:"currentMoves"`
	CurrentMovesCount int                 `json:"currentMovesCount"`
	BatteryRisk       string              `json:"batteryRisk"`
	FogEnabled        bool                `json:"fogEnabled"`
	FogRadius         int                 `json:"fogRadius"`
	MoveDelayMs       int                 `json:"moveDelayMs"`

	access GridAccess
}

// SetGridAccess sets the policy guarding the grid field.
func (g *GameState) SetGridAccess(a GridAccess) { g.access = a }

// GridAccess returns the policy guarding the grid field.
func (g *GameState) GridAccess() GridAccess { return g.access }

// GameMap is bound in gqlgen.yml instead of being generated so it can carry
// the unexported layout access policy.
type GameMap struct {
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	GridSize          int                `json:"gridSize"`
	MaxBattery        int                `json:"maxBattery"`
	StartingBattery   int                `json:"startingBattery"`
	Layout            []string           `json:"layout"`
	Legend            []*LegendEntry     `json:"legend"`
	CellConfigs       []*CellConfigEntry `json:"cellConfigs"`

	access GridAccess
}

// SetLayoutAccess sets the policy guarding the layout field.
func (m *GameMap) SetLayoutAccess(a GridAccess) { m.access = a }

// LayoutAccess returns the policy guarding the layout field.
func (m *GameMap) LayoutAccess() GridAccess { return m.access }
