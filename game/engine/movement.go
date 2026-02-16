package engine

import (
	"fmt"
	"time"
)

// CanMoveTo checks if the player can move to the specified coordinates
func (gs *GameState) CanMoveTo(x, y int) bool {
	// Check bounds - handle non-square grids properly
	if y < 0 || y >= len(gs.Grid) {
		return false
	}
	if x < 0 || x >= len(gs.Grid[0]) {
		return false
	}
	cellType := gs.Grid[y][x].Type
	// Only water and buildings are obstacles - homes are passable and charge battery
	return cellType != Water && cellType != Building
}

// MovePlayer attempts to move the player in the specified direction
func (gs *GameState) MovePlayer(direction string, config *GameConfig) bool {
	if gs.GameOver {
		return false
	}

	newX, newY := gs.PlayerPos.X, gs.PlayerPos.Y

	switch direction {
	case "up":
		newY--
	case "down":
		newY++
	case "left":
		newX--
	case "right":
		newX++
	default:
		return false
	}

	// Check wall collision BEFORE battery check
	if !gs.CanMoveTo(newX, newY) {
		// Get the type of obstacle hit
		obstacleType := "boundary"
		if newY >= 0 && newY < len(gs.Grid) && newX >= 0 && newX < len(gs.Grid[0]) {
			obstacleType = string(gs.Grid[newY][newX].Type)
		}

		// Check if wall crash ends game
		if config.WallCrashEndsGame {
			gs.Message = fmt.Sprintf("COLLISION: Hit %s at (%d,%d) moving %s from (%d,%d)! Game Over!",
				obstacleType, newX, newY, direction, gs.PlayerPos.X, gs.PlayerPos.Y)
			if config.Messages.HitWall != "" {
				gs.Message = config.Messages.HitWall + fmt.Sprintf(" [Hit: %s at (%d,%d)]", obstacleType, newX, newY)
			}
			gs.GameOver = true
			return false
		}
		gs.Message = fmt.Sprintf("Can't move %s: %s at (%d,%d)", direction, obstacleType, newX, newY)
		if config.Messages.CantMove != "" {
			gs.Message = config.Messages.CantMove + fmt.Sprintf(" [Blocked by: %s]", obstacleType)
		}
		return false
	}

	// Now check battery for valid moves
	if gs.Battery <= 0 {
		gs.Message = config.Messages.OutOfBattery
		gs.GameOver = true
		return false
	}

	// Move player and consume battery
	gs.PlayerPos.X = newX
	gs.PlayerPos.Y = newY
	gs.Battery--

	// Check current cell
	currentCell := &gs.Grid[newY][newX]

	switch currentCell.Type {
	case Home:
		gs.Battery = gs.MaxBattery
		gs.Message = config.Messages.HomeCharge

	case Supercharger:
		gs.Battery = gs.MaxBattery
		gs.Message = config.Messages.SuperchargerCharge

	case Park:
		if currentCell.ID != "" && !gs.VisitedParks[currentCell.ID] {
			gs.VisitedParks[currentCell.ID] = true
			currentCell.Visited = true
			gs.Score++
			gs.Message = fmt.Sprintf(config.Messages.ParkVisited, gs.Score)

			// Check victory condition
			if gs.Score == CountTotalParks(gs.Grid) {
				gs.Victory = true
				gs.GameOver = true
				gs.Message = fmt.Sprintf(config.Messages.Victory, gs.Score)
			}
		} else if currentCell.Visited {
			gs.Message = config.Messages.ParkAlreadyVisited
		}

	default:
		gs.Message = fmt.Sprintf(config.Messages.BatteryStatus, gs.Battery, gs.MaxBattery)
	}

	// Check if stranded
	if gs.Battery == 0 && !gs.CanReachCharger() {
		gs.GameOver = true
		gs.Message = config.Messages.Stranded
	}

	return true
}

// CanReachCharger checks if the player can reach a charger from their current position
func (gs *GameState) CanReachCharger() bool {
	currentCell := gs.Grid[gs.PlayerPos.Y][gs.PlayerPos.X]
	return currentCell.Type == Home || currentCell.Type == Supercharger
}

// GenerateLocalView creates list of 8 surrounding cells around the player
func (gs *GameState) GenerateLocalView() []SurroundingCell {
	gridSize := len(gs.Grid)
	px, py := gs.PlayerPos.X, gs.PlayerPos.Y

	getCellType := func(x, y int) CellType {
		if x >= 0 && x < gridSize && y >= 0 && y < gridSize {
			return gs.Grid[y][x].Type
		}
		return Building // Out of bounds = building
	}

	directions := []struct{ dx, dy int }{
		{0, -1},  // North
		{1, -1},  // North-East
		{1, 0},   // East
		{1, 1},   // South-East
		{0, 1},   // South
		{-1, 1},  // South-West
		{-1, 0},  // West
		{-1, -1}, // North-West
	}

	surroundings := make([]SurroundingCell, 8)
	for i, dir := range directions {
		x, y := px+dir.dx, py+dir.dy
		surroundings[i] = SurroundingCell{
			X:    x,
			Y:    y,
			Type: getCellType(x, y),
		}
	}

	return surroundings
}

// AddMoveToHistory adds a move to the game's move history
func (gs *GameState) AddMoveToHistory(action string, fromPos, toPos Position, success bool) {
	entry := MoveHistoryEntry{
		Action:       action,
		FromPosition: fromPos,
		ToPosition:   toPos,
		Battery:      gs.Battery,
		Timestamp:    time.Now().Unix(),
		Success:      success,
		MoveNumber:   gs.TotalMoves + 1,
	}
	// Append to cumulative history (never cleared by reset) and increment total
	gs.MoveHistory = append(gs.MoveHistory, entry)
	gs.TotalMoves++

	// Append to current segment history and increment its counter
	gs.CurrentMoves = append(gs.CurrentMoves, entry)
	gs.CurrentMovesCount++
}
