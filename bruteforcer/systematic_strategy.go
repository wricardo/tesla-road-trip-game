package main

import (
	"log"
	"math"
)

// SystematicStrategy plans complete park collection routes before execution
type SystematicStrategy struct {
	width       int
	height      int
	allParks    []ParkInfo
	allChargers []Position
	parkMap     map[Position]string // Position -> Park ID

	// Route planning
	collectionOrder []Position // Planned park collection order
	currentTarget   *Position  // Current park we're navigating to
	targetIndex     int        // Index in collectionOrder

	// Battery management
	chargingTarget *Position // Charger we're committed to reaching
	needsCharge    bool      // Flag to force charging mode

	// State tracking
	visitedCells map[Position]int
	stuckCount   int
	lastProgress int
}

type ParkInfo struct {
	Pos Position
	ID  string
}

func NewSystematicStrategy(state *GameState) *SystematicStrategy {
	s := &SystematicStrategy{
		width:        len(state.Grid[0]),
		height:       len(state.Grid),
		allParks:     make([]ParkInfo, 0),
		allChargers:  make([]Position, 0),
		parkMap:      make(map[Position]string),
		visitedCells: make(map[Position]int),
		targetIndex:  0,
		stuckCount:   0,
		lastProgress: 0,
	}

	// Scan grid for parks and chargers
	for y := 0; y < len(state.Grid); y++ {
		for x := 0; x < len(state.Grid[0]); x++ {
			cell := state.Grid[y][x]
			pos := Position{X: x, Y: y}

			if cell.Type == "park" {
				s.allParks = append(s.allParks, ParkInfo{Pos: pos, ID: cell.ID})
				s.parkMap[pos] = cell.ID
			} else if cell.Type == "home" || cell.Type == "supercharger" {
				s.allChargers = append(s.allChargers, pos)
			}
		}
	}

	log.Printf("📊 Systematic Strategy: %d parks, %d chargers", len(s.allParks), len(s.allChargers))

	// Plan initial collection order
	s.planCollectionOrder(state)

	return s
}

// planCollectionOrder creates an optimized park collection sequence
func (s *SystematicStrategy) planCollectionOrder(state *GameState) {
	if len(s.allParks) == 0 {
		return
	}

	// Build distance matrix once (optimization: use Manhattan for initial estimate)
	distMatrix := make(map[Position]map[Position]int)
	allPositions := []Position{state.PlayerPos}
	for _, park := range s.allParks {
		allPositions = append(allPositions, park.Pos)
	}

	// Cache distances
	for _, from := range allPositions {
		distMatrix[from] = make(map[Position]int)
		for _, to := range allPositions {
			if from == to {
				distMatrix[from][to] = 0
			} else {
				// Use Manhattan as fast heuristic, BFS only when needed
				distMatrix[from][to] = s.manhattanDistance(from, to)
			}
		}
	}

	// Nearest-neighbor with battery awareness
	remaining := make(map[int]bool)
	for i := range s.allParks {
		remaining[i] = true
	}

	s.collectionOrder = make([]Position, 0, len(s.allParks))
	currentPos := state.PlayerPos
	currentBattery := state.MaxBattery

	// Build route considering battery constraints
	for len(remaining) > 0 {
		nearestIdx := -1
		minScore := math.MaxFloat64

		for idx := range remaining {
			parkPos := s.allParks[idx].Pos
			dist := distMatrix[currentPos][parkPos]

			// Calculate score: distance + charging penalty
			score := float64(dist)

			// If we'd need to charge, add penalty
			if currentBattery < dist+5 {
				// Find nearest charger
				chargerDist := s.findNearestChargerDistance(currentPos)
				score += float64(chargerDist) * 1.5 // Penalty for detour
			}

			if score < minScore {
				minScore = score
				nearestIdx = idx
			}
		}

		if nearestIdx == -1 {
			log.Printf("⚠️  No reachable parks remaining")
			break
		}

		parkPos := s.allParks[nearestIdx].Pos
		dist := distMatrix[currentPos][parkPos]

		// Simulate battery usage
		if currentBattery < dist+5 {
			// Would need to charge
			currentBattery = state.MaxBattery - dist
		} else {
			currentBattery -= dist
		}

		// Add to route
		s.collectionOrder = append(s.collectionOrder, parkPos)
		currentPos = parkPos
		delete(remaining, nearestIdx)
	}

	log.Printf("📋 Planned collection order: %d parks", len(s.collectionOrder))
	for i, pos := range s.collectionOrder {
		parkID := s.parkMap[pos]
		log.Printf("  %d. Park %s at (%d,%d)", i+1, parkID, pos.X, pos.Y)
	}
}

