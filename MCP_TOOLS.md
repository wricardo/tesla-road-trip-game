# Tesla Road Trip Game - MCP Tools Reference

MCP is served over Streamable HTTP at `POST /mcp` (stdio is not supported). Most tools return **TOON format** (Token-Oriented Object Notation) text with snake_case fields. Exceptions: `create_map`, `update_map`, and `delete_map` return plain text (`map "name" created|updated|deleted`); `list_maps` returns a plain list of map names followed by a TOON table.

With the bundled `mcp.json` the server is named `tesla-game`, so Claude Code exposes tools as `mcp__tesla-game__<tool>`.

## Tools Summary

| Tool | Purpose | Required Params |
|------|---------|-----------------|
| `list_maps` | List available game maps/configs | (none) |
| `get_map` | Get full map definition | `name` |
| `validate_map` | Validate a map definition without saving | `map` |
| `create_map` | Create a new map (admin) | `name`, `grid_size`, `max_battery`, `starting_battery`, `layout`, `legend` |
| `update_map` | Partially update a map (admin) | `name` |
| `delete_map` | Delete a map (admin) | `name` |
| `create_session` | Create new game session | (none) |
| `get_session` | Get session details (metadata) | `session_id` |
| `update_session` | Set session display name | `id`, `display_name` |
| `delete_session` | Delete a session | `id` |
| `list_sessions` | List sessions (sortable) | (none) |
| `unified_sessions` | List sessions with full game state, optionally per map | (none) |
| `game_state` | Get current game state (position, battery, local view) | `session_id` |
| `move` | Move one step (up/down/left/right) | `session_id`, `direction` |
| `bulk_move` | Execute multiple moves at once (max 50) | `session_id`, `moves` (array) |
| `reset_game` | Reset session to initial state | `session_id` |
| `move_history` | Get paginated move history for session | `session_id` |

Coordinates: `grid[y][x]`; `right` = x+1, `down` = y+1. Each move costs 1 battery; entering home (H) or a supercharger (S) refills to max. Moving into a building, water, or off the map ends the game immediately; never probe by bumping. One-way cells (`allowed_directions`) reject wrong-way moves without cost.

---

## Tool Details

### 1. list_maps
**Purpose:** List all available game maps/configurations.

**Parameters:**
- (none)

**Response (text list + TOON):**
```
- bayou_braids
- classic
- easy
... (map names list)

[26]{description,filename,grid_size,map_id,max_battery,name}:
  Braided bayou roads with alternating bridges and narrow timing windows.,bayou_braids.json,16,bayou_braids,20,bayou_braids
  The original Tesla Road Trip game layout,classic.json,15,classic,20,Classic Layout
  ...
```

**Fields:**
- Simple list of map names (first section)
- Array of map objects with: `description`, `filename`, `grid_size`, `map_id`, `max_battery`, `name`

---

### 2. get_map
**Purpose:** Get full map details (layout, legend, battery settings, cell configs).

