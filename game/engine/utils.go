package engine

// CountTotalParks counts the total number of parks in the grid
func CountTotalParks(grid [][]Cell) int {
	count := 0
	for _, row := range grid {
		for _, cell := range row {
			if cell.Type == Park {
				count++
			}
		}
	}
	return count
}

// ManhattanDistance calculates the Manhattan distance between two positions
func ManhattanDistance(from, to Position) int {
	dx := from.X - to.X
	if dx < 0 {
		dx = -dx
	}
	dy := from.Y - to.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// FindNearestUnvisitedPark finds the closest unvisited park and returns its position and distance
func FindNearestUnvisitedPark(state *GameState) (Position, int, bool) {
	minDistance := -1
	var nearestPos Position
	found := false

	for y := 0; y < len(state.Grid); y++ {
		for x := 0; x < len(state.Grid[y]); x++ {
			cell := state.Grid[y][x]
			if cell.Type == Park && !cell.Visited {
				pos := Position{X: x, Y: y}
				distance := ManhattanDistance(state.PlayerPos, pos)
				if minDistance == -1 || distance < minDistance {
					minDistance = distance
					nearestPos = pos
					found = true
				}
			}
		}
	}

	return nearestPos, minDistance, found
}

// FindNearestCharger finds the closest charging station (Home or Supercharger) and returns position, distance, and type
func FindNearestCharger(state *GameState) (Position, int, CellType, bool) {
	minDistance := -1
	var nearestPos Position
	var chargerType CellType
	found := false

	for y := 0; y < len(state.Grid); y++ {
		for x := 0; x < len(state.Grid[y]); x++ {
			cell := state.Grid[y][x]
			if cell.Type == Home || cell.Type == Supercharger {
				pos := Position{X: x, Y: y}
				distance := ManhattanDistance(state.PlayerPos, pos)
				if minDistance == -1 || distance < minDistance {
					minDistance = distance
					nearestPos = pos
					chargerType = cell.Type
					found = true
				}
			}
		}
	}

	return nearestPos, minDistance, chargerType, found
}

// AnalyzeBatteryRisk assesses battery danger level based on current battery and distance to nearest charger
func AnalyzeBatteryRisk(state *GameState) string {
	if state.Battery <= 0 {
		return "CRITICAL: Battery empty!"
	}

	_, chargerDistance, _, chargerFound := FindNearestCharger(state)
	if !chargerFound {
		return "WARNING: No chargers available!"
	}

	if state.Battery <= chargerDistance {
		return "DANGER: Insufficient battery to reach nearest charger!"
	} else if state.Battery <= chargerDistance+2 {
		return "CAUTION: Low battery, prioritize charging"
	} else if state.Battery <= state.MaxBattery/3 {
		return "LOW: Consider charging soon"
	}

	return "SAFE: Battery sufficient"
}

// CountCellType counts the total number of cells of a specific type in the grid
func CountCellType(grid [][]Cell, cellType CellType) int {
	count := 0
	for _, row := range grid {
		for _, cell := range row {
			if cell.Type == cellType {
				count++
			}
		}
	}
	return count
}