func (s *SystematicStrategy) findNearestChargerDistance(pos Position) int {
	minDist := math.MaxInt32
	for _, chargerPos := range s.allChargers {
		dist := s.manhattanDistance(pos, chargerPos)
		if dist < minDist {
			minDist = dist
		}
	}
	return minDist
}

func (s *SystematicStrategy) NextMove(state *GameState) string {
	s.visitedCells[state.PlayerPos]++

	cellType := state.Grid[state.PlayerPos.Y][state.PlayerPos.X].Type
	isOnCharger := (cellType == "home" || cellType == "supercharger")

	// Check if we've reached charger and have sufficient charge
	if s.chargingTarget != nil && isOnCharger {
		if state.Battery >= state.MaxBattery-1 {
			log.Printf("⚡ Fully charged: %d/%d", state.Battery, state.MaxBattery)
			s.chargingTarget = nil
			s.needsCharge = false
		}
		// Don't clear charging target until we're on a charger and full
	}

	// If we're in charging mode, commit to reaching the charger
	if s.chargingTarget != nil {
		if isOnCharger {
			// On charger but not full - wiggle to charge
			for _, dir := range []string{"up", "down", "left", "right"} {
				newPos := s.getNewPosition(state.PlayerPos, dir)
				if s.isValidPosition(newPos, state) {
					return dir
				}
			}
		} else {
			// Still navigating to charger
			path := s.BFS(state.PlayerPos, *s.chargingTarget, state)
			if path != nil && len(path) > 0 {
				return path[0]
			}
			// Path failed, clear and recompute
			s.chargingTarget = nil
		}
	}

	// Check if we need to charge
	batteryLow := state.Battery < (state.MaxBattery / 3)
	if (batteryLow || s.needsCharge) && !isOnCharger && s.chargingTarget == nil {
		// Find nearest charger and commit to it
		var nearestCharger *Position
		minDist := 999999
		for _, chargerPos := range s.allChargers {
			path := s.BFS(state.PlayerPos, chargerPos, state)
			if path != nil && len(path) < minDist && state.Battery >= len(path) {
				minDist = len(path)
				cp := chargerPos
				nearestCharger = &cp
			}
		}
		if nearestCharger != nil {
			log.Printf("🔋 Going to charger (%d,%d) - Battery: %d/%d",
				nearestCharger.X, nearestCharger.Y, state.Battery, state.MaxBattery)
			s.chargingTarget = nearestCharger
			return s.NextMove(state) // Recurse with charging target set
		} else {
			log.Printf("❌ No reachable charger!")
			return ""
		}
	}

	// Track progress
	parksCollected := len(state.VisitedParks)
	if parksCollected > s.lastProgress {
		s.lastProgress = parksCollected
		s.stuckCount = 0
		log.Printf("✅ Parks: %d/%d", parksCollected, len(s.allParks))
	} else {
		s.stuckCount++
	}

	// Update target if current was collected
	if s.currentTarget != nil {
		parkID := s.parkMap[*s.currentTarget]
		if state.VisitedParks[parkID] {
			log.Printf("✅ Collected %s", parkID)
			s.currentTarget = nil
			s.targetIndex++
			s.stuckCount = 0

			// CRITICAL: After collecting park, check if we can still reach a charger
			// If battery is below 50%, proactively charge to avoid getting stranded
			nearestChargerDist := 999999
			for _, chargerPos := range s.allChargers {
				path := s.BFS(state.PlayerPos, chargerPos, state)
				if path != nil && len(path) < nearestChargerDist {
					nearestChargerDist = len(path)
				}
			}

			if state.Battery <= nearestChargerDist+2 || state.Battery < (state.MaxBattery/2) {
				log.Printf("⚡ Post-park charge needed: %d battery, charger %d away", state.Battery, nearestChargerDist)
				s.needsCharge = true
				return s.NextMove(state)
			}
		}
	}

	// Select next target from planned order
	if s.currentTarget == nil {
		for s.targetIndex < len(s.collectionOrder) {
			pos := s.collectionOrder[s.targetIndex]
			parkID := s.parkMap[pos]

			if !state.VisitedParks[parkID] {
				s.currentTarget = &pos
				log.Printf("🎯 %s (%d,%d)", parkID, pos.X, pos.Y)
				break
			}
			s.targetIndex++
		}
	}

	// No more targets - try to find ANY remaining unvisited park
	if s.currentTarget == nil {
		for _, parkInfo := range s.allParks {
			if !state.VisitedParks[parkInfo.ID] {
				path := s.BFS(state.PlayerPos, parkInfo.Pos, state)
				if path != nil {
					s.currentTarget = &parkInfo.Pos
					log.Printf("🔄 Trying previously skipped park %s at (%d,%d)",
						parkInfo.ID, parkInfo.Pos.X, parkInfo.Pos.Y)
					break
				}
			}
		}
	}

	// Still no target - game should be won or unwinnable
	if s.currentTarget == nil {
		return ""
	}

	// Try to find path to current target
	path := s.BFS(state.PlayerPos, *s.currentTarget, state)

	// If no path found, skip this park
	if path == nil {
		log.Printf("⚠️  No path to park at (%d,%d) - skipping", s.currentTarget.X, s.currentTarget.Y)
		s.currentTarget = nil
		s.targetIndex++
		return s.NextMove(state) // Try next park
	}

	// If stuck on same park for too long, skip it
	if s.stuckCount > 200 {
		log.Printf("⚠️  Stuck on park at (%d,%d) for %d moves - skipping",
			s.currentTarget.X, s.currentTarget.Y, s.stuckCount)
		s.currentTarget = nil
		s.targetIndex++
		s.stuckCount = 0
		return s.NextMove(state) // Try next park
	}

	// Battery check: ensure we can reach target AND get to a charger from there
	pathLength := len(path)

	// Find nearest charger from target position
	nearestChargerFromTarget := 999999
	for _, chargerPos := range s.allChargers {
		dist := s.manhattanDistance(*s.currentTarget, chargerPos)
		if dist < nearestChargerFromTarget {
			nearestChargerFromTarget = dist
		}
	}

	// Need: path to target + distance to charger from target + buffer
	requiredBattery := pathLength + nearestChargerFromTarget + 3

	// If this exceeds max, we'll have to find intermediate charger during journey
	if requiredBattery > state.MaxBattery {
		requiredBattery = pathLength + 3
	}

	// Only charge if we need more AND we're not already near full
	if state.Battery < requiredBattery && state.Battery < (state.MaxBattery - 2) {
		log.Printf("⚠️  Battery: %d < %d needed (%d to target + %d escape)",
			state.Battery, requiredBattery, pathLength, nearestChargerFromTarget)
		s.needsCharge = true
		return s.NextMove(state)
	}

	// Return first move in path
	if len(path) > 0 {
		return path[0]
	}

	// Stuck - try exploration
	if s.stuckCount > 100 {
		return s.exploreMove(state)
	}

	return ""
}

