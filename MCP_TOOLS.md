# Tesla Road Trip Game - MCP Tools Reference

All MCP tools return responses in **TOON format** (Token-Oriented Object Notation). Output is compact and token-efficient.

## Tools Summary

| Tool | Purpose | Required Params |
|------|---------|-----------------|
| `list_maps` | List available game maps/configs | (none) |
| `create_session` | Create new game session | `map_name` |
| `get_session` | Get session details (metadata) | `session_id` |
| `list_sessions` | List all active sessions | (none) |
| `game_state` | Get current game state (grid, position, battery) | `session_id` |
| `move` | Move one step (up/down/left/right) | `session_id`, `direction` |
| `bulk_move` | Execute multiple moves at once | `session_id`, `moves` (array) |
| `reset_game` | Reset session to initial state | `session_id` |
| `move_history` | Get move history for session | `session_id` |

---

## Tool Details

### 1. list_maps
**Purpose:** List all available game maps/configurations.

**Parameters:**
- (none)

**Response (TOON format):**
```
- classic
- easy
- easy_circuit
... (map names list)

[N]{description,filename,grid_size,map_id,max_battery,name}:
  The original Tesla Road Trip..., classic.json, 15, classic, 20, Classic Layout
  ...
```

**Fields:**
- Simple list of map names (first section)
- Array of map objects with: `description`, `filename`, `grid_size`, `map_id`, `max_battery`, `name`

---

### 2. create_session
**Purpose:** Create a new game session.

**Parameters:**
- `map_name` (string, required) - Map config name (e.g., "easy", "medium_downtown", "strategic")

**Request:**
```json
{
  "session_id": "...",
  "map_name": "easy"
}
```

**Response (TOON format):**
```
created_at: "2026-05-22T05:57:22.482315-04:00"
game_map:
  description: Beginner-friendly layout with more superchargers
  grid_size: 10
  layout[10]: BBBBBBBBBB,BRRRRSRRRB,...
  legend:
    B: building
    H: home
    P: park
    R: road
    S: supercharger
    W: water
  max_battery: 15
  starting_battery: 15
  ...
game_state:
  battery: 15
  grid[10]: ...
  player_pos:
    x: 5
    y: 5
  score: 0
  victory: false
  visited_parks: (empty)
id: edbb
last_accessed_at: "2026-05-22T05:57:22.482315-04:00"
map_name: easy
```

**Fields:**
- `id` - Session ID (use for all future calls with this session)
- `game_map` - Map config metadata
- `game_state` - Initial game state
- `created_at`, `last_accessed_at` - Timestamps

---

### 3. get_session
**Purpose:** Get session metadata (config, map info, timestamps). Does NOT return current game state — use `game_state` for that.

**Parameters:**
- `session_id` (string, required) - Session ID

**Response (TOON format):**
```
created_at: "2026-05-22T05:57:22.482315-04:00"
game_map:
  description: Beginner-friendly layout with more superchargers
  grid_size: 10
  layout[10]: BBBBBBBBBB,...
  legend: {...}
  max_battery: 15
  name: "Easy Mode"
  ...
id: edbb
last_accessed_at: "2026-05-22T05:57:22.482315-04:00"
map_name: easy
```

**Fields:**
- Session metadata, map config, timestamps
- Does NOT include current game state (position, battery, visited parks)

---

### 4. list_sessions
**Purpose:** List all active sessions.

**Parameters:**
- (none)

**Response (TOON format):**
```
[N]:
  - created_at: "2026-05-22T05:57:22.482315-04:00"
    game_map:
      description: Beginner-friendly layout...
      grid_size: 10
      ...
    id: edbb
    last_accessed_at: "2026-05-22T05:57:22.482315-04:00"
    map_name: easy
  - created_at: "..."
    ...
```

**Fields:**
- Array of session objects (same structure as `get_session`)

---

### 5. game_state
**Purpose:** Get current game state (player position, battery, grid, visited parks).

**Parameters:**
- `session_id` (string, required) - Session ID

**Response (TOON format):**
```
battery: 14
battery_risk: SAFE
current_moves[1]:
  - action: right
    battery: 14
    from_position:
      x: 5
      y: 5
    move_number: 1
    success: true
    timestamp: 1779443842
    to_position:
      x: 6
      y: 5
current_moves_count: 1
game_over: false
grid[10]:
  - [10]{type}:
    building
    building
    ...
  - [10]{type}:
    building
    road
    ...
map_name: Easy Mode
max_battery: 15
message: "Moved right to (6,5)"
move_history[4]: (limited history)
  - action: right
    ...
player_pos:
  x: 6
  y: 5
score: 0
total_moves: 1
victory: false
visited_parks: (empty object)
```

**Fields:**
- `battery`, `max_battery` - Current/max battery
- `player_pos` - Current position (x, y)
- `grid` - 2D grid with cell types (building, road, park, home, supercharger, water)
- `current_moves` - Moves in current session
- `move_history` - Limited history of past moves
- `visited_parks` - Object of visited park IDs (empty if none visited)
- `score` - Count of parks visited
- `victory` - Game won?
- `game_over` - Game lost? (out of battery, hit wall)
- `battery_risk` - Risk level: SAFE, WARNING, CRITICAL

