# AI Strategy Guide for Tesla Road Trip Game

This guide provides strategies and techniques for AI agents playing the Tesla Road Trip game.

## Table of Contents

1. [Critical: Character Recognition](#critical-character-recognition)
2. [Game API Reference](#game-api-reference)
3. [Navigation Strategies](#navigation-strategies)
4. [Proven Success Patterns](#proven-success-patterns)

## Critical: Character Recognition

### The #1 Problem: Misreading 'R' as 'B' or 'W'

AI agents frequently fail to recognize road characters ('R') when they appear between obstacles.

**Character Reference:**
```
R = Road (PASSABLE) - You CAN move here
H = Home (PASSABLE + charges battery)
P = Park (PASSABLE + collectible objective)
S = Supercharger (PASSABLE + charges battery)
W = Water (IMPASSABLE)
B = Building (IMPASSABLE)
```

### Common Misreading Examples

**Hidden Road Between Buildings:**
```
What you might see: BBBBBWWWWWBBBBB
What it actually is: BBBBRWWWWWBBBBB
                          ^
                          This is an R (road)!
```

**Single Road in Building Cluster:**
```
What you might see: BBBBBBBBBB
What it actually is: BBBBRBBBBBB
                          ^
                          This is an R (road)!
```

### Mandatory Grid Analysis Protocol

When analyzing any grid row, you MUST:

1. **Parse character-by-character** - Don't scan patterns visually
2. **Verify suspected blockages** - If a row appears blocked, re-examine it position by position
3. **Double-check R vs B/W** - These characters look similar in monospace fonts
4. **Inspect, never probe** - If uncertain, read the cell data (`grid` / `nearbyGrid` `type` plus `allowedDirections`) before moving. Moving into a building, water, or off the map ends the game immediately, so exploratory "test" moves are never safe

## Game API Reference

Gameplay uses GraphQL at `/graphql` (or the MCP tools at `/mcp`: `create_session`, `game_state`, `move`, `bulk_move`, `reset_game`, ...). The only REST routes are `POST/GET /api/sessions`; there is no REST move/state endpoint.

### Base URL
```
http://localhost:8000/graphql
```

### Create a Session
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { createSession(mapID: \"easy\", moveDelayMs: 0) { id gameState { playerPos { x y } battery maxBattery } } }"}' | jq
```

Fog variant: `createSession(mapID: "easy", fogEnabled: true, fogRadius: 2, gridPassword: "secret", moveDelayMs: 0)`.

### Get Current State
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ gameState(sessionID: \"SESSION_ID\") { playerPos { x y } battery maxBattery batteryRisk score totalParks gameOver victory message grid { type visited id allowedDirections } } }"}' | jq
```

**Key fields:**
- `playerPos`: {x, y}; `grid` is row-major `grid[y][x]` (RIGHT = x+1, DOWN = y+1)
- `battery` / `maxBattery`; `batteryRisk`: SAFE, LOW, CAUTION, DANGER, CRITICAL, WARNING (no charger), UNKNOWN. Its charger distance is Manhattan distance ignoring walls and one-way rules: a heuristic, not reachability
- `score` / `totalParks` / `visitedParks`
- `gameOver` / `victory`
- `moveHistory`: complete move trail

Always select `allowedDirections` with `type`. One-way rule: a move is allowed only if its direction (north=UP, south=DOWN, east=RIGHT, west=LEFT) is listed on both the cell you leave and the cell you enter, whenever those lists are non-empty. Wrong-way moves are rejected, cost no battery, and do not end the game.

### Fog Sessions
In a fog session never select `grid` unless you pass the right password (`grid(password: "secret")`): a wrong or missing password errors and nulls the whole `gameState` response. Use `nearbyGrid` instead, and select `x y` on its cells:
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ gameState(sessionID: \"SESSION_ID\") { playerPos { x y } battery fogRadius nearbyGrid { x y type id allowedDirections } } }"}' | jq
```
`nearbyGrid` is a (2r+1)x(2r+1) window with r = `fogRadius` (r = 1, a 3x3 window, without fog). Each cell's `x` / `y` are its map coordinates, so store cells by those; off-map cells read `building`.

### Single Move
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { move(sessionID: \"SESSION_ID\", direction: RIGHT) { success message gameState { playerPos { x y } battery gameOver victory } } }"}' | jq
```

Directions: `UP`, `DOWN`, `LEFT`, `RIGHT`. Each move costs 1 battery; entering H or S refills to `maxBattery`; reaching 0 battery away from a charger ends the game.

### Bulk Moves
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { bulkMove(sessionID: \"SESSION_ID\", moves: [RIGHT, RIGHT, DOWN, LEFT]) { movesExecuted stoppedReason truncated batteryRisk gameState { playerPos { x y } battery gameOver victory } } }"}' | jq
```

- Processes up to 50 moves in sequence; extras are dropped with `truncated: true`
- Stops early on a failed move (see `stoppedReason`)

### Reset
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { reset(sessionID: \"SESSION_ID\") { playerPos { x y } battery } }"}' | jq
```

`move` and `bulkMove` also accept `reset: true` to reset before moving.

### Session Management
```bash
# List recently active sessions
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ sessions(sort: ACTION, order: DESC, limit: 5) { count total sessions { id mapName lastActionAt } } }"}' | jq

# Delete a session
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"mutation { deleteSession(id: \"SESSION_ID\") { message } }"}' | jq
```

The jq snippets below assume a state fetched into `state.json`:
```bash
curl -s http://localhost:8000/graphql -H 'Content-Type: application/json' \
  -d '{"query":"{ gameState(sessionID: \"SESSION_ID\") { playerPos { x y } visitedParks { id visited } grid { type visited id allowedDirections } } }"}' \
  | jq '.data.gameState' > state.json
```
(Non-fog sessions, or pass `grid(password: ...)` for fog.)

## Navigation Strategies

### 🗺️ Systematic World Mapping

Create ASCII representations to track understanding:
```bash
# Visual grid
jq -r '
  .playerPos as $p |
  .grid | to_entries | map(.key as $row |
    .value | to_entries | map(
      if .key == $p.x and $row == $p.y then "T"
      elif .value.type == "road" then "R"
      elif .value.type == "building" then "B"
      elif .value.type == "water" then "W"
      elif .value.type == "home" then "H"
      elif .value.type == "park" then "P"
      elif .value.type == "supercharger" then "S"
      else "?"
      end
    ) | join("")
  ) | join("\n")' state.json
```

### 🧩 Corridor Navigation Technique

Identify safe passages for efficient travel:
- **Golden Corridors**: Obstacle-free rows/columns
- **Multi-Corridor Routes**: Chain safe passages to bypass clusters
- **Perpendicular Approaches**: Try N/S vs E/W when blocked

**Find horizontal highways:**
```bash
jq '
  .grid | to_entries | map({
    row: .key,
    road_count: (.value | map(select(.type == "road")) | length)
  }) | sort_by(.road_count) | reverse | .[0:3]' state.json
```

### ⚡ Proactive Battery Management

**Find nearest chargers:**
```bash
jq '
  .playerPos as $p |
  .grid | to_entries | map(.key as $y |
  .value | to_entries |
  map(select(.value.type == "supercharger" or .value.type == "home") |
  {
    type: .value.type,
    x: .key,
    y: $y,
    distance: (((.key - $p.x)|abs) + (($y - $p.y)|abs))
  })) | flatten | sort_by(.distance)' state.json
```

**Safety principles:**
- Maintain 3+ battery buffer when far from chargers
- Recharge proactively, not reactively
- Plan routes passing through charging stations

### 🎯 Section-Based Problem Solving

- Divide large grids into manageable sections
- Complete one section fully before moving to next
- Build comprehensive maps iteratively
- Document successful routes for pattern reuse

## Proven Success Patterns

### Iterative Mastery Framework

**Phase 1 - World Analysis**
1. Map all parks, chargers, and obstacle clusters
2. Identify safe corridors and water crossings
3. Analyze building patterns for alternatives
4. Create ASCII representation of world

**Phase 2 - Route Architecture**
1. Design section-based completion strategy
2. Plan charging station utilization per segment
3. Calculate battery requirements with contingencies
4. Map alternative routes for each objective

**Phase 3 - Systematic Execution**
1. Execute planned routes using bulk moves
2. Use single moves for precise navigation around obstacles
3. Monitor battery continuously
4. Document successful routes

**Phase 4 - Adaptive Refinement**
1. Analyze failures - which obstacle pattern?
2. Apply perpendicular approach strategies
3. Update world map with new obstacle info
4. Refine techniques for remaining objectives

### Victory Optimization Techniques

**Corridor-First Approach**: Use safe passages to reach difficult objectives

**Charging Hub Strategy**: Establish strategic bases at superchargers

**Alternative Angle Mastery**: When blocked, try different approach directions

**Progressive Section Clearing**: Complete easier areas first

## Advanced Pathfinding

### Breadth-First Search (BFS)
Find shortest path:
- Start from current position
- Explore cells at distance 1, then 2, etc.
- Track parent to reconstruct path

### Battery-Aware Pathfinding
Modify BFS to consider battery:
- Track battery level at each position
- Only explore if battery > 0
- Consider chargers as battery reset points

### Multi-Objective Planning
For collecting all parks:
- Calculate distances between all objectives
- Find order minimizing total distance
- Ensure path to charger always exists

## Debugging Navigation

**Check adjacent cells:**
```bash
jq '
  .playerPos as $p |
  {
    up: .grid[$p.y - 1][$p.x],
    down: .grid[$p.y + 1][$p.x],
    left: .grid[$p.y][$p.x - 1],
    right: .grid[$p.y][$p.x + 1]
  }' state.json
```

**Find uncollected parks:**
```bash
jq '
  .grid | to_entries | map(.key as $y |
  .value | to_entries |
  map(select(.value.type == "park") |
  {
    id: .value.id,
    position: {x: .key, y: $y},
    visited: .value.visited
  })) | flatten' state.json
```

## Key Success Principles

🎯 **Systematic over Speed**: Focus on consistent, methodical completion

🗺️ **Documentation-Driven**: Maintain maps and pattern recognition

⚡ **Proactive Resources**: Charge before you need to

🧩 **Iterative Refinement**: Build on partial successes

🚀 **Corridor Navigation**: Use safe passages as primary technique

---

These strategies have achieved consistent victory across multiple configurations through systematic application of proven techniques.
