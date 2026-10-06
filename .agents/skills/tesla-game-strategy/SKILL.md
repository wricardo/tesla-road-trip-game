---
name: tesla-game-strategy
description: Strategic guidance for playing the Tesla Road Trip grid-based navigation game. Use when the user asks to play the Tesla game, needs help with game strategy, wants to understand game mechanics, or requests assistance with navigation and resource management. Provides proven techniques for pathfinding, battery management, and systematic completion without revealing specific solutions.
---

# Tesla Road Trip Game Strategy

Strategic guidance for mastering the Tesla Road Trip game through intelligent planning, resource management, and optimal pathfinding.

## Game Mechanics

### Cell Types
- **R (Road)**: Passable paths - standard movement costs 1 battery
- **H (Home)**: Passable charging station - restores battery to maximum
- **S (Supercharger)**: Passable charging hub - restores battery to maximum
- **P (Park)**: Passable objective - collect all parks to win
- **B (Building)**: Impassable obstacle - blocks movement
- **W (Water)**: Impassable barrier - blocks movement
- **T (Tesla)**: Your current position (in MCP `local_view_3x3`)
- **Custom one-way glyphs**: Defined in the map's cell configs (`cellConfigs` in GraphQL, `cell_configs` via MCP `get_map`) with allowed directions

### Movement Rules
- Grid is row-major `grid[y][x]`; RIGHT = x+1, DOWN = y+1
- Moving into B, W, or off the map ends the game immediately - never probe by bumping
- **One-way rule**: a move is allowed only if its direction (UP/DOWN/LEFT/RIGHT) is listed in `allowedDirections` of both the cell you leave and the cell you enter, when those lists are non-empty. Wrong-way moves are rejected, cost no battery, and do not end the game. Always read `allowedDirections` together with cell `type`.

### Resource Management
- Each move costs 1 battery
- Reaching 0 battery away from a charger ends the run
- Entering a charging station (H, S) refills battery to maximum immediately
- Victory achieved by collecting all parks

### Fog of War
- Fog sessions hide the full grid. In GraphQL, never select `grid` without the session's grid password - the error nulls the whole `gameState` response.
- GraphQL `nearbyGrid` is a (2r+1)x(2r+1) window around the car (r = `fogRadius`, 1 without fog). Select `x y` on its cells and key your map by those coordinates; off-map cells read `building`.
- MCP `game_state`/`move` return a fixed 3x3 `local_view_3x3` regardless of `fog_radius` (row 0 = y-1, col 0 = x-1, `T` = car, off-map = `B`); MCP never returns the full grid for fog sessions.

### Critical Character Recognition

**⚠️ VISUAL SIMILARITY WARNING**: The characters R, B, and W can look similar in monospace fonts.

**Always verify character identity before planning routes:**
1. Parse grids character-by-character, not by visual pattern
2. When a path appears blocked, re-examine each position individually
3. Single R characters often appear between B or W clusters
4. Confirm uncertain positions from structured cell data: GraphQL `grid`/`nearbyGrid` cell `type`, or MCP `local_view_3x3` when adjacent

**Common misreads:**
- "BBBBR" read as "BBBBB" (missing the road at position 4)
- "RWWWW" read as "WWWWW" (missing the road at position 0)
- "BBRBB" read as "BBBBB" (missing the road in the middle)

## Strategic Framework

### Phase 1: World Analysis

Before making any moves:

1. **Map all objectives**: Count total parks and note their coordinates
2. **Identify charging infrastructure**: Locate all H and S positions
3. **Find safe corridors**: Identify obstacle-free rows/columns for efficient travel
4. **Analyze water crossings**: Find passages between grid sections divided by water

### Phase 2: Route Planning

Design a systematic collection strategy:

1. **Section the grid**: Divide large areas into manageable subsections
2. **Calculate distances**: Measure battery cost to each objective and nearest charger
3. **Plan charging stops**: Ensure routes pass through chargers before battery depletion
4. **Identify alternative routes**: Have backup paths for blocked corridors

