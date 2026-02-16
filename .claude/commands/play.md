---
allowed-tools: mcp__gamemcp__*, Write, Read, Edit
description: Play the Tesla Road Trip game strategically
argument-hint: [optional: specific goal like "collect park_3" or "reach supercharger"]
---

# Play Tesla Road Trip Game

You are playing a grid-based navigation game where you control a Tesla to collect all parks while managing battery. Use the MCP game tools to interact with the game.

## Game Mechanics
- Movement costs 1 battery per move (up, down, left, right)
- Homes (H) and Superchargers (S) restore battery to maximum (20)
- Parks (0-9) are objectives to collect - you win by collecting all 10
- Buildings (B) and Water (W) are impassable obstacles
- Roads (.) are passable paths

## Strategic Approach

### 1. Initial Assessment
First, use `mcp__gamemcp__game_state` to understand:
- Current position and battery level
- Which parks have been collected
- Grid layout and obstacles

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
- Use `mcp__gamemcp__move` for single careful moves when navigating tight spaces
- Use `mcp__gamemcp__bulk_move` for known safe paths
- Both commands support optional `reset: true` to restart before moving (saves API calls)
- ALWAYS verify you're not moving into obstacles (buildings/water)
- Check grid position after each bulk move to ensure you're where expected

### 5. Battery Management
Key principle: Never venture far from charging without a plan
- Homes on row 7 provide free charging while moving horizontally
- Superchargers are strategic hubs - use them to explore nearby areas
- Plan routes that pass through charging stations when possible
- Keep a battery reserve for emergencies (don't go below 3 if far from charging)

### 6. Collision Prevention
Common navigation errors to avoid:
- Water barriers at rows 6 and 8 (columns 5-9) - **ONLY column 4** can cross between sections
- Buildings create mazes - trace paths carefully
- Grid boundaries - don't try to move outside 0-14 coordinates
- Some rows/columns have limited access points
- The center column (7) has superchargers that serve as strategic hubs
- **Park 3 trap**: Park 3 at (11,3) is surrounded by buildings at (10,3) and (12,3). Access it from row 3's roads (columns 5-9) by going right twice from column 9
- **Building clusters**: Rows 2-4 have complex building patterns - always verify paths through these areas

### 7. Optimal Collection Order
Consider grouping parks by proximity to charging:
- Parks near each supercharger form natural clusters
- Collect nearby parks before moving to distant areas
- Use a hub-and-spoke model with superchargers as hubs

### 8. Recovery from Mistakes
If you hit an obstacle or run out of battery:
- Use `mcp__gamemcp__reset_game` to start over
- Update your scratchpad with what went wrong
- Adjust your route to avoid the same mistake

## Execution Guidelines

1. Start by examining the current game state
2. Create or update your planning scratchpad
3. Identify the next objective based on current position and battery
4. Plan and validate the path to that objective
5. Execute moves carefully, checking position after each sequence
6. Update your progress tracking after each park collection

## Quick Reference Card

### Safe Routes (from center row 7):
- To upper grid: move to col 4 → up to row 5 → navigate (column 4 is the ONLY crossing point!)
- To lower grid: move to col 4 → down to row 9 → navigate  
- East-West corridor: Row 7 has homes (cols 5-9) for free charging while traveling
- **Park 3 route**: From upper area, use row 3's roads (cols 5-9), then go right to reach Park 3 at (11,3)

### Park Collection Priority (Optimal Route):
1. **Row 7 Parks**: Start at (9,7) → Park 5 at (14,7) → Park 4 at (0,7) [uses homes for charging]
2. **Upper section via column 4**: Parks 0-1-3 first (Park 3 requires special approach)
3. **Park 2**: Requires backtracking through supercharger at (7,2)
4. **Lower section**: Parks 6-7-8-9 via lower supercharger at (7,12)

### Battery Math:
- Distance to nearest charger = minimum battery needed + 2 buffer
- Max safe exploration radius from charger = 9 moves
- Critical battery level = 3 (immediate return to charging)

## Command Examples

### View Current State
```
mcp__gamemcp__game_state
# Returns: Grid display, position (x,y), battery level, collected parks
```

### Single Move
```
mcp__gamemcp__move direction:"right"
# Moves player one cell right, consumes 1 battery
# Returns: Updated grid with new position

mcp__gamemcp__move direction:"right" reset:true
# Resets game first, then moves right (saves reset + move API calls)
```

### Bulk Movement
```
mcp__gamemcp__bulk_move moves:["right", "right", "up", "up", "left"]
# Executes 5 moves in sequence, stops if obstacle hit
# Returns: Final position after all moves, shows successful/failed moves

mcp__gamemcp__bulk_move moves:["right", "left"] reset:true
# Resets game first, then executes bulk moves (saves reset + bulk move API calls)
```

### Reset Game
```
mcp__gamemcp__reset_game
# Returns: Fresh game state with starting position (0,7)
```

### Game Information
```
mcp__gamemcp__game_info
# Returns: Configuration details, grid size, battery limits
```

## Available MCP Game Tools
- `mcp__gamemcp__game_state` - View current game status
- `mcp__gamemcp__move` - Single move (up/down/left/right)
- `mcp__gamemcp__bulk_move` - Execute multiple moves
- `mcp__gamemcp__reset_game` - Start over
- `mcp__gamemcp__game_info` - Get game configuration details

Remember: Patience and careful planning beat speed. Validate paths before execution!

$ARGUMENTS