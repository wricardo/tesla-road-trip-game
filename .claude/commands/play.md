---
allowed-tools: mcp__tesla-game__*, Write, Read, Edit
description: Play the Tesla Road Trip game strategically
argument-hint: [optional: specific goal like "collect park_3" or "reach supercharger"]
---

# Play Tesla Road Trip Game

You are playing a grid-based navigation game where you control a Tesla to collect all parks while managing battery. Use the MCP game tools to interact with the game.

## Game Mechanics
- Movement costs 1 battery per move (up, down, left, right); reaching 0 battery away from a charger ends the run
- Homes (H) and Superchargers (S) restore battery to maximum as soon as you enter them
- Parks (P) are objectives to collect - you win by collecting all of them
- Buildings (B) and Water (W) are impassable; moving into them (or off the map) ends the game immediately
- Roads (R) are passable paths
- One-way cells: custom glyphs defined in the map's `cell_configs` (see `mcp__tesla-game__get_map`) carry `allowed_directions`. A move is allowed only if its direction is listed for both the cell you leave and the cell you enter (when those lists are non-empty). Wrong-way moves are rejected without costing battery.

## Strategic Approach

### 1. Initial Assessment
If you don't have a session yet, pick a map with `mcp__tesla-game__list_maps` and start one with `mcp__tesla-game__create_session map_id:"<map>"`. Keep the returned `session_id`; every gameplay tool requires it.

Then use `mcp__tesla-game__game_state session_id:"<id>" grid:true` to understand:
- Current position and battery level
- Which parks have been collected
- Grid layout and obstacles (`grid[y][x]`; RIGHT = x+1, DOWN = y+1)

In fog sessions the full grid is never returned; rely on the 3x3 `local_view_3x3` (row 0 = y-1, col 0 = x-1, `T` = car, off-map = `B`) in each `game_state`/`move` result.

### 2. Planning Phase
Create a scratchpad file (`game_plan.txt`) to track:
- Park locations and collection status
- Identified safe paths between key locations
- Battery management checkpoints (homes and superchargers)
- Obstacle patterns to avoid

### 3. Path Validation Strategy
Before executing moves:
- Trace your planned path on the grid
- Count battery consumption vs available battery
- Identify nearest charging stations along the route
- Plan escape routes if battery runs low

### 4. Movement Execution
- Use `mcp__tesla-game__move` for single careful moves when navigating tight spaces
- Use `mcp__tesla-game__bulk_move` for known safe paths (max 50 moves per call; extras are dropped with `truncated: true`)
- Both commands support optional `reset: true` to restart before moving (saves API calls)
- NEVER probe by bumping: moving into a building, water, or off the map ends the game
- Check position after each bulk move to ensure you're where expected

### 5. Battery Management
Key principle: Never venture far from charging without a plan
- Rows or columns lined with homes provide free charging while you travel along them
- Superchargers are strategic hubs - use them to explore nearby areas
- Plan routes that pass through charging stations when possible
- Keep a battery reserve for emergencies (don't go below 3 if far from charging)

### 6. Collision Prevention
Common navigation errors to avoid:
- Water barriers often leave only a few crossing points - find them before planning cross-section routes
- Buildings create mazes - trace paths carefully
- Grid boundaries - don't try to move outside the grid
- Some rows/columns have limited access points
- Superchargers placed near the center of a region serve as strategic hubs
- **Trapped parks**: A park walled in by buildings on some sides usually has a single approach - check all four neighbors
- **Building clusters**: Dense building areas form mazes - always verify paths through them

### 7. Optimal Collection Order
Consider grouping parks by proximity to charging:
- Parks near each supercharger form natural clusters
- Collect nearby parks before moving to distant areas
- Use a hub-and-spoke model with superchargers as hubs

### 8. Recovery from Mistakes
If you hit an obstacle or run out of battery:
- Use `mcp__tesla-game__reset_game session_id:"<id>"` to start over
- Update your scratchpad with what went wrong
- Adjust your route to avoid the same mistake

## Execution Guidelines

1. Start by examining the current game state
2. Create or update your planning scratchpad
3. Identify the next objective based on current position and battery
4. Plan and validate the path to that objective
5. Execute moves carefully, checking position after each sequence
6. Update your progress tracking after each park collection

## Command Examples

### Start a Session
```
mcp__tesla-game__list_maps
# Returns: available maps

mcp__tesla-game__create_session map_id:"easy"
# Returns: session_id and initial state
```

### View Current State
```
mcp__tesla-game__game_state session_id:"<id>" grid:true
# Returns: position (x,y), battery, collected parks, local_view_3x3, and the full grid (omitted in fog sessions)
```

### Single Move
```
mcp__tesla-game__move session_id:"<id>" direction:"right"
# Moves player one cell right, consumes 1 battery
# Returns: Updated state with new position and local_view_3x3

mcp__tesla-game__move session_id:"<id>" direction:"right" reset:true
# Resets game first, then moves right (saves reset + move API calls)
```

### Bulk Movement
```
mcp__tesla-game__bulk_move session_id:"<id>" moves:["right", "right", "up", "up", "left"]
# Executes moves in sequence, stops at the first failure
# Returns: Final position, successful/failed moves, possible_moves (legal directions from the end position: walls, map edge, one-way roads, battery considered; empty after game over)

mcp__tesla-game__bulk_move session_id:"<id>" moves:["right", "left"] reset:true
# Resets game first, then executes bulk moves (saves reset + bulk move API calls)
```

### Reset Game
```
mcp__tesla-game__reset_game session_id:"<id>"
# Returns: Fresh game state at the starting position
```

### Session and Map Information
```
mcp__tesla-game__get_session session_id:"<id>"
# Returns: Session details (map, fog settings, state)
```

## Available MCP Game Tools
- `mcp__tesla-game__list_maps` - List available maps
- `mcp__tesla-game__create_session` - Start a new session (`map_id`, optional fog settings)
- `mcp__tesla-game__game_state` - View current game status (`session_id`, optional `grid`)
- `mcp__tesla-game__move` - Single move (`session_id`, `direction`: up/down/left/right)
- `mcp__tesla-game__bulk_move` - Execute multiple moves (`session_id`, `moves`)
- `mcp__tesla-game__reset_game` - Start over (`session_id`)
- `mcp__tesla-game__get_session` - Session details (`session_id`)

Remember: Patience and careful planning beat speed. Validate paths before execution!

$ARGUMENTS