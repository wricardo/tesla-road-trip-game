package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
)

// Client is a thin MCP client that proxies to the REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
	mcpServer  *server.MCPServer
}

// NewClient creates a new MCP client that calls the REST API
func NewClient(baseURL string) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	c.initMCPServer()
	return c
}

// initMCPServer initializes the MCP server with all tools
func (c *Client) initMCPServer() {
	c.mcpServer = server.NewMCPServer(
		"Tesla Road Trip Game",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(`Tesla Road Trip Game - MCP Interface

This is a thin client that proxies all requests to the REST API server.

GAME OBJECTIVE:
Visit all parks (P) to win. Your Tesla (T) starts with limited battery that depletes with each move.

AVAILABLE TOOLS:
- game_state: Get current game state
- move: Single move (up/down/left/right) - requires intent explanation
- bulk_move: Multiple moves at once - requires intent explanation
- reset_game: Reset to initial state
- move_history: View past moves
- create_session: Create new game session
- get_session: Get session details
- list_sessions: List all active sessions
- list_maps: List available maps
- game_instructions: Get comprehensive game instructions and rules
- describe_cell: Get detailed info about a specific grid cell (helps verify R vs B vs W)

NOTE: The 'intent' parameter on move/bulk_move tools serves as rubber duck debugging - explain your reasoning!`),
	)

	// Register all tools
	c.registerTools()
}

// registerTools registers all MCP tools
func (c *Client) registerTools() {
	// Session management
	c.mcpServer.AddTool(mcp.Tool{
		Name:        "create_session",
		Description: "Create a new game session with optional map selection",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"map_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the map to use (optional)",
				},
			},
		},
	}, c.handleCreateSession)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "list_sessions",
		Description: "List all active game sessions",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, c.handleListSessions)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "get_session",
		Description: "Get details of a specific session",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID to retrieve",
				},
			},
			Required: []string{"session_id"},
		},
	}, c.handleGetSession)

	// Game operations
	c.mcpServer.AddTool(mcp.Tool{
		Name:        "game_state",
		Description: "Get the current game state",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
			},
			Required: []string{"session_id"},
		},
	}, c.handleGameState)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "move",
		Description: "Move the player in a direction",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
				"direction": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"up", "down", "left", "right"},
					"description": "Direction to move",
				},
				"intent": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of the intent behind this move (serves as a rubber duck to help explain your reasoning)",
				},
				"reset": map[string]interface{}{
					"type":        "boolean",
					"description": "Reset before moving",
				},
			},
			Required: []string{"session_id", "direction"},
		},
	}, c.handleMove)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "bulk_move",
		Description: "Execute multiple moves in sequence",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
				"moves": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"up", "down", "left", "right"},
					},
					"description": "Array of moves",
				},
				"intent": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of the intent behind this sequence of moves (serves as a rubber duck to help explain your reasoning)",
				},
				"reset": map[string]interface{}{
					"type":        "boolean",
					"description": "Reset before moving",
				},
			},
			Required: []string{"session_id", "moves"},
		},
	}, c.handleBulkMove)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "reset_game",
		Description: "Reset the game to initial state",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
			},
			Required: []string{"session_id"},
		},
	}, c.handleReset)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "move_history",
		Description: "Get move history for a session",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "Page number",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Items per page",
				},
			},
			Required: []string{"session_id"},
		},
	}, c.handleMoveHistory)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "list_maps",
		Description: "List available game maps",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, c.handleListMaps)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "get_map",
		Description: "Get full details of a map including layout, battery settings, and messages",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Map name/ID"},
			},
			Required: []string{"name"},
		},
	}, c.handleGetMap)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "create_map",
		Description: "Create a new map. Layout rows are strings of R/H/P/S/W/B characters.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name":              map[string]interface{}{"type": "string", "description": "Unique map ID (lowercase, underscores)"},
				"description":       map[string]interface{}{"type": "string", "description": "Short description"},
				"grid_size":         map[string]interface{}{"type": "integer", "description": "Grid dimension (e.g. 10 for 10x10)"},
				"max_battery":       map[string]interface{}{"type": "integer", "description": "Maximum battery capacity"},
				"starting_battery":  map[string]interface{}{"type": "integer", "description": "Battery at game start"},
				"layout":            map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Grid rows, each a string of cell chars"},
				"wall_crash_ends_game": map[string]interface{}{"type": "boolean", "description": "Whether hitting a wall ends the game"},
			},
			Required: []string{"name", "grid_size", "max_battery", "starting_battery", "layout"},
		},
	}, c.handleMCPCreateMap)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "update_map",
		Description: "Partially update an existing map. Only provided fields are changed.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name":              map[string]interface{}{"type": "string", "description": "Map name/ID to update"},
				"description":       map[string]interface{}{"type": "string", "description": "New description"},
				"max_battery":       map[string]interface{}{"type": "integer", "description": "New max battery"},
				"starting_battery":  map[string]interface{}{"type": "integer", "description": "New starting battery"},
				"layout":            map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New grid layout rows"},
				"wall_crash_ends_game": map[string]interface{}{"type": "boolean", "description": "Wall collision behaviour"},
			},
			Required: []string{"name"},
		},
	}, c.handleMCPUpdateMap)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "delete_map",
		Description: "Delete a map by name. Cannot delete maps that are part of active sessions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Map name/ID to delete"},
			},
			Required: []string{"name"},
		},
	}, c.handleMCPDeleteMap)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "game_instructions",
		Description: "Get comprehensive game instructions and rules",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, c.handleGameInstructions)

	c.mcpServer.AddTool(mcp.Tool{
		Name:        "describe_cell",
		Description: "Get detailed information about a specific cell in the grid, including its exact character type. Useful for verifying whether a cell is passable (R, H, P, S) or impassable (W, B).",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID",
				},
				"x": map[string]interface{}{
					"type":        "integer",
					"description": "X coordinate (column) of the cell to describe (0-based)",
				},
				"y": map[string]interface{}{
					"type":        "integer",
					"description": "Y coordinate (row) of the cell to describe (0-based)",
				},
			},
			Required: []string{"session_id", "x", "y"},
		},
	}, c.handleDescribeCell)
}

