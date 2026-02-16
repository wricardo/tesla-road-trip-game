# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Tesla Road Trip Game - A grid-based multi-session game server written in Go where players control a Tesla to collect parks while managing battery. Features per-session configuration system, isolated game sessions, WebSocket support, RESTful API, and MCP server integration for AI assistance.

## Development Commands

WHEN running the game server, try to use default port 8080

### Build and Run
```bash
# Build the game
go build -o statefullgame

# Run the game server
./statefullgame                                # Default server mode
./statefullgame -port 9090                     # Custom port
```

### Testing
```bash
# Run API tests
./test-api.sh

# Test specific endpoints (default session)
curl http://localhost:8080/api                                          # Get game state
curl -X POST http://localhost:8080/api -d '{"action":"right"}'         # Move player
curl -X POST http://localhost:8080/api -d '{"action":"save"}'          # Save game
curl http://localhost:8080/api/configs                                  # List configs
curl http://localhost:8080/api/saves                                    # List saves

# Test session management
curl -X POST http://localhost:8080/api/sessions                         # Create session (default config)
curl -X POST http://localhost:8080/api/sessions -d '{"config_name":"easy"}'             # Create session with easy config
curl -X POST http://localhost:8080/api/sessions -d '{"config_name":"medium_maze"}'      # Create session with medium maze
curl http://localhost:8080/api/sessions/a3x7                           # Get specific session state

# Test session-specific gameplay
curl http://localhost:8080/api?sessionId=a3x7                          # Get session state
curl -X POST http://localhost:8080/api?sessionId=a3x7 -d '{"action":"right"}'          # Move in specific session
curl -X POST http://localhost:8080/api?sessionId=a3x7 -d '{"actions":["up","left"]}'   # Bulk moves in session
curl http://localhost:8080/api/history?sessionId=a3x7                  # Session move history

# Test reset functionality
curl -X POST http://localhost:8080/api -d '{"action":"right","reset":true}'              # Move with reset
curl -X POST http://localhost:8080/api -d '{"actions":["up","left"],"reset":true}'      # Bulk moves with reset

# Test MCP server integration
./test-mcp-integration.sh
```

### MCP Server
```bash
# Run MCP stdio server (self-contained, no separate server needed)
./statefullgame stdio-mcp                                              # Stdio mode with internal HTTP server

# OR run HTTP server with MCP endpoint always available
./statefullgame                                                        # Default server mode
# Then use MCP via HTTP endpoint: http://localhost:8080/mcp

# MCP tools support session management
# create_session(config_name?) - Create session with optional config
# get_session(session_id) - Get specific session state
# describe_cell(session_id, x, y) - Get detailed info about a specific grid cell
# All game tools accept optional session_id parameter
```

### Dependencies
```bash
go mod download  # Install dependencies (gorilla/websocket)
```

## Architecture

### Core Components

**main.go** - Single-file architecture containing:
- **GameSession**: Multi-session container with unique ID, state, and configuration
- **GameState**: Runtime game state with grid, player position, battery, score
- **GameConfig**: JSON-loaded configuration defining grid layout, battery limits, messages
- **WebSocket Hub**: Broadcasts state changes to connected clients in real-time
- **HTTP Handlers**: REST API endpoints with session support and WebSocket upgrade handler

### Data Flow
1. **Session Creation**: Client creates session → select config → server initializes GameSession with chosen config
2. **Multi-Session Game Loop**: API request with sessionId → action processing on specific session → state mutation → WebSocket broadcast to session clients  
3. **Session Isolation**: Each session maintains independent state, configuration, and history
4. **Persistence**: Save action → serialize GameState to `saves/` directory as timestamped JSON
5. **State Recovery**: Load action → deserialize saved JSON → replace current GameState

### Configuration System

Game configurations (`configs/*.json`) define:
- Grid layout using character mapping:
  - **R = road (passable)** - CRITICAL: Look for 'R' carefully, it can be adjacent to B/W
  - **H = home (passable, charges battery to full)** - represents home base/garage
  - **P = park (passable, objective to collect)**
  - **S = supercharger (passable, charges battery to full)**
  - **W = water (impassable obstacle)** - Do NOT confuse with R
  - **B = building (impassable obstacle)** - Do NOT confuse with R