---

### 6. move
**Purpose:** Move Tesla one step in a direction.

**Parameters:**
- `session_id` (string, required) - Session ID
- `direction` (string, required) - Direction: "up", "down", "left", "right"
- `reset` (boolean, optional) - Reset session before move (saves API call on retry)
- `intent` (string, optional) - Explain reasoning (for logging/analysis)

**Request:**
```json
{
  "session_id": "edbb",
  "direction": "right",
  "reset": false,
  "intent": "Moving towards nearest park"
}
```

**Response (TOON format):**
```
events[1]:
  - message: "Moved right to (6,5)"
    position:
      x: 6
      y: 5
    timestamp: "2026-05-22T05:57:22.555801-04:00"
    type: move
game_state: {...}  (full game state after move)
success: true
```

**Fields:**
- `success` - Move succeeded?
- `events` - Array of events (move, charge, park visit, game over, etc.)
- `game_state` - Updated game state after move

---

### 7. bulk_move
**Purpose:** Execute multiple moves at once (efficient for known safe paths).

**Parameters:**
- `session_id` (string, required) - Session ID
- `moves` (array of strings, required) - Array of directions ["up", "down", "left", "right"]
- `reset` (boolean, optional) - Reset before executing moves
- `intent` (string, optional) - Explain reasoning

**Request:**
```json
{
  "session_id": "edbb",
  "moves": ["up", "up", "left"],
  "reset": false,
  "intent": "Navigate towards supercharger at (4,3)"
}
```

**Response (TOON format):**
```
battery_risk: SAFE
end_battery: 15
end_pos:
  x: 5
  y: 3
events[5]:
  - message: "Moved up to (6,4)"
    position: {...}
    timestamp: "..."
    type: move
  - message: Battery charged to 15/15
    position: {...}
    timestamp: "..."
    type: charge
  - message: "Moved up to (6,3)"
    ...
  - message: "Moved left to (5,3)"
    ...
game_state: {...}  (final game state)
moves_executed: 3
moves_requested: 3
start_battery: 15
start_pos:
  x: 6
  y: 5
success: true
```

**Fields:**
- `moves_executed` vs `moves_requested` - Shows if all moves completed
- `start_pos`, `end_pos` - Before/after positions
- `start_battery`, `end_battery` - Battery change
- `events` - Array of move/charge/victory/obstacle events
- `game_state` - Final state after all moves
- `success` - All moves succeeded?

---

### 8. reset_game
**Purpose:** Reset session to initial state (starting position, battery, no moves).

**Parameters:**
- `session_id` (string, required) - Session ID

**Response (TOON format):**
```
battery: 15
battery_risk: SAFE
current_moves[0]:  (empty)
current_moves_count: 0
game_over: false
grid[10]: ...
map_name: Easy Mode
max_battery: 15
message: "Game reset!"
move_history[0]:  (empty)
player_pos:
  x: 5
  y: 5
score: 0
total_moves: 0
victory: false
visited_parks: (empty)
```

**Fields:**
- Same as `game_state`, but with all progress cleared
- Position reset to start
- Battery reset to max
- Move history cleared
- Visited parks cleared

---

### 9. move_history
**Purpose:** Get paginated move history for a session.

**Parameters:**
- `session_id` (string, required) - Session ID
- `limit` (integer, optional) - Max moves to return (default: all)

**Request:**
```json
{
  "session_id": "edbb",
  "limit": 5
}
```

**Response (TOON format):**
```
has_next: false
has_previous: false
moves[4]:
  - action: left
    battery: 15
    from_position:
      x: 6
      y: 3
    move_number: 4
    success: true
    timestamp: 1779443842
    to_position:
      x: 5
      y: 3
  - action: up
    battery: 14
    from_position:
      x: 6
      y: 4
    move_number: 3
    success: true
    timestamp: 1779443842
    to_position:
      x: 6
      y: 3
  - action: up
    battery: 15
    ...
  - action: right
    ...
```

**Fields:**
- `moves` - Array of move records
- `has_next`, `has_previous` - Pagination flags
- Each move has: `action`, `from_position`, `to_position`, `battery`, `timestamp`, `move_number`, `success`

---

## TOON Format Notes

- **Compact array notation:** `[N]{key1,key2,key3}:` means N items with these keys
- **Nested objects** shown with indentation
- **String values** unquoted where unambiguous
- **No commas** between items in compact arrays
- **More token-efficient** than JSON (typically 30-40% smaller)

## Error Handling

If a tool call fails, you'll receive:
```
error: <error message>
```

Common errors:
- "session not found" - Session ID invalid
- "invalid direction" - Direction not up/down/left/right
- "map not found" - Map name invalid
- "out of battery" - Battery depleted, game over
- "can't move there" - Hit wall/building
