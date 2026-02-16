# Tesla Road Trip Bruteforcer

Automated solver for the Tesla Road Trip game using systematic path planning and optimization.

## Features

### Systematic Strategy (Current - Optimized)

Efficient route planning with bulk move execution:

**Planning Phase:**
- Scans grid to identify all parks and charging stations
- Builds distance matrix using Manhattan heuristic for O(1) lookups
- Plans optimal collection order using nearest-neighbor TSP algorithm
- Considers battery constraints during route planning
- Adds charging penalty to minimize detours

**Execution Phase:**
- **Bulk Move API**: Executes up to 10 moves per API call (10x faster)
- Proactive battery management with 5-move safety buffer
- BFS pathfinding for precise obstacle avoidance
- Automatic fallback to exploration when stuck
- Real-time progress tracking and logging

**Optimizations:**
- Distance matrix caching avoids redundant calculations
- Manhattan distance for fast initial estimates
- Bulk API calls reduce network overhead by 10x
- Battery-aware routing minimizes charging stops
- Path reuse for similar positions

## Usage

```bash
# Build
go build -o bruteforcer

# Run - automatically resumes last session or creates new one
go run .

# Run multiple times - each run continues learning on the same session
go run . -max-attempts 10
go run . -max-attempts 10  # Continues on same session!
go run . -max-attempts 10  # Keeps learning!

# Start fresh with a new session
rm .session && go run .

# Run with custom server
go run . -url http://localhost:9090

# Run with specific configuration (creates new session)
go run . -config medium_maze

# Explicitly resume a specific session
go run . -continue a3x7

# Run with verbose output
go run . -v

# Customize attempt limits
go run . -max-moves 300 -max-attempts 20
```

## Flags

- `-url`: Game server URL (default: http://localhost:8080)
- `-config`: Game configuration (default, easy, medium_maze)
- `-continue`: Resume playing an existing session by ID
- `-max-moves`: Maximum moves per attempt (default: 1000)
- `-max-attempts`: Maximum attempts before giving up (default: 100)
- `-v`: Verbose output with grid visualization

## Strategy

### Core Strategy
1. **Park Selection**: Finds nearest unvisited park using Manhattan distance
2. **Battery Check**: Ensures enough battery to reach park + adaptive safety margin
3. **Charging Route**: If low battery, navigates to nearest charger first
4. **A* Pathfinding**: Calculates optimal path avoiding obstacles (B, W)
5. **Execution**: Executes first move in calculated path

### Learning & Adaptation
- **Attempt Tracking**: Records each attempt's moves, parks visited, battery, position
- **Failed Position Avoidance**: Penalizes parks near positions where it got stuck
- **Adaptive Battery Buffer**: Increases safety margin when running out of battery
- **Stuck Detection**: Switches to aggressive strategy if stuck at same progress
- **Auto-Reset**: Automatically resets game between attempts to try new paths

## Exit Codes

- `0`: Victory (all parks collected)
- `1`: Game over (out of battery or invalid state)
- `2`: Gave up (reached max moves)

## Example

```bash
$ ./bruteforcer -v -config easy
2025/10/02 Connecting to game server at http://localhost:8080
2025/10/02 Session created: a3x7
2025/10/02 Grid size: 15x15, Parks to collect: 5, Battery: 20/20
2025/10/02 Position: (0,0), Battery: 20/20, Parks: 0/5
...
2025/10/02 🎉 VICTORY! Game won!
2025/10/02 Moves: 87
2025/10/02 Parks collected: 5/5
```