**⚠️ CRITICAL CHARACTER RECOGNITION**:
- The character 'R' (road) is visually similar to 'B' and may appear between obstacles
- When analyzing grids, examine each character individually: R≠B≠W
- Roads (R) often appear as single-character gaps between obstacles
- Double-check any row that appears "completely blocked" - there may be an R hidden between B/W characters
- Battery parameters (max_battery, starting_battery)
- Custom messages for game events
- Grid size and difficulty settings

Sessions can use any available configuration independently. The server loads the default configuration at startup via `-config` flag, but each session can override this. Multiple sessions can run different configurations simultaneously without server restart.

### Save System

- Saves stored as `saves/save_<timestamp>.json`
- Complete state serialization including grid, visited parks, position, battery
- Survives server restarts
- Can load saves from different game configurations

### API Design

**Multi-Session Support**: All endpoints support optional `sessionId` query parameter for session-specific operations.

**Session Management**:
- `POST /api/sessions` - Create new session with optional config selection
- `GET /api/sessions/{sessionId}` - Get specific session state

**Single endpoint multiplexing** (`/api`):
- GET: Returns current GameState (supports `?sessionId=`)
- POST with action field: Executes game actions (supports `?sessionId=`)
  - `{"action": "up/down/left/right"}` - Single move
  - `{"actions": ["up", "right", "down"]}` - Bulk moves
  - `{"action": "up", "reset": true}` - Single move with reset
  - `{"actions": ["up", "right"], "reset": true}` - Bulk moves with reset
  - `{"action": "reset"}` - Reset game only

**Reset Parameter**: Both single and bulk move requests support an optional `reset: true/false` parameter that resets the game to initial state before executing the move(s). This saves an API call by combining reset + move operations.

**Session-Aware Auxiliary endpoints**:
- `/api/configs`: Lists available game configurations
- `/api/saves`: Lists saved game sessions
- `/api/history?sessionId=`: Paginated move history for specific session
- `/ws?sessionId=`: WebSocket for real-time state updates (session-specific)

## Key Implementation Details

- **Multi-session architecture**: Thread-safe session management with 4-character IDs
- **Per-session configuration**: Each session can run different maze/config independently  
- **Session isolation**: Complete state separation between concurrent sessions
- **Embedded templates**: HTML template with session selection UI
- **Atomic saves**: Game state serialized with `json.MarshalIndent` for readability
- **Grid validation**: `canMoveTo()` checks bounds and cell type before movement (only water and buildings are impassable)
- **Victory condition**: Tracked by comparing visited parks count to total parks in grid
- **Battery management**: Movement costs 1 battery, home tiles and superchargers restore to max
- **Home tiles**: Represent Tesla home garage/base - passable and provide full battery charge

## Testing Workflow

When testing game functionality:
1. Start server (loads default config)
2. Create multiple sessions with different configs via API/UI
3. Test session isolation - verify moves in one session don't affect others
4. Use `curl` or `test-api.sh` to verify endpoints with sessionId parameters
5. Test save/load across server restarts
6. Verify WebSocket updates with browser dev tools (session-specific)
7. Test MCP server with session management tools

## AI Strategy Guidelines

When working with AI agents to play or solve the Tesla Road Trip game, the following systematic approaches have proven highly effective:

### Grid Character Recognition Strategy

**🔤 ESSENTIAL: Accurate Character Recognition**

Before any navigation planning, agents MUST:

1. **Character-by-Character Analysis**
   - Parse each row character by character, not as patterns
   - Example: "BBBBRWWWWWBBBBB" should be parsed as:
     ```
     Position 0: B (building)
     Position 1: B (building)
     Position 2: B (building)
     Position 3: B (building)
     Position 4: R (road) ← CRITICAL: This is an R, not B or W!
     Position 5: W (water)
     ... and so on
     ```

2. **Visual Similarity Awareness**
   - R (road) can look similar to B (building) in monospace fonts
   - Always double-check rows that appear "completely blocked"
   - Single R characters often appear between clusters of B or W