// NextMoves returns up to maxMoves planned moves for efficient bulk execution
func (s *SystematicStrategy) NextMoves(state *GameState, maxMoves int) []string {
	s.visitedCells[state.PlayerPos]++

	// CRITICAL FIX: If standing on a charger with full battery, move off immediately
	// This prevents infinite loops when bulk moves cross charger tiles
	cellType := state.Grid[state.PlayerPos.Y][state.PlayerPos.X].Type
	if (cellType == "home" || cellType == "supercharger") && state.Battery >= state.MaxBattery {
		// Try to move to a non-charger position
		for _, dir := range []string{"up", "down", "left", "right"} {
			newPos := s.getNewPosition(state.PlayerPos, dir)
			if s.isValidPosition(newPos, state) {
				newCellType := state.Grid[newPos.Y][newPos.X].Type
				if newCellType != "home" && newCellType != "supercharger" {
					log.Printf("Moving off charger: %s", dir)
					return []string{dir}
				}
			}
		}

		// If surrounded by chargers, move toward nearest park to get out
		if s.currentTarget != nil {
			// Try moving toward target
			dx := s.currentTarget.X - state.PlayerPos.X
			dy := s.currentTarget.Y - state.PlayerPos.Y

			var preferredDir string
			if abs(dx) > abs(dy) {
				if dx > 0 {
					preferredDir = "right"
				} else {
					preferredDir = "left"
				}
			} else {
				if dy > 0 {
					preferredDir = "down"
				} else {
					preferredDir = "up"
				}
			}

			newPos := s.getNewPosition(state.PlayerPos, preferredDir)
			if s.isValidPosition(newPos, state) {
				log.Printf("Moving through charger field toward target: %s", preferredDir)
				return []string{preferredDir}
			}
		}

		// Last resort: any valid move
		for _, dir := range []string{"up", "down", "left", "right"} {
			newPos := s.getNewPosition(state.PlayerPos, dir)
			if s.isValidPosition(newPos, state) {
				log.Printf("Moving through chargers: %s", dir)
				return []string{dir}
			}
		}
	}

	// Track progress
	parksCollected := len(state.VisitedParks)
	if parksCollected > s.lastProgress {
		s.lastProgress = parksCollected
		s.stuckCount = 0
		log.Printf("✅ Progress: %d/%d parks", parksCollected, len(s.allParks))
	} else {
		s.stuckCount++
	}

	// Update target if current was collected
	if s.currentTarget != nil {
		parkID := s.parkMap[*s.currentTarget]
		if state.VisitedParks[parkID] {
			log.Printf("✅ Collected park %s at (%d,%d)", parkID, s.currentTarget.X, s.currentTarget.Y)
			s.currentTarget = nil
			s.targetIndex++
			s.stuckCount = 0
		}
	}

	// Select next target from planned order
	if s.currentTarget == nil {
		for s.targetIndex < len(s.collectionOrder) {
			pos := s.collectionOrder[s.targetIndex]
			parkID := s.parkMap[pos]

			if !state.VisitedParks[parkID] {
				s.currentTarget = &pos
				log.Printf("🎯 %s (%d,%d)", parkID, pos.X, pos.Y)
				break
			}
			s.targetIndex++
		}
	}

	// No more targets - victory or stuck
	if s.currentTarget == nil {
		if parksCollected == len(s.allParks) {
			log.Printf("🎉 All parks collected!")
		} else {
			log.Printf("⚠️  No valid target but %d/%d parks collected", parksCollected, len(s.allParks))
		}
		return []string{}
	}

	// Try to find path to current target
	path := s.BFS(state.PlayerPos, *s.currentTarget, state)

	// If no path found, mark this park as problematic and try next one
	if path == nil {
		log.Printf("⚠️  No path to target park at (%d,%d) - trying next park",
			s.currentTarget.X, s.currentTarget.Y)
		s.stuckCount += 10 // Heavily penalize
		s.currentTarget = nil
		s.targetIndex++

		// Try next target
		for s.targetIndex < len(s.collectionOrder) {
			pos := s.collectionOrder[s.targetIndex]
			parkID := s.parkMap[pos]
			if !state.VisitedParks[parkID] {
				testPath := s.BFS(state.PlayerPos, pos, state)
				if testPath != nil {
					s.currentTarget = &pos
					log.Printf("🎯 Switched target: Park %s at (%d,%d)", parkID, pos.X, pos.Y)
					return s.NextMoves(state, maxMoves) // Recursive call with new target
				}
			}
			s.targetIndex++
		}

		// No reachable parks found
		log.Printf("❌ No reachable parks found!")
		return []string{}
	}

	if path != nil {
		pathCost := len(path)
		safetyBuffer := 5

		// Need to charge?
		if state.Battery < pathCost+safetyBuffer {
			// Check if already on charger
			cellType := state.Grid[state.PlayerPos.Y][state.PlayerPos.X].Type
			if cellType == "home" || cellType == "supercharger" {
				// Already charging - move off the charger first to avoid "charging" message loop
				// Just return first move of path to target
				if len(path) > 0 {
					return []string{path[0]}
				}
			} else {
				// Need to find charger - get path to nearest charger
				chargerPath := s.findPathToNearestCharger(state)
				if len(chargerPath) > 0 {
					// Return path to charger (up to maxMoves)
					if len(chargerPath) > maxMoves {
						return chargerPath[:maxMoves]
					}
					return chargerPath
				}
				log.Printf("⚠️  Need charge but no charger path!")
			}
		}

		// Return path to target (limit to fewer moves to avoid getting stuck on chargers)
		limit := maxMoves
		if limit > 5 {
			limit = 5 // Reduce batch size to handle unexpected chargers in path
		}
		if len(path) > limit {
			return path[:limit]
		}
		return path
	}

	// Stuck - try single exploration move
	if s.stuckCount > 50 {
		log.Printf("⚠️  Stuck for %d moves, trying exploration", s.stuckCount)
		move := s.exploreMove(state)
		if move != "" {
			return []string{move}
		}
	}

	return []string{}
}

