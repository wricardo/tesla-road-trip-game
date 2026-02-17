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
4. **Test with exploratory moves** - If uncertain, try moving to verify

## Game API Reference

### Base URL
```
http://localhost:8080/api
```

### Get Current State
```bash
curl -s http://localhost:8080/api | jq
```

**Response includes:**
- `player_pos`: {x, y} coordinates
- `battery`: Current battery level
- `max_battery`: Maximum capacity
- `score`: Parks collected
- `grid`: 2D array of game board
- `visited_parks`: Collected park tracking
- `game_over` / `victory`: Game state flags
- `move_history`: Complete move trail

### Single Move
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"right"}'
```

Actions: `up`, `down`, `left`, `right`

### Bulk Moves
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"actions":["right","right","down","left"]}'
```

- Processes up to 50 moves in sequence
- Validates state after each move
- Returns detailed audit trail

### Reset with Move
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"up","reset":true}'
```

Combines reset + move in one API call.

### Session Management
```bash
# Create session
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"config_name":"easy"}'

# Get session state
curl http://localhost:8080/api?sessionId=a3x7

# Move in session
curl -X POST http://localhost:8080/api?sessionId=a3x7 \
  -d '{"action":"right"}'
```

## Navigation Strategies

### 🗺️ Systematic World Mapping

Create ASCII representations to track understanding:
```bash
# Visual grid
curl -s http://localhost:8080/api | jq -r '
  .player_pos as $p |
  .grid | to_entries | map(
    .value | to_entries | map(
      if .key == $p.x then "T"
      elif .value.type == "road" then "R"
      elif .value.type == "building" then "B"
      elif .value.type == "water" then "W"
      elif .value.type == "home" then "H"
      elif .value.type == "park" then "P"
      elif .value.type == "supercharger" then "S"
      else "?"
      end
    ) | join("")
  ) | join("\n")'
```

### 🧩 Corridor Navigation Technique

Identify safe passages for efficient travel:
- **Golden Corridors**: Obstacle-free rows/columns
- **Multi-Corridor Routes**: Chain safe passages to bypass clusters
- **Perpendicular Approaches**: Try N/S vs E/W when blocked

**Find horizontal highways:**
```bash
curl -s http://localhost:8080/api | jq '
  .grid | to_entries | map({
    row: .key,
    road_count: (.value | map(select(.type == "road")) | length)
  }) | sort_by(.road_count) | reverse | .[0:3]'
```

### ⚡ Proactive Battery Management

**Find nearest chargers:**
```bash
curl -s http://localhost:8080/api | jq '
  .player_pos as $p |
  .grid | to_entries | map(.key as $y |
  .value | to_entries |
  map(select(.value.type == "supercharger" or .value.type == "home") |
  {
    type: .value.type,
    x: .key,
    y: $y,
    distance: ((.key - $p.x)|abs + (($y|tonumber) - $p.y)|abs)
  })) | flatten | sort_by(.distance)'
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
curl -s http://localhost:8080/api | jq '
  .player_pos as $p |
  {
    up: .grid[$p.y - 1][$p.x].type,
    down: .grid[$p.y + 1][$p.x].type,
    left: .grid[$p.y][$p.x - 1].type,
    right: .grid[$p.y][$p.x + 1].type
  }'
```

**Find uncollected parks:**
```bash
curl -s http://localhost:8080/api | jq '
  .visited_parks as $v |
  .grid | to_entries | map(.key as $y |
  .value | to_entries |
  map(select(.value.type == "park") |
  {
    id: .value.id,
    position: {x: .key, y: $y},
    visited: ($v[.value.id] // false)
  })) | flatten'
```

## Key Success Principles

🎯 **Systematic over Speed**: Focus on consistent, methodical completion

🗺️ **Documentation-Driven**: Maintain maps and pattern recognition

⚡ **Proactive Resources**: Charge before you need to

🧩 **Iterative Refinement**: Build on partial successes

🚀 **Corridor Navigation**: Use safe passages as primary technique

---

These strategies have achieved consistent victory across multiple configurations through systematic application of proven techniques.