3. **Systematic Verification**
   - When a row seems impassable, re-examine it position by position
   - Test movement to verify character interpretation
   - If movement fails where you expect an R, re-parse that specific position

4. **Grid Mapping Best Practice**
   ```
   CORRECT approach:
   - Parse: B B B B R W W W W W B B B B B
   - Index: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14
   - Note: Road at position 4!

   INCORRECT approach:
   - Quick scan: "All blocked, no roads"
   - Missing the critical R character
   ```

### Proven Navigation Strategies

**🗺️ Systematic World Mapping**
- Create ASCII representations of the grid to track understanding
- Document obstacle patterns and safe navigation corridors
- Build comprehensive maps of all parks, chargers, and obstacles
- Update maps iteratively as new areas are explored

**🧩 Corridor Navigation Technique**
- Identify horizontal and vertical corridors of passable cells
- Use "golden corridors" (obstacle-free rows/columns) for efficient navigation
- Plan multi-corridor routes to bypass complex obstacle clusters
- Apply perpendicular approach strategies when direct routes are blocked

**⚡ Proactive Battery Management**
- Calculate distances to charging stations before starting routes
- Recharge proactively rather than waiting until battery is critically low
- Use charging stations as strategic "base camps" between game sections
- Plan routes that efficiently pass through available charging infrastructure

**🎯 Section-Based Problem Solving**
- Divide large grids into manageable sections
- Complete one section fully before moving to the next
- Use iterative refinement when initial approaches encounter obstacles
- Document successful routes for reuse in similar patterns

### Iterative Development Methodology

**📝 Documentation-Driven Approach**
- Maintain ASCII maps showing evolving understanding of the game world
- Create versioned scripts/approaches to track learning progression
- Document both successful techniques and failed attempts with reasons
- Build systematic knowledge that applies across different grid configurations

**🔄 Systematic Iteration Process**
1. **Analysis Phase**: Map the world, identify objectives and charging infrastructure
2. **Planning Phase**: Design section-based routes using corridor navigation
3. **Execution Phase**: Implement planned routes with proactive battery management
4. **Refinement Phase**: Analyze failures, update world understanding, iterate

**🧠 Adaptive Problem Solving**
- When direct routes fail, systematically try alternative approach angles
- Use building pattern recognition to predict and avoid similar obstacles
- Build upon partial successes rather than restarting from scratch
- Apply lessons learned from one grid section to similar patterns elsewhere

### Best Practices for AI Implementation

**🎮 API Usage Patterns**
- Use batch move execution for efficiency rather than individual API calls
- Implement proper error handling for obstacle collisions
- Monitor game state continuously during execution
- Leverage save/load functionality for complex route testing

**🚨 Common Pitfalls to Avoid**
- Don't attempt direct routes without systematic obstacle analysis
- Avoid depleting battery without clear path to charging stations
- Don't abandon partially successful routes - refine them instead
- Don't ignore corridor navigation opportunities in complex areas
- **CRITICAL: Don't assume a row is fully blocked without character-by-character verification**
- **Don't confuse R (road) with B (building) or W (water) - they look similar**

**🐛 Debugging Character Recognition Issues**
When agents report "completely blocked" rows:
1. Request the exact grid display output
2. Parse each character position individually
3. Look specifically for 'R' characters between obstacles
4. Common misread patterns:
   - "BBBBR" may be read as "BBBBB"
   - "RWWWW" may be read as "WWWWW"
   - "BBRBB" - the R in the middle is often missed
5. Use exploratory moves to verify character interpretation

**🏆 Victory Optimization**
- Focus on systematic completion rather than speed optimization
- Maintain resource safety buffers for unexpected route changes
- Use section-based approaches to ensure progress isn't lost
- Apply proven techniques consistently across different game configurations

These strategies have achieved consistent victory across multiple game configurations through systematic application of corridor navigation, proactive battery management, and iterative problem-solving approaches.
- when we are playing the game, questions to think about: where
are the nearst chargers/home? how much battery do I have left?
any walls nearby?