**Parameters:**
- `name` (string, required) - Map name/ID
- `password` (string, optional) - UI map password (required when the server's map password policy is enabled; otherwise the call returns `error: forbidden: invalid or missing map password`)

---

### 3. validate_map
**Purpose:** Validate a map definition (structure and winnability) without saving it.

**Parameters:**
- `map` (object, required) with:
  - `name`, `description` (string, required)
  - `grid_size`, `max_battery`, `starting_battery` (integer, required; `grid_size` 5-50)
  - `layout` (array of strings, required) - One row per string of R/H/P/S/W/B chars
  - `legend` (array of `{key, value}`, required) - e.g. `{"key":"R","value":"road"}`
  - `cell_configs` (array of `{key, type, allowed_directions}`, optional) - One-way cells

**Response (TOON format):**
```
valid: true
winnable: true
message: "Map \"my_map\" is solvable."
```
On failure: `valid: false`, `winnable: false`, plus `message` and `error` with the reason.

---

### 4. create_map / update_map / delete_map
**Purpose:** Create, partially update, or permanently delete a map. Require admin authorization (`ADMIN_API_KEY`, or `ALLOW_UNAUTHENTICATED_ADMIN=true` locally).

**Parameters:**
- `create_map`: `name` (unique ID, lowercase/underscores), `grid_size`, `max_battery`, `starting_battery`, `layout`, `legend` (all required); `description`, `cell_configs` (optional). Layout needs at least one P and one H.
- `update_map`: `name` (required); any of `description`, `grid_size`, `max_battery`, `starting_battery`, `layout`, `legend`, `cell_configs`. Omitted fields keep current values.
- `delete_map`: `name` (required).

`layout`, `legend`, `cell_configs` use the same shapes as `validate_map`.

**Response (plain text):** `map "my_map" created` / `map "my_map" updated` / `map "my_map" deleted`

---

### 5. create_session
**Purpose:** Create a new game session.

**Parameters (all optional):**
- `map_id` (string) - Map ID to load; wins over `map_name` when both are given
- `map_name` (string) - Map/config name. If neither is given, the server's default map is used
- `fog_enabled` (boolean) - Enable fog of war
- `fog_radius` (integer) - Fog radius when fog is enabled
- `grid_password` (string) - Password required to view the full grid of a fog session. If omitted with fog enabled, one is generated and returned once as `generated_grid_password`
- `move_delay_ms` (integer) - Per-session move delay in milliseconds (`bulk_move_delay_ms` is a deprecated alias)

**Request:**
```json
{
  "map_id": "easy",
  "fog_enabled": true,
  "fog_radius": 2,
  "move_delay_ms": 0
}
```

**Response (TOON format, fog session):**
```
created_at: "2026-10-06T17:09:33.623582-04:00"
fog_enabled: true
game_map:
  description: Beginner-friendly layout with more superchargers
  grid_size: 10
  legend:
    B: building
    H: home
    P: park
    R: road
    S: supercharger
    W: water
  max_battery: 15
  name: Easy Mode
  starting_battery: 15
game_state:
  battery: 15
  current_moves[0]:
  current_moves_count: 0
  fog_enabled: true
  fog_radius: 2
  game_over: false
  map_name: Easy Mode
  max_battery: 15
  message: Welcome! Drive your Tesla to collect parks. Watch your battery!
  move_history[0]:
  player_pos:
    x: 5
    y: 5
  reset_count: 0
  score: 0
  total_moves: 0
  victory: false
  visited_parks:
generated_grid_password: k765zzrp698k
id: 2cb4
last_action_at: "2026-10-06T17:09:33.623582-04:00"
map_name: easy
```

Non-fog sessions additionally include `game_map.layout[N]` (rows such as `BBBBBBBBBB,BRRRRSRRRB,...`) and `game_state.grid[N]`. Fog sessions never return `layout` or `grid` over MCP.

**Fields:**
- `id` - Session ID (use for all future calls with this session)
- `game_map` - Map config metadata
- `game_state` - Initial game state
- `generated_grid_password` - Only when fog is enabled and no `grid_password` was given
- `created_at`, `last_action_at` - Timestamps

---

### 6. get_session
**Purpose:** Get session metadata (map info, timestamps). Does NOT return current game state — use `game_state` for that.

**Parameters:**
- `session_id` (string, required) - Session ID

**Response (TOON format):**
```
created_at: "2026-10-06T17:09:30.109364-04:00"
game_map:
  description: Beginner-friendly layout with more superchargers
  grid_size: 10
  layout[10]: BBBBBBBBBB,BRRRRSRRRB,BRPRRRRPRB,...
  legend: {...}
  max_battery: 15
  name: Easy Mode
  starting_battery: 15
id: cab8
last_action_at: "2026-10-06T17:09:33.571763-04:00"
map_name: easy
```

---

### 7. update_session
**Purpose:** Update session metadata (display name).

**Parameters:**
- `id` (string, required) - Session ID
- `display_name` (string, required) - New display name

**Response:** Full session in TOON (as `create_session`) with `display_name` set.

---

### 8. delete_session
**Purpose:** Delete a session.

**Parameters:**
- `id` (string, required) - Session ID

**Response (TOON format):**
```
message: Session cab8 deleted
```

---

### 9. list_sessions
**Purpose:** List active sessions.

**Parameters:**
- `sort` (string, optional) - `CREATED` or `ACTION` (last action time)
- `order` (string, optional) - `ASC` or `DESC`
- `limit` (integer, optional) - Max sessions to return

**Response (TOON format):**
```
count: 1
order: desc
sessions[1]:
  - created_at: "2026-10-06T17:09:30.109364-04:00"
    game_map:
      description: Beginner-friendly layout with more superchargers
      grid_size: 10
      ...
    id: cab8
    last_action_at: "2026-10-06T17:09:33.571763-04:00"
    map_name: easy
sort: action
total: 58
```

**Fields:**
- `count` - Sessions returned; `total` - Sessions that exist
- `sort`, `order` - Applied sort
- `sessions` - Session objects (same structure as `get_session`)

---

### 10. unified_sessions
**Purpose:** List sessions including their game state, optionally filtered by map.

**Parameters:**
- `map_name` (string, optional) - Only sessions on this map

**Response (TOON format):** `count`, `map_name`, and `sessions[N]` where each session includes `game_map` and `game_state` (with `current_moves`).

---

### 11. game_state
**Purpose:** Get current game state (player position, battery, local view, visited parks).

**Parameters:**
- `session_id` (string, required) - Session ID
- `grid` (boolean, optional) - Include the full grid (`grid[y][x]`). Ignored for fog sessions

**Response (TOON format, fog session with `fog_radius: 2`):**
```
battery: 15
battery_risk: SAFE
fog_enabled: true
fog_radius: 2
game_over: false
local_view_3x3[3]: HHR,HTR,RRR
map_name: Easy Mode
max_battery: 15
message: Welcome! Drive your Tesla to collect parks. Watch your battery!
player_pos:
  x: 5
  y: 5
reset_count: 0
score: 0
total_moves: 0
victory: false
visited_parks:
```

Without `grid: true`, non-fog sessions show `grid: null`. With `grid: true` the grid is returned row by row:
```
grid[10]:
  - [10]{type}:
    building
    building
    ...
  - [10]:
    - type: building
    - type: road
    - id: park_0
      type: park
    ...
```

**Fields:**
- `battery`, `max_battery` - Current/max battery
- `player_pos` - Current position (x, y)
- `local_view_3x3` - Fixed 3x3 view around the car regardless of `fog_radius`: row 0 = y-1, column 0 = x-1, `T` = car at `player_pos`, off-map = `B`, other letters as in the map legend
- `fog_enabled`, `fog_radius` - Fog settings
- `grid` - Only when `grid: true` on a non-fog session; cells have `type` and, for parks, `id`
- `visited_parks` - Object of visited park IDs (empty if none visited)
- `score` - Count of parks visited
- `victory` - Game won?
- `game_over` - Game lost? (out of battery, crashed)
- `battery_risk` - `SAFE`, `LOW` (battery <= 1/3 max), `CAUTION` (battery <= distance to nearest charger + 2), `DANGER` (battery <= that distance), `CRITICAL` (0), `WARNING` (no charger on map), `UNKNOWN`. Distance is Manhattan distance ignoring walls and one-way rules: a heuristic, not reachability

---

### 12. move
**Purpose:** Move Tesla one step in a direction.

**Parameters:**
- `session_id` (string, required) - Session ID
- `direction` (string, required) - Direction: "up", "down", "left", "right"
- `reset` (boolean, optional) - Reset session before move (saves API call on retry)
- `intent` (string, optional) - Explain reasoning (for logging/analysis)

**Request:**
```json
{
  "session_id": "cab8",
  "direction": "right",
  "reset": false,
  "intent": "Moving towards nearest park"
}
```

**Response (TOON format):**
```
game_state:
  battery: 14
  battery_risk: SAFE
  fog_radius: 1
  game_over: false
  grid: null
  local_view_3x3[3]: HRR,HTR,RRR
  ...
  player_pos:
    x: 6
    y: 5
  ...
message: "Battery: 14/15"
step:
  battery_after: 14
  battery_before: 15
  dir: right
  from:
    x: 5
    y: 5
  idx: 1
  success: true
  tile_char: R
  tile_type: road
  to:
    x: 6
    y: 5
success: true
```

A move into a blocked tile reports the target in `attempted_to` instead of `step`:
```
attempted_to:
  passable: false
  tile_char: B
  tile_type: building
  x: 8
  y: 5
```
(Crashing into a building/water/off-map ends the game.)

**Fields:**
- `success` - Move succeeded?
- `message` - Outcome message (charge, park visit, game over, etc.)
- `step` - The executed step: from/to, tile, battery before/after
- `attempted_to` - Target tile of a failed move
- `game_state` - Updated game state after move (same shape as `game_state`)

---

### 13. bulk_move
**Purpose:** Execute multiple moves at once (efficient for known safe paths). Stops at the first failed move. At most 50 moves are executed; extra moves are dropped and the result has `truncated: true` and `limit: 50`.

**Parameters:**
- `session_id` (string, required) - Session ID
- `moves` (array of strings, required) - Array of directions ["up", "down", "left", "right"]
- `reset` (boolean, optional) - Reset before executing moves
- `intent` (string, optional) - Explain reasoning

**Request:**
```json
{
  "session_id": "cab8",
  "moves": ["up", "up"],
  "reset": false,
  "intent": "Navigate towards park at (7,2)"
}
```

**Response (TOON format):**
```
battery_risk: SAFE
end_battery: 12
end_pos:
  x: 6
  y: 3
events: null
game_over: false
game_state: {...}  (final game state)
local_view_3x3[3]: RRP,RTR,HRR
message: "Battery: 12/15"
moves_executed: 2
possible_moves[4]: up,down,left,right
requested_moves: 2
score_delta: 0
start_battery: 14
start_pos:
  x: 6
  y: 5
steps[2]:
  - battery_after: 13
    battery_before: 14
    dir: up
    from:
      x: 6
      y: 5
    idx: 1
    success: true
    tile_char: R
    tile_type: road
    to:
      x: 6
      y: 4
  - battery_after: 12
    ...
success: true
total_moves: 2
```

**Fields:**
- `moves_executed` vs `requested_moves` - Shows if all moves completed
- `truncated`, `limit` - Present when more than 50 moves were sent
- `steps` - Per-move details (same shape as `move`'s `step`)
- `start_pos`, `end_pos` - Before/after positions
- `start_battery`, `end_battery` - Battery change
- `score_delta` - Parks collected during this call
- `possible_moves` - Directions you can legally move from the end position (walls, map edge, one-way roads, battery considered); empty after game over
- `local_view_3x3` - 3x3 view at the end position
- `game_state` - Final state after all moves
- `success` - All moves succeeded?

---

### 14. reset_game
**Purpose:** Reset session to initial state (starting position, battery, no visited parks).

**Parameters:**
- `session_id` (string, required) - Session ID

**Response (TOON format):**
```
battery: 15
battery_risk: SAFE
fog_radius: 1
game_over: false
grid: null
local_view_3x3[3]: HHR,HTR,RRR
map_name: Easy Mode
max_battery: 15
message: Welcome! Drive your Tesla to collect parks. Watch your battery!
player_pos:
  x: 5
  y: 5
reset_count: 1
score: 0
total_moves: 3
victory: false
visited_parks:
```

**Fields:**
- Same as `game_state`, with position, battery, and visited parks reset
- `reset_count` increments; `total_moves` keeps counting across resets

---

### 15. move_history
**Purpose:** Get paginated move history for a session.

**Parameters:**
- `session_id` (string, required) - Session ID
- `page` (integer, optional) - Page number (default: 1)
- `limit` (integer, optional) - Moves per page (default: 50)
- `order` (string, optional) - `ASC` or `DESC` (default: `DESC`, newest first)

**Request:**
```json
{
  "session_id": "cab8",
  "limit": 2
}
```

**Response (TOON format):**
```
has_next: true
has_previous: false
moves[2]:
  - action: up
    battery: 12
    from_position:
      x: 6
      y: 4
    move_number: 3
    success: true
    timestamp: 1791320973
    to_position:
      x: 6
      y: 3
  - action: up
    battery: 13
    ...
page: 1
page_size: 2
total_moves: 3
total_pages: 2
```

**Fields:**
- `moves` - Array of move records
- `page`, `page_size`, `total_pages`, `total_moves` - Pagination info
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

A move made after the game ended returns `success: false` with a message such as `Game is already over (...). No move was made — call reset (or pass reset: true) to play again.`

Common errors:
- Session not found - Session ID invalid
- Invalid direction - Direction not up/down/left/right
- Map not found - Map name invalid
- `forbidden: invalid or missing map password` - `get_map` without the required map password
