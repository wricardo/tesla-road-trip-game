# Tesla Road Trip Game - AI Agent Instructions

You are playing a Tesla Road Trip game via API. The game is a grid-based game where you control a Tesla car to collect parks while managing battery.

## GAME API ENDPOINT
Base URL: `http://localhost:8080/api`

## CURRENT GAME STATE
To check the current state, use:
```bash
curl -s http://localhost:8080/api | jq
```

The response includes:
- `player_pos`: {x, y} coordinates of the Tesla
- `battery`: current battery level (0-10 or varies by config)
- `max_battery`: maximum battery capacity
- `score`: number of parks collected
- `message`: current status message
- `grid`: 2D array representing the game board
- `visited_parks`: object tracking which parks have been collected
- `game_over`: whether the game has ended
- `victory`: whether the player has won
- `config_name`: name of the current configuration
- `save_id`: ID of the last save (if any)

## AVAILABLE ACTIONS
Send POST requests with JSON body for single or bulk operations:

### Single Movement Actions
Send `{"action":"ACTION"}` where ACTION is:
- `"up"`: Move Tesla up (y-1)
- `"down"`: Move Tesla down (y+1)  
- `"left"`: Move Tesla left (x-1)
- `"right"`: Move Tesla right (x+1)

Example single move:
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"left"}'
```

### Bulk Movement Actions
Send `{"actions":["ACTION1","ACTION2",...]}` for multiple moves with validation between each:
- Processes up to 50 moves in sequence
- Validates game state after each move
- Stops on game over or invalid moves
- Returns detailed audit trail for each move

Example bulk moves:
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"actions":["right","right","down","left"]}'
```

### WebSocket Broadcasting Control
Add `"broadcast":"MODE"` to control real-time updates:
- `"final"` (default): Broadcast only final state
- `"each"`: Broadcast after each move in bulk operations

Example with broadcasting:
```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"actions":["up","right"],"broadcast":"each"}'
```

### Game Management Actions
- `"reset"`: Start a new game with current configuration
- `"save"`: Save current game state (returns save_id)
- `"load"`: Load a saved game (requires save_id parameter)

Example save/load:
```bash
# Save game
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"save"}'

# Load game
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"load","save_id":"save_1757156572"}'
```

## AUXILIARY ENDPOINTS

### List Available Configurations
```bash
curl -s http://localhost:8080/api/configs | jq
```
Returns array of available game configurations with names and descriptions.

### List Saved Games
```bash
curl -s http://localhost:8080/api/saves | jq
```
Returns array of saved games with save_id and timestamp.

## GRID CELL TYPES
When analyzing the grid, each cell has a `type` field:
- `"road"`: Driveable path (Tesla can move here)
- `"home"`: Starting position and charging station (🏠)
- `"park"`: Collectible objective, has unique `id` field (🌳)
- `"supercharger"`: Charging station (⚡)
- `"water"`: Obstacle, cannot drive through (💧)
- `"building"`: Obstacle, cannot drive through (🏢)

## GAME RULES
1. **Movement Cost**: Each move consumes 1 battery unit
2. **Charging**: Battery fully restored when reaching home or supercharger
3. **Park Collection**: Driving over a park increases score by 1
4. **Victory Condition**: Collect all parks in the grid
5. **Defeat Condition**: Battery reaches 0 with no charging station on current tile
6. **Grid Boundaries**: Cannot move outside grid bounds
7. **Obstacles**: Cannot move into water or building tiles

## STRATEGY TIPS FOR AI AGENTS
1. **Path Planning**: Calculate shortest paths to objectives using breadth-first search
2. **Battery Management**: Always track distance to nearest charging station
3. **Efficiency**: Collect parks in clusters to minimize movement
4. **Safety Margin**: Keep enough battery to reach a charger (home or supercharger)
5. **State Analysis**: Check grid dimensions and park locations before planning routes
6. **Bulk Move Optimization**: Use bulk moves for efficient multi-step execution:
   - Plan entire routes before execution
   - Use state validation to detect failures early
   - Implement failsafe logic for battery management
   - Monitor `move_history` for debugging failed moves
7. **Move History Utilization**: 
   - `move_history` provides complete session trail data for UI visualization
   - Each entry contains `from_position` and `to_position` for path reconstruction
   - History persists across single and bulk operations
   - Use `success` field to identify failed moves in the trail

## EXAMPLE GAME FLOW
```bash
# 1. Check initial state
curl -s http://localhost:8080/api | jq '.player_pos, .battery, .score'

# 2. Find parks in grid
curl -s http://localhost:8080/api | jq '.grid[][] | select(.type=="park")'

# 3. Move towards nearest park (single moves)
curl -X POST http://localhost:8080/api -d '{"action":"right"}'
curl -X POST http://localhost:8080/api -d '{"action":"down"}'

# 3b. Or use bulk moves for efficiency
curl -X POST http://localhost:8080/api -d '{"actions":["right","down"]}'

# 4. Check battery status
curl -s http://localhost:8080/api | jq '.battery, .message'

# 5. Save progress
curl -X POST http://localhost:8080/api -d '{"action":"save"}' | jq '.save_id'

# 6. Continue collecting parks or recharge as needed
```

## ERROR HANDLING
- Invalid moves return success but don't change position
- Message field indicates why move failed ("Can't move there!")
- Game continues until victory or defeat condition met
- After game_over=true, only reset action is effective

## OPTIMAL PLAY ALGORITHM
1. Parse grid to identify all parks, charging stations
2. Build adjacency graph of valid road tiles
3. Use Dijkstra's algorithm considering battery constraints
4. Plan route that visits all parks with charging stops
5. Execute plan with periodic state verification
6. Save game after collecting each park for safety

## RESPONSE PARSING
Key fields to monitor:
```javascript
{
  "player_pos": {"x": 9, "y": 7},  // Current position
  "battery": 8,                     // Remaining battery
  "score": 3,                       // Parks collected
  "game_over": false,               // Game status
  "message": "Park visited! Score: 3",  // Last action result
  "move_history": [                 // Complete move history for the session
    {
      "action": "right",
      "from_position": {"x": 8, "y": 7},
      "to_position": {"x": 9, "y": 7},
      "battery": 9,
      "timestamp": 1757159245,
      "success": true,
      "move_number": 1
    }
  ],
  "total_moves": 4                  // Total number of moves in the session
}
```

## CONFIGURATION DIFFERENCES
Different configs have varying:
- Grid sizes (10x10, 15x15, 20x20)
- Battery capacity (8-15 units)
- Number and placement of parks
- Charging station locations
- Difficulty based on park/charger ratio

Always check `max_battery` and grid dimensions when starting!