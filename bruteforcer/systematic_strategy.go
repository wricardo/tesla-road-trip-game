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

	// Build BFS distance matrix (accurate for mazes, not Manhattan).
	allPositions := []Position{state.PlayerPos}
	for _, park := range s.allParks {
		allPositions = append(allPositions, park.Pos)
	}
	distMatrix := make(map[Position]map[Position]int)
	for _, from := range allPositions {
		distMatrix[from] = make(map[Position]int)
		for _, to := range allPositions {
			if from == to {
				distMatrix[from][to] = 0
			} else {
				p := s.BFS(from, to, state)
				if p != nil {
					distMatrix[from][to] = len(p)
				} else {
					distMatrix[from][to] = math.MaxInt32 / 2
				}
			}
		}
	}

	// Nearest-neighbor ordering by true BFS distance.
	remaining := make(map[int]bool)
	for i := range s.allParks {
		remaining[i] = true
	}

	s.collectionOrder = make([]Position, 0, len(s.allParks))
	currentPos := state.PlayerPos

	for len(remaining) > 0 {
		nearestIdx := -1
		minDist := math.MaxInt32

		for idx := range remaining {
			d := distMatrix[currentPos][s.allParks[idx].Pos]
			if d < minDist {
				minDist = d
				nearestIdx = idx
			}
		}

		if nearestIdx == -1 {
			log.Printf("⚠️  No reachable parks remaining")
			break
		}

		s.collectionOrder = append(s.collectionOrder, s.allParks[nearestIdx].Pos)
		currentPos = s.allParks[nearestIdx].Pos
		delete(remaining, nearestIdx)
	}

	// Post-process: move parks with high escape cost (> maxBattery*2/3) to end.
	// These parks are easiest to collect last (no escape needed from final park).
	threshold := state.MaxBattery * 2 / 3
	var hardParks, easyParks []Position
	for _, pos := range s.collectionOrder {
		if s.bfsNearestChargerDistStatic(pos, state) > threshold {
			hardParks = append(hardParks, pos)
		} else {
			easyParks = append(easyParks, pos)
		}
	}
	s.collectionOrder = append(easyParks, hardParks...)

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
	isOnCharger := cellType == "home" || cellType == "supercharger"

	// On charger: game already restored battery — always clear charge state
	if isOnCharger {
		s.chargingTarget = nil
		s.needsCharge = false
	}

	// Committed to a charger: navigate there
	if s.chargingTarget != nil {
		path := s.BFS(state.PlayerPos, *s.chargingTarget, state)
		if path != nil && len(path) > 0 {
			return path[0]
		}
		s.chargingTarget = nil // path failed, replan
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

	// Clear collected target
	if s.currentTarget != nil {
		if state.VisitedParks[s.parkMap[*s.currentTarget]] {
			log.Printf("✅ Collected %s", s.parkMap[*s.currentTarget])
			s.currentTarget = nil
			s.targetIndex++
			s.stuckCount = 0
		}
	}

	// Select next target
	if s.currentTarget == nil {
		for s.targetIndex < len(s.collectionOrder) {
			pos := s.collectionOrder[s.targetIndex]
			if !state.VisitedParks[s.parkMap[pos]] {
				s.currentTarget = &pos
				log.Printf("🎯 %s (%d,%d)", s.parkMap[pos], pos.X, pos.Y)
				break
			}
			s.targetIndex++
		}
	}
	// Fallback: any unvisited park reachable by BFS
	if s.currentTarget == nil {
		for _, p := range s.allParks {
			if !state.VisitedParks[p.ID] {
				if path := s.BFS(state.PlayerPos, p.Pos, state); path != nil {
					pos := p.Pos
					s.currentTarget = &pos
					log.Printf("🔄 Fallback target %s (%d,%d)", p.ID, pos.X, pos.Y)
					break
				}
			}
		}
	}
	if s.currentTarget == nil {
		return ""
	}

	if s.stuckCount > 200 {
		log.Printf("⚠️ Stuck %d moves, exploring", s.stuckCount)
		s.currentTarget = nil
		s.targetIndex++
		s.stuckCount = 0
		return s.exploreMove(state)
	}

	// Path to target
	path := s.BFS(state.PlayerPos, *s.currentTarget, state)
	if path == nil {
		log.Printf("⚠️ No path to (%d,%d), skipping", s.currentTarget.X, s.currentTarget.Y)
		s.currentTarget = nil
		s.targetIndex++
		return s.NextMove(state)
	}

	// Simulate the path, accounting for mid-route charger restores.
	batteryAtPark := s.batteryAfterPath(state.PlayerPos, path, state, state.Battery)
	if batteryAtPark < 0 {
		// Run out of battery on the way — charge first
		return s.goCharge(state)
	}

	// Can we escape from the park to a charger with the battery we'll have?
	escapeLen := s.bfsNearestChargerDist(*s.currentTarget, state)
	if batteryAtPark < escapeLen {
		// If this is the last uncollected park, no escape needed — just reach it.
		uncollected := 0
		for _, p := range s.allParks {
			if !state.VisitedParks[p.ID] {
				uncollected++
			}
		}
		if uncollected == 1 {
			return path[0]
		}
		// Already at max battery: route via charger nearest to destination.
		if state.Battery >= state.MaxBattery {
			return s.routeViaChargerNearDest(state)
		}
		return s.goCharge(state)
	}

	return path[0]
}