// GetMCPServer returns the underlying MCP server for serving
func (c *Client) GetMCPServer() *server.MCPServer {
	return c.mcpServer
}

// Helper methods for API calls

func (c *Client) apiCall(method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		if msg, ok := errResp["error"]; ok {
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// Tool handlers

func (c *Client) handleCreateSession(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	mapName, _ := args["map_name"].(string)

	body := map[string]string{}
	if mapName != "" {
		body["map_name"] = mapName
	}

	var session service.SessionInfo
	err := c.apiCall("POST", "/api/sessions", body, &session)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := fmt.Sprintf("Created session: %s\nMap: %s\n", session.ID, session.MapName)
	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleListSessions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var response struct {
		Count    int                   `json:"count"`
		Sessions []service.SessionInfo `json:"sessions"`
	}

	err := c.apiCall("GET", "/api/sessions", nil, &response)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := fmt.Sprintf("Active Sessions (%d):\n\n", response.Count)
	for _, s := range response.Sessions {
		result += fmt.Sprintf("- %s (Map: %s, Created: %s)\n",
			s.ID, s.MapName, s.CreatedAt.Format("15:04:05"))
	}

	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleGetSession(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)

	var session service.SessionInfo
	err := c.apiCall("GET", fmt.Sprintf("/api/sessions/%s", sessionID), nil, &session)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := formatSessionInfo(&session)
	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleGameState(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)

	var state engine.GameState
	err := c.apiCall("GET", fmt.Sprintf("/api/sessions/%s/state", sessionID), nil, &state)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := formatGameState(&state)
	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)
	direction, _ := args["direction"].(string)
	intent, _ := args["intent"].(string)
	reset, _ := args["reset"].(bool)

	// Intent parameter serves as rubber duck debugging - we don't need to process it further
	_ = intent

	body := map[string]interface{}{
		"direction": direction,
		"reset":     reset,
	}

	var result service.MoveResult
	err := c.apiCall("POST", fmt.Sprintf("/api/sessions/%s/move", sessionID), body, &result)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	response := formatMoveResult(&result)
	return mcp.NewToolResultText(response), nil
}

func (c *Client) handleBulkMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)
	movesRaw, _ := args["moves"].([]interface{})
	intent, _ := args["intent"].(string)
	reset, _ := args["reset"].(bool)

	// Intent parameter serves as rubber duck debugging - we don't need to process it further
	_ = intent

	// Convert moves to string array
	moves := make([]string, 0, len(movesRaw))
	for _, m := range movesRaw {
		if move, ok := m.(string); ok {
			moves = append(moves, move)
		}
	}

	body := map[string]interface{}{
		"moves": moves,
		"reset": reset,
	}

	var result service.BulkMoveResult
	err := c.apiCall("POST", fmt.Sprintf("/api/sessions/%s/bulk-move", sessionID), body, &result)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	response := formatBulkMoveResult(sessionID, &result)
	return mcp.NewToolResultText(response), nil
}

