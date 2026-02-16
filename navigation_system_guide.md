# Tesla Road Trip Game - Navigation System Guide

## Game State JSON Structure

The game provides a rich JSON API that contains all information needed for navigation and pathfinding.

### 1. Core State Fields

```json
{
  "player_pos": {"x": 8, "y": 8},      // Current position
  "battery": 22,                       // Current battery level
  "max_battery": 22,                   // Maximum battery capacity
  "score": 0,                          // Parks collected
  "visited_parks": {},                 // Which parks have been visited
  "game_over": false,                  // Game state flags
  "victory": false,
  "message": "Welcome!",               // Status messages
  "config_name": "Strategic Maze",     // Current map
  "total_moves": 0                     // Move counter
}
```

### 2. Grid System

The `grid` field is a 2D array where:
- `grid[y][x]` gives you the cell at position (x,y)
- Array is indexed [row][column] or [y][x]

Each cell has a structure:

#### Basic Cell Types

**Road Cell** (driveable):
```json
{
  "type": "road"
}
```

**Building/Water** (obstacles):
```json
{
  "type": "building"  // or "water"
}
```

**Home** (starting position + charger):
```json
{
  "type": "home"
}
```

**Supercharger** (charging station):
```json
{
  "type": "supercharger"
}
```

#### Special Cells

**Park** (collectible objective):
```json
{
  "type": "park",
  "id": "park_0"     // Unique identifier
}
```

**Visited Park**:
```json
{
  "type": "park",
  "id": "park_5",
  "visited": true    // Added after collection
}
```

### 3. Move History System

The `move_history` array tracks every move:

```json
{
  "action": "right",
  "from_position": {"x": 9, "y": 7},
  "to_position": {"x": 10, "y": 7},
  "battery": 9,
  "timestamp": 1757171272,
  "success": true,
  "move_number": 1
}
```

## Navigation Algorithms You Can Build

### 1. Cell Type Checker
```bash
# Check if a cell is driveable
curl -s http://localhost:8080/api | jq --arg x 5 --arg y 10 \
  '.grid[$y|tonumber][$x|tonumber].type' | grep -E "road|home|park|supercharger"
```

### 2. Find All Driveable Cells
```bash
# Get all road positions
curl -s http://localhost:8080/api | jq -r '
  .grid | to_entries | map(.key as $y | 
  .value | to_entries | 
  map(select(.value.type == "road" or 
             .value.type == "home" or 
             .value.type == "park" or 
             .value.type == "supercharger") | 
  {x: .key, y: $y})) | flatten'
```

### 3. Distance Calculator
```bash
# Manhattan distance between two points
curl -s http://localhost:8080/api | jq --arg tx 14 --arg ty 7 '
  .player_pos as $p | 
  (($tx|tonumber) - $p.x)|abs + (($ty|tonumber) - $p.y)|abs'
```

### 4. Find Nearest Charger
```bash
# Find all chargers with distances
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

### 5. Check Adjacent Cells
```bash
# Check what's around current position
curl -s http://localhost:8080/api | jq '
  .player_pos as $p |
  {
    up: .grid[$p.y - 1][$p.x].type,
    down: .grid[$p.y + 1][$p.x].type,
    left: .grid[$p.y][$p.x - 1].type,
    right: .grid[$p.y][$p.x + 1].type
  }'
```

### 6. Park Collection Status
```bash
# See which parks remain
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

## Path Planning Strategies

### 1. Breadth-First Search (BFS)
Best for finding shortest path considering only distance:
- Start from current position
- Explore all cells at distance 1, then 2, etc.
- Track parent of each cell to reconstruct path
- Stop when target reached

### 2. Battery-Aware Pathfinding
Modify BFS to consider battery:
- Track battery level at each position
- Only explore if battery > 0
- Consider chargers as battery reset points
- Plan routes that end at chargers

### 3. Multi-Objective Planning
For collecting all parks:
- Use Traveling Salesman Problem (TSP) approach
- Calculate distances between all objectives
- Find order that minimizes total distance
- Always ensure path to charger exists

## Key Navigation Insights

### Highway Detection
Look for long horizontal/vertical stretches of roads:
```bash
# Find horizontal highways (rows with many roads)
curl -s http://localhost:8080/api | jq '
  .grid | to_entries | map({
    row: .key,
    road_count: (.value | map(select(.type == "road")) | length)
  }) | sort_by(.road_count) | reverse | .[0:3]'
```

### Dead End Detection
Identify cells with only one exit:
```bash
# Check if current position is a dead end
curl -s http://localhost:8080/api | jq '
  .player_pos as $p |
  [
    .grid[$p.y - 1][$p.x].type,
    .grid[$p.y + 1][$p.x].type,
    .grid[$p.y][$p.x - 1].type,
    .grid[$p.y][$p.x + 1].type
  ] | map(select(. == "road" or . == "home" or . == "park" or . == "supercharger")) | length'
```

### Safe Battery Zones
Calculate areas reachable from chargers:
- Any position within battery/2 distance of a charger is "safe"
- Positions beyond max_battery from any charger are unreachable

## Advanced Techniques

### 1. State Space Search
Treat game as state space where state = (position, battery, parks_collected)

### 2. A* Algorithm
Use heuristic like: f(n) = g(n) + h(n)
- g(n) = actual battery used to reach n
- h(n) = estimated battery to goal (Manhattan distance)

### 3. Dynamic Programming
Memoize best paths between key positions to avoid recalculation

### 4. Graph Representation
Convert grid to graph where:
- Nodes = driveable cells
- Edges = valid moves
- Weights = battery cost (usually 1)

## Debugging Navigation

### Visual Grid Mapper
```bash
# Create ASCII visualization
curl -s http://localhost:8080/api | jq -r '
  .player_pos as $p |
  .grid | to_entries | map(
    .value | to_entries | map(
      if .key == $p.x and (.key|tonumber) == $p.y then "T"
      elif .value.type == "road" then "."
      elif .value.type == "building" then "#"
      elif .value.type == "water" then "~"
      elif .value.type == "home" then "H"
      elif .value.type == "park" then "P"
      elif .value.type == "supercharger" then "S"
      else "?"
      end
    ) | join("")
  ) | join("\n")'
```

This system provides everything needed for sophisticated pathfinding and game solving!