// routeViaChargerNearDest navigates to the charger nearest to the current target,
// then from there we will have maxBattery to attempt the final leg.
func (s *SystematicStrategy) routeViaChargerNearDest(state *GameState) string {
	if s.currentTarget == nil {
		return s.goCharge(state)
	}
	// Find charger with shortest BFS distance from destination.
	var best *Position
	minFromDest := math.MaxInt32
	for _, cp := range s.allChargers {
		p := s.BFS(*s.currentTarget, cp, state)
		if p != nil && len(p) < minFromDest {
			minFromDest = len(p)
			c := cp
			best = &c
		}
	}
	if best == nil || *best == state.PlayerPos {
		// Fallback: just go (might die, but no better option)
		path := s.BFS(state.PlayerPos, *s.currentTarget, state)
		if path != nil && len(path) > 0 {
			return path[0]
		}
		return s.exploreMove(state)
	}
	log.Printf("🔋 Via-dest charger (%d,%d) battery=%d/%d", best.X, best.Y, state.Battery, state.MaxBattery)
	s.chargingTarget = best
	path := s.BFS(state.PlayerPos, *best, state)
	if path != nil && len(path) > 0 && state.Battery >= len(path) {
		return path[0]
	}
	// Can't reach that charger either — just go for the park directly
	path = s.BFS(state.PlayerPos, *s.currentTarget, state)
	if path != nil && len(path) > 0 {
		return path[0]
	}
	return s.exploreMove(state)
}

// goCharge routes to the nearest reachable charger (that is not the current cell).
func (s *SystematicStrategy) goCharge(state *GameState) string {
	var nearest *Position
	minDist := math.MaxInt32
	for _, cp := range s.allChargers {
		p := s.BFS(state.PlayerPos, cp, state)
		// Require len > 0 so we never "charge" at current position (infinite loop).
		if p != nil && len(p) > 0 && len(p) < minDist && state.Battery >= len(p) {
			minDist = len(p)
			c := cp
			nearest = &c
		}
	}
	if nearest == nil {
		log.Printf("❌ No reachable charger from (%d,%d) battery=%d", state.PlayerPos.X, state.PlayerPos.Y, state.Battery)
		return s.exploreMove(state)
	}
	log.Printf("🔋 Charging at (%d,%d) battery=%d/%d", nearest.X, nearest.Y, state.Battery, state.MaxBattery)
	s.chargingTarget = nearest
	path := s.BFS(state.PlayerPos, *nearest, state)
	if path != nil && len(path) > 0 {
		return path[0]
	}
	return s.exploreMove(state)
}

// batteryAfterPath simulates walking path from start, accounting for charger restores.
// Returns battery level at destination, or -1 if battery dies before the end.
func (s *SystematicStrategy) batteryAfterPath(start Position, path []string, state *GameState, startBattery int) int {
	battery := startBattery
	pos := start
	for _, dir := range path {
		if battery <= 0 {
			return -1
		}
		pos = s.getNewPosition(pos, dir)
		battery--
		cellType := state.Grid[pos.Y][pos.X].Type
		if cellType == "home" || cellType == "supercharger" {
			battery = state.MaxBattery
		}
	}
	return battery
}

// bfsNearestChargerDistStatic is like bfsNearestChargerDist but callable during planning
// (same implementation, separate name for clarity).
func (s *SystematicStrategy) bfsNearestChargerDistStatic(pos Position, state *GameState) int {
	return s.bfsNearestChargerDist(pos, state)
}

// bfsNearestChargerDist returns the BFS distance from pos to the nearest charger.
func (s *SystematicStrategy) bfsNearestChargerDist(pos Position, state *GameState) int {
	min := math.MaxInt32
	for _, cp := range s.allChargers {
		p := s.BFS(pos, cp, state)
		if p != nil && len(p) < min {
			min = len(p)
		}
	}
	if min == math.MaxInt32 {
		return 0
	}
	return min
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