func (c *Client) handleReset(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)

	var response struct {
		Message string            `json:"message"`
		State   *engine.GameState `json:"state"`
	}

	err := c.apiCall("POST", fmt.Sprintf("/api/sessions/%s/reset", sessionID), nil, &response)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := fmt.Sprintf("%s\n\n%s", response.Message, formatGameState(response.State))
	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleMoveHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)

	params := "?"
	if page, ok := args["page"].(float64); ok {
		params += fmt.Sprintf("page=%d&", int(page))
	}
	if limit, ok := args["limit"].(float64); ok {
		params += fmt.Sprintf("limit=%d&", int(limit))
	}

	var history service.HistoryResponse
	err := c.apiCall("GET", fmt.Sprintf("/api/sessions/%s/history%s", sessionID, params), nil, &history)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Also fetch current segment from live state
	var session service.SessionInfo
	if err := c.apiCall("GET", fmt.Sprintf("/api/sessions/%s", sessionID), nil, &session); err != nil {
		// If fetching session fails, still return the history
		result := formatHistory(&history)
		return mcp.NewToolResultText(result), nil
	}

	result := formatHistory(&history)
	result += "\n" + formatCurrentSegment(session.GameState)
	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleListMaps(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var maps []service.MapInfo
	err := c.apiCall("GET", "/api/maps", nil, &maps)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := "Available Maps:\n\n"
	for _, m := range maps {
		result += fmt.Sprintf("• %s\n  %s\n  Grid: %dx%d, Battery: %d\n\n",
			m.Name, m.Description, m.GridSize, m.GridSize, m.MaxBattery)
	}

	return mcp.NewToolResultText(result), nil
}