### Phase 3: Execution

Implement the planned strategy:

1. **Use bulk moves for known safe paths**: Efficient for pre-validated corridors
2. **Use single moves for exploration**: Precise navigation in uncertain areas
3. **Monitor battery continuously**: Maintain 3+ battery buffer when far from chargers
4. **Verify position after moves**: Confirm location matches expectations

### Phase 4: Adaptive Refinement

When routes fail:

1. **Analyze the obstacle pattern**: Understand why the route was blocked
2. **Try perpendicular approaches**: If north is blocked, try east/west/south
3. **Update mental map**: Document newly discovered obstacles
4. **Apply lessons to remaining objectives**: Use pattern recognition for similar areas

## Proven Navigation Techniques

### Corridor Navigation
- **Golden Corridors**: Obstacle-free rows/columns allow rapid long-distance travel
- **Multi-Corridor Routes**: Chain safe passages to bypass complex obstacle clusters
- **Perpendicular Approaches**: When direct routes fail, try different cardinal directions

### Battery Management Principles
- **Safety Buffer**: Maintain 3+ battery when far from charging infrastructure
- **Strategic Base Camps**: Use chargers as staging points between grid sections
- **Route-Integrated Charging**: Plan paths that naturally pass through chargers
- **Distance Awareness**: Always know distance to nearest charging point

### Section-Based Completion
- **Progressive Mapping**: Build comprehensive mental maps as you explore
- **Complete Before Moving**: Fully clear one section before advancing
- **Pattern Replication**: Apply successful techniques from one area to similar areas

## Common Pitfalls to Avoid

1. **Misreading the grid**: Always verify cell types character-by-character
2. **Battery depletion**: Don't venture far without a charging plan
3. **Backtracking inefficiency**: Collect objectives systematically by section
4. **Reactive playing**: Plan routes before execution rather than moving randomly
5. **Ignoring alternatives**: When blocked, explore different approach angles

## API Usage Best Practices

### Game State Tools
- `game_state(session_id, grid?)`: Check position, battery, score before planning; `grid: true` adds the full grid (non-fog only); `local_view_3x3` shows the cells around you
- `list_maps()`, `get_map(...)`: Map list and details including cell configs

### Movement Tools
- `move(session_id, direction, intent)`: Single careful move with reasoning
- `bulk_move(session_id, moves, intent)`: Execute planned sequences with reasoning (max 50 moves; extras dropped with `truncated: true`)
- **Intent parameter**: Brief explanation of your reasoning (serves as rubber duck debugging to clarify your strategy)
- Both support optional `reset: true` parameter to restart before moving
- `bulk_move` results include `possible_moves`: directions you can legally move from the end position (walls, map edge, one-way roads, battery considered); empty after game over

### Session Management
- `create_session(map_id?, fog_enabled?, fog_radius?, grid_password?)`: Start new game on a chosen map
- `get_session(session_id)`: Session details
- `reset_game(session_id)`: Return to initial state, preserving session

## Iterative Mastery Approach

**Systematic improvement over speed**:
1. Analyze the complete grid before moving
2. Document obstacles and safe passages mentally
3. Execute planned routes methodically
4. Learn from failures and refine techniques
5. Build pattern recognition for obstacle formations

**Key Success Principles**:
- 🗺️ Systematic mapping beats random exploration
- ⚡ Proactive charging beats reactive emergency stops
- 🧩 Section-based completion beats scattered collection
- 🎯 Strategic planning beats rapid execution

## Strategy Documentation

For complex configurations, maintain mental notes of:
- Successful navigation routes by section
- Obstacle patterns and bypass techniques
- Optimal charging station sequences
- Alternative approach angles for difficult objectives

See [ADVANCED_STRATEGIES.md](references/ADVANCED_STRATEGIES.md) for detailed techniques on complex obstacle navigation and optimization patterns.