func (s *SystematicStrategy) navigateToTarget(state *GameState, target Position) string {
	path := s.BFS(state.PlayerPos, target, state)

	if path != nil && len(path) > 0 {
		return path[0]
	}

	// Try exploring if no direct path
	return s.exploreMove(state)
}

func (s *SystematicStrategy) findDirectionToNearestCharger(state *GameState) string {
	path := s.findPathToNearestCharger(state)
	if len(path) > 0 {
		return path[0]
	}
	return ""
}

func (s *SystematicStrategy) findPathToNearestCharger(state *GameState) []string {
	var shortestPath []string
	minDist := math.MaxInt32

	for _, chargerPos := range s.allChargers {
		path := s.BFS(state.PlayerPos, chargerPos, state)
		if path != nil && len(path) < minDist {
			minDist = len(path)
			shortestPath = path
		}
	}

	return shortestPath
}

func (s *SystematicStrategy) exploreMove(state *GameState) string {
	// Try least visited direction
	type DirScore struct {
		dir   string
		score int
	}

	options := []DirScore{}
	for _, dir := range []string{"up", "down", "left", "right"} {
		newPos := s.getNewPosition(state.PlayerPos, dir)
		if !s.isValidPosition(newPos, state) {
			continue
		}

		visitCount := s.visitedCells[newPos]
		options = append(options, DirScore{dir: dir, score: visitCount})
	}

	if len(options) == 0 {
		return ""
	}

	// Pick least visited
	best := options[0]
	for _, opt := range options {
		if opt.score < best.score {
			best = opt
		}
	}

	return best.dir
}