func (c *Client) handleGetMap(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := request.GetArguments()["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	var result interface{}
	if err := c.apiCall("GET", fmt.Sprintf("/api/maps/%s", name), nil, &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (c *Client) handleMCPCreateMap(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	var result interface{}
	if err := c.apiCall("POST", fmt.Sprintf("/api/maps/%s", name), args, &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q created", name)), nil
}

func (c *Client) handleMCPUpdateMap(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	var result interface{}
	if err := c.apiCall("PATCH", fmt.Sprintf("/api/maps/%s", name), args, &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q updated", name)), nil
}

func (c *Client) handleMCPDeleteMap(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := request.GetArguments()["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	var result interface{}
	if err := c.apiCall("DELETE", fmt.Sprintf("/api/maps/%s", name), nil, &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q deleted", name)), nil
}

func (c *Client) handleGameInstructions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instructions := `🎮 Tesla Road Trip Game - Complete Instructions

GAME OBJECTIVE:
Navigate your Tesla to visit all parks (P) while managing battery life and avoiding obstacles.

GAME MECHANICS:
• Movement: Each move consumes 1 battery unit
• Charging: Restore battery to full at home tiles (H) or superchargers (S)
• Victory: Collect all parks to win the game
• Game Over: Battery depleted with no reachable charging stations

GRID LEGEND:
• T - Tesla (your current position)
• R - Road (passable terrain) ⚠️ CRITICAL: Can look similar to B in some fonts!
• H - Home (passable, charging station, represents your home base/garage)
• P - Park (passable, collectible objective)
• S - Supercharger (passable, charging station)
• W - Water (impassable obstacle) ⚠️ Do NOT confuse with R
• B - Building (impassable obstacle) ⚠️ Do NOT confuse with R
• ✓ - Visited park (shows completed objectives)

🤖 AI AGENTS - CRITICAL SUCCESS STRATEGIES:

⚠️ CHARACTER RECOGNITION (MOST COMMON FAILURE POINT):
BEFORE any navigation planning, you MUST:

1. **Parse Character-by-Character**: Never scan visually - examine each position
   Example: "BBBBRWWWWWBBBBB" must be parsed as:
   Position 0-3: B B B B (buildings)
   Position 4: R (ROAD!) ← This is passable!
   Position 5-9: W W W W W (water)
   Position 10-14: B B B B B (buildings)

2. **Common Misreading Patterns**:
   - "BBBBR" often misread as "BBBBB"
   - "RWWWW" often misread as "WWWWW"
   - "BBRBB" - the middle R is frequently missed

3. **Verification Strategy**:
   - If a row appears "completely blocked", re-examine position by position
   - Look for single R characters between B/W clusters
   - Use test moves to verify character interpretation
   - Double-check any row that seems to have no passages

🗺️ SYSTEMATIC WORLD MAPPING:
- Create ASCII grid representations showing your understanding
- Mark all parks, chargers, and obstacle patterns
- Update maps iteratively as you explore
- Build comprehensive understanding before major route planning

🧩 CORRIDOR NAVIGATION TECHNIQUE:
- Identify horizontal and vertical corridors of passable cells (R, H, P, S)
- Use "golden corridors" (completely obstacle-free rows/columns)
- Plan multi-corridor routes to bypass obstacle clusters
- Apply perpendicular approaches when direct routes are blocked

⚡ PROACTIVE BATTERY MANAGEMENT:
- Calculate distances to ALL charging stations before starting routes
- Recharge when battery > 50% if near charger (don't wait until critical)
- Use charging stations as strategic "base camps" between sections
- Always maintain enough battery to reach nearest charger + safety margin
- Questions to ask: Where are nearest chargers? How much battery left? Any walls nearby?

🎯 SECTION-BASED PROBLEM SOLVING:
- Divide large grids into logical sections
- Complete one section fully before moving to next
- Use iterative refinement when approaches fail
- Document successful routes for pattern reuse

🔄 ITERATIVE DEVELOPMENT:
1. **Analysis**: Character-by-character grid parsing, locate objectives and charging
2. **Planning**: Design section-based routes using corridor navigation
3. **Execution**: Implement with proactive battery management
4. **Refinement**: Analyze failures, update understanding, iterate

🚨 CRITICAL PITFALLS TO AVOID:
- ❌ Attempting direct routes without systematic obstacle analysis
- ❌ Depleting battery without clear charging path
- ❌ Abandoning partially successful routes (refine them instead)
- ❌ Ignoring corridor navigation opportunities
- ❌ **MOST CRITICAL**: Assuming rows are "completely blocked" without character-by-character verification
- ❌ Confusing R (road) with B (building) or W (water) - they look similar in text
- ❌ Visual pattern scanning instead of systematic character parsing

🐛 DEBUGGING CHARACTER RECOGNITION:
When you think a row is "completely blocked":
1. Request exact grid display output
2. Parse each character position individually: grid[row][0], grid[row][1], etc.
3. Look specifically for R characters between obstacles
4. Test exploratory moves to verify interpretation
5. Common hidden patterns: BBRBB, WWRWW, BBRWB

🎮 API USAGE BEST PRACTICES:
- Use bulk_move for efficiency rather than individual moves
- Implement proper error handling for collisions
- Monitor game state continuously during execution
- Save/load for complex route testing and recovery

MOVEMENT COMMANDS:
- up, down, left, right - Single moves in cardinal directions
- Bulk moves - Execute multiple moves in sequence for efficiency
- Reset parameter available for fresh starts

CHARGING LOCATIONS:
- Home tiles (H): Your Tesla garage/base, provides full charge
- Superchargers (S): Public charging stations, provide full charge

VICTORY CONDITIONS:
- Visit ALL parks in the grid to achieve victory
- Parks show as ✓ when successfully visited
- Game displays "🎉 VICTORY!" when all parks collected

GAME OVER CONDITIONS:
- Battery reaches 0 with no accessible charging stations
- Game displays "💀 GAME OVER" when this occurs

CONFIGURATION OPTIONS:
- Easy maps: Smaller grids, more chargers, simple layouts
- Medium maps: Balanced challenge with strategic elements
- Hard maps: Complex mazes requiring careful planning

SESSION MANAGEMENT:
- Multiple game sessions can run simultaneously
- Each session has unique 4-character ID
- Sessions maintain independent state and configuration
- Use session-specific tools for multi-game management

Remember: Success requires meticulous character recognition, systematic mapping, and proactive battery management. The most common AI failure is misreading grid characters - always verify R vs B vs W carefully!

Good luck navigating your Tesla Road Trip! 🚗⚡🌳`

	return mcp.NewToolResultText(instructions), nil
}

func (c *Client) handleDescribeCell(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	sessionID, _ := args["session_id"].(string)
	x := int(args["x"].(float64))
	y := int(args["y"].(float64))

	// Get the current game state to access the grid
	var state engine.GameState
	err := c.apiCall("GET", fmt.Sprintf("/api/sessions/%s/state", sessionID), nil, &state)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check bounds
	gridSize := len(state.Grid)
	if x < 0 || x >= gridSize || y < 0 || y >= gridSize {
		return mcp.NewToolResultError(fmt.Sprintf("Coordinates (%d, %d) are out of bounds. Grid size is %dx%d (0-%d for both x and y)",
			x, y, gridSize, gridSize, gridSize-1)), nil
	}

	// Get cell information
	cell := state.Grid[y][x]

	// Determine cell character and description
	var cellChar string
	var cellType string
	var passable bool
	var description string

	// Check if player is at this position
	if x == state.PlayerPos.X && y == state.PlayerPos.Y {
		cellChar = "T"
		description = "Player's current position (Tesla)"
	}

	switch cell.Type {
	case engine.Road:
		if cellChar == "" {
			cellChar = "R"
		}
		cellType = "Road"
		passable = true
		if description == "" {
			description = "Empty road - safe to travel"
		}
	case engine.Home:
		if cellChar == "" {
			cellChar = "H"
		}
		cellType = "Home"
		passable = true
		if description == "" {
			description = "Home/Garage - provides full battery charge"
		}
	case engine.Park:
		if cell.Visited {
			if cellChar == "" {
				cellChar = "✓"
			}
			cellType = "Park (Visited)"
			if description == "" {
				description = "Park already visited - objective completed here"
			}
		} else {
			if cellChar == "" {
				cellChar = "P"
			}
			cellType = "Park"
			if description == "" {
				description = "Park to visit - objective location"
			}
		}
		passable = true
	case engine.Supercharger:
		if cellChar == "" {
			cellChar = "S"
		}
		cellType = "Supercharger"
		passable = true
		if description == "" {
			description = "Supercharger station - provides full battery charge"
		}
	case engine.Water:
		if cellChar == "" {
			cellChar = "W"
		}
		cellType = "Water"
		passable = false
		if description == "" {
			description = "Water obstacle - IMPASSABLE"
		}
	case engine.Building:
		if cellChar == "" {
			cellChar = "B"
		}
		cellType = "Building"
		passable = false
		if description == "" {
			description = "Building obstacle - IMPASSABLE"
		}
	default:
		cellChar = "?"
		cellType = "Unknown"
		passable = false
		description = "Unknown cell type"
	}

	// Build result
	result := fmt.Sprintf(`Cell at position (%d, %d):
━━━━━━━━━━━━━━━━━━━━━━━━
Character: %s
Type: %s
Passable: %v
Description: %s

IMPORTANT: The character '%s' is what appears in the grid display.
%s`,
		x, y,
		cellChar,
		cellType,
		passable,
		description,
		cellChar,
		getCharacterReminder(cellChar))

	return mcp.NewToolResultText(result), nil
}

func getCharacterReminder(char string) string {
	switch char {
	case "R":
		return "⚠️ REMINDER: 'R' (road) is often confused with 'B' (building). This is a ROAD and is PASSABLE!"
	case "B":
		return "⚠️ REMINDER: 'B' (building) is often confused with 'R' (road). This is a BUILDING and is IMPASSABLE!"
	case "W":
		return "⚠️ REMINDER: 'W' (water) is an obstacle. This is IMPASSABLE!"
	case "H":
		return "✅ This is a charging location (Home) - safe to move here and will restore battery!"
	case "S":
		return "✅ This is a charging location (Supercharger) - safe to move here and will restore battery!"
	case "P":
		return "🎯 This is an objective (Park) - you need to visit all parks to win!"
	case "✓":
		return "✅ This park has already been visited."
	case "T":
		return "🚗 This is where you (the Tesla) currently are."
	default:
		return ""
	}
}

// Formatting helpers

func formatSessionInfo(session *service.SessionInfo) string {
	return fmt.Sprintf("Session: %s\nMap: %s\nCreated: %s\n\n%s",
		session.ID, session.MapName,
		session.CreatedAt.Format("2006-01-02 15:04:05"),
		formatGameState(session.GameState))
}

func formatGameState(state *engine.GameState) string {
	if state == nil {
		return "No game state available"
	}

	var result strings.Builder
	gridSize := len(state.Grid)

	// Header (include cumulative total moves)
	result.WriteString(fmt.Sprintf("Position: (%d,%d) | Battery: %d/%d | Score: %d | Moves: %d\n\n",
		state.PlayerPos.X, state.PlayerPos.Y,
		state.Battery, state.MaxBattery, state.Score, state.TotalMoves))

	// Decision aids (if available)
	if state.BatteryRisk != "" {
		result.WriteString(fmt.Sprintf("Battery risk: %s\n", state.BatteryRisk))
	}
	// Prefer server-provided local_view_3x3; otherwise derive
	if len(state.LocalView3x3) == 3 {
		result.WriteString("Local 3x3:\n")
		result.WriteString(state.LocalView3x3[0] + "\n")
		result.WriteString(state.LocalView3x3[1] + "\n")
		result.WriteString(state.LocalView3x3[2] + "\n\n")
	} else if v := formatLocal3x3(state); v != "" {
		result.WriteString("Local 3x3:\n")
		result.WriteString(v + "\n")
	}

	// Grid
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if x == state.PlayerPos.X && y == state.PlayerPos.Y {
				result.WriteString("T")
			} else {
				cell := state.Grid[y][x]
				switch cell.Type {
				case engine.Road:
					result.WriteString("R")
				case engine.Home:
					result.WriteString("H")
				case engine.Park:
					if cell.Visited {
						result.WriteString("✓")
					} else {
						result.WriteString("P")
					}
				case engine.Supercharger:
					result.WriteString("S")
				case engine.Water:
					result.WriteString("W")
				case engine.Building:
					result.WriteString("B")
				default:
					result.WriteString(".")
				}
			}
		}
		result.WriteString("\n")
	}

	// Status
	if state.GameOver {
		if state.Victory {
			result.WriteString("\n🎉 VICTORY!")
		} else {
			result.WriteString("\n💀 GAME OVER")
		}
	}

	if state.Message != "" {
		result.WriteString(fmt.Sprintf("\nMessage: %s", state.Message))
	}

	return result.String()
}

func formatMoveResult(result *service.MoveResult) string {
	response := ""
	if result.Success {
		response = "✓ Move successful\n"
	} else {
		response = "✗ Move failed\n"
	}

	// Compact step summary (if available)
	if result.Step != nil {
		s := result.Step
		status := "✗"
		if s.Success {
			status = "✓"
		}
		response += fmt.Sprintf("Step: %s (%d,%d)→(%d,%d) tile=%s batt=%d %s\n",
			s.Dir, s.From.X, s.From.Y, s.To.X, s.To.Y, s.TileChar, s.BatteryAfter, status)
	}

	// Failure diagnostic (if available)
	if result.AttemptedTo != nil {
		a := result.AttemptedTo
		passStr := "impassable"
		if a.Passable {
			passStr = "passable"
		}
		response += fmt.Sprintf("Blocked: attempted (%d,%d) tile=%s %s (%s)\n", a.X, a.Y, a.TileChar, a.TileType, passStr)
	}

	if len(result.Events) > 0 {
		response += "Events:\n"
		for _, event := range result.Events {
			response += fmt.Sprintf("- %s: %s\n", event.Type, event.Message)
		}
	}

	response += "\n" + formatGameState(result.GameState)
	return response
}

func formatBulkMoveResult(sessionID string, result *service.BulkMoveResult) string {
	var b strings.Builder

	// Session header
	gridSize := 0
	mapName := ""
	if result.GameState != nil {
		gridSize = len(result.GameState.Grid)
		mapName = result.GameState.MapName
	}
	b.WriteString(fmt.Sprintf("Session: %s • Map: %s • Grid: %dx%d\n",
		sessionID, mapName, gridSize, gridSize))

	// Bulk summary
	requested := result.RequestedMoves
	if requested == 0 {
		requested = result.TotalMoves // backward-compat
	}
	b.WriteString(fmt.Sprintf("Executed %d/%d moves\n", result.MovesExecuted, requested))
	if result.StoppedReason != "" {
		b.WriteString(fmt.Sprintf("Stopped: %s\n", result.StoppedReason))
	}

	// Events (keep as-is, concise)
	if len(result.Events) > 0 {
		b.WriteString("\nEvents:\n")
		for _, event := range result.Events {
			b.WriteString(fmt.Sprintf("- %s: %s\n", event.Type, event.Message))
		}
	}

	// Recent steps: last N entries from current segment where N = moves_executed
	if result.GameState != nil && result.MovesExecuted > 0 {
		steps := getRecentSteps(result.GameState, result.MovesExecuted)
		if len(steps) > 0 {
			b.WriteString("\nRecent steps (this call):\n")
			for i, s := range steps {
				b.WriteString(formatStepLine(i+1, s, result.GameState))
			}
		}
	}

	// Stopped diagnostic: infer last attempted cell on failure
	if result.StoppedReason != "" && result.GameState != nil {
		if line := formatStoppedDiagnostic(result.MovesExecuted, result.GameState); line != "" {
			b.WriteString("\n")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Possible moves and local 3x3 view from final state
	if result.GameState != nil {
		pm := computePossibleMoves(result.GameState)
		if len(pm) > 0 {
			b.WriteString("\nPossible moves: ")
			b.WriteString(strings.Join(pm, ","))
			b.WriteString("\n")
		}
		if v := formatLocal3x3(result.GameState); v != "" {
			b.WriteString("Local 3x3:\n")
			b.WriteString(v)
			// Ensure trailing newline
			if !strings.HasSuffix(v, "\n") {
				b.WriteString("\n")
			}
		}
	}

	// Full state at the end (kept for compatibility)
	b.WriteString("\n")
	b.WriteString(formatGameState(result.GameState))
	return b.String()
}

// getRecentSteps returns the last N entries from CurrentMoves
func getRecentSteps(state *engine.GameState, n int) []engine.MoveHistoryEntry {
	total := len(state.CurrentMoves)
	if total == 0 || n <= 0 {
		return nil
	}
	if n > total {
		n = total
	}
	return state.CurrentMoves[total-n:]
}

// formatStepLine renders a single compact step line
func formatStepLine(idx int, entry engine.MoveHistoryEntry, state *engine.GameState) string {
	from := entry.FromPosition
	to := entry.ToPosition
	// Determine tile at destination (for successful steps this is the new cell)
	tileChar := inferTileChar(state, to.X, to.Y)
	status := "✗"
	if entry.Success {
		status = "✓"
	}
	return fmt.Sprintf("%d. %s (%d,%d)→(%d,%d) tile=%s batt=%d %s\n",
		idx, entry.Action, from.X, from.Y, to.X, to.Y, tileChar, entry.Battery, status)
}

// formatStoppedDiagnostic describes why the last attempt likely failed
func formatStoppedDiagnostic(movesExecuted int, state *engine.GameState) string {
	// Last history entry corresponds to the last attempt (may be failed)
	if len(state.MoveHistory) == 0 {
		return ""
	}
	last := state.MoveHistory[len(state.MoveHistory)-1]
	if last.Success {
		// If the last was successful but we stopped, it was likely due to game over after move
		if state.GameOver {
			if state.Victory {
				return "Game over: victory achieved"
			}
			if state.Message != "" {
				return fmt.Sprintf("Game over: %s", state.Message)
			}
			return "Game over"
		}
		return ""
	}

	// Compute attempted target based on direction
	tx, ty := last.FromPosition.X, last.FromPosition.Y
	switch strings.ToLower(last.Action) {
	case "up":
		ty--
	case "down":
		ty++
	case "left":
		tx--
	case "right":
		tx++
	}

	gridSize := len(state.Grid)
	moveNum := movesExecuted + 1 // 1-based index of the failed attempt within this call

	// Boundary check
	if tx < 0 || ty < 0 || ty >= gridSize || (gridSize > 0 && tx >= len(state.Grid[0])) {
		return fmt.Sprintf("Blocked on move %d: attempted (%d,%d) tile=boundary (impassable)", moveNum, tx, ty)
	}

	cell := state.Grid[ty][tx]
	char := mapCellToChar(cell)
	passable := cell.Type != engine.Water && cell.Type != engine.Building

	reason := "blocked"
	if !passable {
		reason = "blocked by obstacle"
	} else if state.Battery == 0 {
		reason = "battery exhausted"
	} else if state.GameOver {
		reason = "game over"
	}

	return fmt.Sprintf("Blocked on move %d: attempted (%d,%d) tile=%s %s", moveNum, tx, ty, char, reason)
}

// computePossibleMoves returns valid directions from the current state
func computePossibleMoves(state *engine.GameState) []string {
	if state == nil || state.GameOver || state.Battery <= 0 {
		return []string{}
	}
	dirs := []string{"up", "down", "left", "right"}
	var res []string
	px, py := state.PlayerPos.X, state.PlayerPos.Y
	gridH := len(state.Grid)
	gridW := 0
	if gridH > 0 {
		gridW = len(state.Grid[0])
	}
	can := func(x, y int) bool {
		if x < 0 || y < 0 || y >= gridH || x >= gridW {
			return false
		}
		t := state.Grid[y][x].Type
		return t != engine.Water && t != engine.Building
	}
	for _, d := range dirs {
		x, y := px, py
		switch d {
		case "up":
			y--
		case "down":
			y++
		case "left":
			x--
		case "right":
			x++
		}
		if can(x, y) {
			res = append(res, d)
		}
	}
	return res
}

// formatLocal3x3 renders a 3x3 character window centered on the player
func formatLocal3x3(state *engine.GameState) string {
	if state == nil {
		return ""
	}
	px, py := state.PlayerPos.X, state.PlayerPos.Y
	var lines [3]string
	for dy := -1; dy <= 1; dy++ {
		var row strings.Builder
		for dx := -1; dx <= 1; dx++ {
			x, y := px+dx, py+dy
			if dx == 0 && dy == 0 {
				row.WriteString("T")
				continue
			}
			row.WriteString(inferTileChar(state, x, y))
		}
		lines[dy+1] = row.String()
	}
	return lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n"
}

// inferTileChar returns a single-character representation for a cell at (x,y), handling OOB
func inferTileChar(state *engine.GameState, x, y int) string {
	gridH := len(state.Grid)
	if x < 0 || y < 0 || y >= gridH || (gridH > 0 && x >= len(state.Grid[0])) {
		return "B" // out-of-bounds treated as building/wall
	}
	cell := state.Grid[y][x]
	return mapCellToChar(cell)
}

func mapCellToChar(cell engine.Cell) string {
	switch cell.Type {
	case engine.Road:
		return "R"
	case engine.Home:
		return "H"
	case engine.Park:
		if cell.Visited {
			return "✓"
		}
		return "P"
	case engine.Supercharger:
		return "S"
	case engine.Water:
		return "W"
	case engine.Building:
		return "B"
	default:
		return "."
	}
}

func formatHistory(history *service.HistoryResponse) string {
	result := fmt.Sprintf("Move History (Page %d/%d) — Total (cumulative): %d\n\n",
		history.Page, history.TotalPages, history.TotalMoves)

	for i, move := range history.Moves {
		num := (history.Page-1)*history.PageSize + i + 1
		status := "✓"
		if !move.Success {
			status = "✗"
		}
		result += fmt.Sprintf("%d. %s %s [Battery: %d]\n",
			num, move.Action, status, move.Battery)
	}

	return result
}

func formatCurrentSegment(state *engine.GameState) string {
	if state == nil {
		return "Current Segment: unavailable"
	}
	moves := state.CurrentMoves
	total := state.CurrentMovesCount
	header := fmt.Sprintf("Current Move Segment — Moves: %d\n\n", total)
	if len(moves) == 0 {
		return header + "(no moves in current segment)"
	}
	var b strings.Builder
	b.WriteString(header)
	for i, move := range moves {
		status := "✓"
		if !move.Success {
			status = "✗"
		}
		// i is zero-based within the segment
		b.WriteString(fmt.Sprintf("%d. %s %s [Battery: %d]\n", i+1, move.Action, status, move.Battery))
	}
	return b.String()
}