func (s *SystematicStrategy) BFS(start, goal Position, state *GameState) []string {
	if start == goal {
		return []string{}
	}

	type QueueItem struct {
		pos  Position
		path []string
	}

	queue := []QueueItem{{pos: start, path: []string{}}}
	visited := make(map[Position]bool)
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range []string{"up", "down", "left", "right"} {
			newPos := s.getNewPosition(current.pos, dir)

			if visited[newPos] || !s.isValidPosition(newPos, state) {
				continue
			}

			newPath := append([]string{}, current.path...)
			newPath = append(newPath, dir)

			if newPos == goal {
				return newPath
			}

			visited[newPos] = true
			queue = append(queue, QueueItem{pos: newPos, path: newPath})
		}
	}

	return nil
}

func (s *SystematicStrategy) isValidPosition(pos Position, state *GameState) bool {
	if pos.Y < 0 || pos.Y >= len(state.Grid) || pos.X < 0 || pos.X >= len(state.Grid[0]) {
		return false
	}
	cellType := state.Grid[pos.Y][pos.X].Type

	// Water and buildings are always impassable
	if cellType == "water" || cellType == "building" {
		return false
	}

	// Allow movement through chargers - they're passable
	// We'll handle the "charging" message at the API level
	return true
}

func (s *SystematicStrategy) getNewPosition(pos Position, dir string) Position {
	switch dir {
	case "up":
		return Position{X: pos.X, Y: pos.Y - 1}
	case "down":
		return Position{X: pos.X, Y: pos.Y + 1}
	case "left":
		return Position{X: pos.X - 1, Y: pos.Y}
	case "right":
		return Position{X: pos.X + 1, Y: pos.Y}
	}
	return pos
}

func (s *SystematicStrategy) manhattanDistance(a, b Position) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *SystematicStrategy) Reset() {
	s.visitedCells = make(map[Position]int)
	s.currentTarget = nil
	s.targetIndex = 0
	s.stuckCount = 0
	s.lastProgress = 0
}
