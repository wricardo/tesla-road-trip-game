package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/toon-format/toon-go"
	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

// mcpHTTPRequestKey is the context key for the incoming *http.Request.
type mcpHTTPRequestKey struct{}

const maxMCPRequestBodyBytes int64 = 1 << 20 // 1 MiB

// checkAdminKey requires X-Admin-Key for admin mutations. ADMIN_API_KEY must be
// configured; local development can opt out explicitly with
// ALLOW_UNAUTHENTICATED_ADMIN=true.
func checkAdminKey(ctx context.Context) error {
	required := os.Getenv("ADMIN_API_KEY")
	if required == "" {
		if os.Getenv("ALLOW_UNAUTHENTICATED_ADMIN") == "true" {
			return nil
		}
		return errors.New("admin operation disabled: set ADMIN_API_KEY and send X-Admin-Key")
	}
	r, _ := ctx.Value(mcpHTTPRequestKey{}).(*http.Request)
	if r == nil {
		return errors.New("admin operation requires X-Admin-Key header")
	}
	if r.Header.Get("X-Admin-Key") != required {
		return errors.New("forbidden: invalid or missing X-Admin-Key")
	}
	return nil
}

// Server is an MCP server backed directly by GameService.
type Server struct {
	svc       service.GameService
	hub       *websocket.Hub
	mcpServer *server.MCPServer
}

// NewServer creates an MCP server that calls GameService directly.
func NewServer(svc service.GameService, hubs ...*websocket.Hub) *Server {
	var hub *websocket.Hub
	if len(hubs) > 0 {
		hub = hubs[0]
	}
	s := &Server{svc: svc, hub: hub}
	s.mcpServer = server.NewMCPServer(
		"Tesla Road Trip Game",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(`Tesla Road Trip Game MCP Interface.
Visit all parks (P) to win. Battery depletes 1 per move. H/S restore battery.
Tools: game_state, move, bulk_move, reset_game, move_history, create_session, get_session, list_sessions, list_maps.`),
	)
	s.registerTools()
	return s
}

// Handler returns an http.Handler serving MCP over Streamable HTTP transport.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxMCPRequestBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusRequestEntityTooLarge)
			return
		}
		defer r.Body.Close()

		ctx := context.WithValue(r.Context(), mcpHTTPRequestKey{}, r)
		result := s.mcpServer.HandleMessage(ctx, body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "game_state",
		Description: "Get current game state for a session.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string", "description": "Session ID"},
			},
		},
	}, s.handleGameState)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "move",
		Description: "Move the Tesla one step. Direction: up/down/left/right.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id", "direction"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string"},
				"direction":  map[string]interface{}{"type": "string", "enum": []string{"up", "down", "left", "right"}},
				"reset":      map[string]interface{}{"type": "boolean", "description": "Reset before moving"},
				"intent":     map[string]interface{}{"type": "string", "description": "Explain reasoning"},
			},
		},
	}, s.handleMove)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "bulk_move",
		Description: "Execute multiple moves at once. Each move: up/down/left/right.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id", "moves"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string"},
				"moves":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"reset":      map[string]interface{}{"type": "boolean"},
				"intent":     map[string]interface{}{"type": "string"},
			},
		},
	}, s.handleBulkMove)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "reset_game",
		Description: "Reset session to initial state.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string"},
			},
		},
	}, s.handleReset)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "move_history",
		Description: "Get move history for a session.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string"},
				"limit":      map[string]interface{}{"type": "integer"},
			},
		},
	}, s.handleMoveHistory)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "create_session",
		Description: "Create a new game session.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"map_name"},
			Properties: map[string]interface{}{
				"map_name": map[string]interface{}{"type": "string", "description": "Map/config name (e.g. easy, medium, hard)"},
			},
		},
	}, s.handleCreateSession)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "get_session",
		Description: "Get session details.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"session_id"},
			Properties: map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string"},
			},
		},
	}, s.handleGetSession)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_sessions",
		Description: "List all active game sessions.",
		InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
	}, s.handleListSessions)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_maps",
		Description: "List available game maps/configs.",
		InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
	}, s.handleListMaps)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "get_map",
		Description: "Get full map details including layout, battery settings, and messages.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Map name/ID"},
			},
		},
	}, s.handleGetMap)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "create_map",
		Description: "Create a new map. Layout rows are strings of R/H/P/S/W/B characters. Requires at least one P (park) and one H (home).",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"name", "grid_size", "max_battery", "starting_battery", "layout"},
			Properties: map[string]interface{}{
				"name":                 map[string]interface{}{"type": "string", "description": "Unique map ID (lowercase, underscores)"},
				"description":          map[string]interface{}{"type": "string", "description": "Short description"},
				"grid_size":            map[string]interface{}{"type": "integer", "description": "Grid dimension (e.g. 10 for 10x10)"},
				"max_battery":          map[string]interface{}{"type": "integer", "description": "Maximum battery capacity"},
				"starting_battery":     map[string]interface{}{"type": "integer", "description": "Battery at game start"},
				"layout":               map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Grid rows, one string of cell chars per row"},
				"wall_crash_ends_game": map[string]interface{}{"type": "boolean", "description": "Whether hitting a wall ends the game"},
			},
		},
	}, s.handleCreateMap)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "update_map",
		Description: "Partially update an existing map. Only provided fields are changed; omitted fields keep their current values.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]interface{}{
				"name":                 map[string]interface{}{"type": "string", "description": "Map name/ID to update"},
				"description":          map[string]interface{}{"type": "string", "description": "New description"},
				"max_battery":          map[string]interface{}{"type": "integer", "description": "New max battery"},
				"starting_battery":     map[string]interface{}{"type": "integer", "description": "New starting battery"},
				"layout":               map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New grid layout rows"},
				"wall_crash_ends_game": map[string]interface{}{"type": "boolean", "description": "Wall collision behaviour"},
			},
		},
	}, s.handleUpdateMap)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "delete_map",
		Description: "Permanently delete a map by name.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Map name/ID to delete"},
			},
		},
	}, s.handleDeleteMap)
}

func str(req mcp.CallToolRequest, key string) string {
	v, _ := req.GetArguments()[key].(string)
	return v
}

func boolParam(req mcp.CallToolRequest, key string) bool {
	v, _ := req.GetArguments()[key].(bool)
	return v
}

func intParam(req mcp.CallToolRequest, key string) int {
	v, ok := req.GetArguments()[key].(float64)
	if ok {
		return int(v)
	}
	return 0
}

func extractResponseOptions(req mcp.CallToolRequest) *service.ResponseOptions {
	args := req.GetArguments()
	opts := &service.ResponseOptions{}

	// IncludeGrid - defaults to false for game_state (user must explicitly request)
	if grid, ok := args["grid"].(bool); ok {
		opts.IncludeGrid = grid
	}

	// Minimal response
	if minimal, ok := args["minimal"].(bool); ok {
		opts.Minimal = minimal
	}

	// Include metadata/map
	if metadata, ok := args["metadata"].(bool); ok {
		opts.IncludeGameMap = metadata
	}

	// Include history
	if history, ok := args["history"].(bool); ok {
		opts.IncludeHistory = history
	}

	// History limit
	if limit, ok := args["history_limit"].(float64); ok {
		opts.HistoryLimit = int(limit)
	} else {
		opts.HistoryLimit = 10 // Default
	}

	return opts
}

func extractSessionListOptions(req mcp.CallToolRequest) *service.SessionListOptions {
	args := req.GetArguments()
	opts := &service.SessionListOptions{}

	if includeMaps, ok := args["include_maps"].(bool); ok {
		opts.IncludeMaps = includeMaps
	}

	return opts
}

// filterGameState removes fields based on options
func filterGameState(state *engine.GameState, opts *service.ResponseOptions) *engine.GameState {
	if state == nil {
		return nil
	}
	if opts == nil {
		return state
	}

	stateCopy := *state

	if opts.Minimal || !opts.IncludeGrid {
		stateCopy.Grid = nil
	}

	applyHistoryFilter(&stateCopy, opts)
	return &stateCopy
}

func applyHistoryFilter(state *engine.GameState, opts *service.ResponseOptions) {
	if state == nil {
		return
	}
	if !opts.IncludeHistory {
		state.MoveHistory = nil
		state.CurrentMoves = nil
		state.CurrentMovesCount = 0
		return
	}

	limit := opts.HistoryLimit
	if limit <= 0 {
		limit = 10
	}

	if len(state.MoveHistory) > limit {
		state.MoveHistory = state.MoveHistory[len(state.MoveHistory)-limit:]
	}
	if len(state.CurrentMoves) > limit {
		state.CurrentMoves = state.CurrentMoves[len(state.CurrentMoves)-limit:]
	}
	state.CurrentMovesCount = len(state.CurrentMoves)
}

// filterSessionInfo removes fields based on options
func filterSessionInfo(info *service.SessionInfo, opts *service.ResponseOptions) *service.SessionInfo {
	if opts == nil {
		return info
	}

	infoCopy := *info

	// Don't include GameState by default
	infoCopy.GameState = nil

	// Filter grid if GameState was requested
	if opts.IncludeGameState && infoCopy.GameState != nil {
		if !opts.IncludeGrid {
			stateCopy := *infoCopy.GameState
			stateCopy.Grid = nil
			infoCopy.GameState = &stateCopy
		}
	}

	return &infoCopy
}

// filterMoveResult removes fields based on options
func filterMoveResult(result *service.MoveResult, opts *service.ResponseOptions) *service.MoveResult {
	if opts == nil {
		return result
	}

	resultCopy := *result

	if opts.Minimal {
		// Keep only essential fields
		resultCopy.GameState = nil
	} else if resultCopy.GameState != nil {
		resultCopy.GameState = filterGameState(resultCopy.GameState, opts)
	}

	return &resultCopy
}

// filterBulkMoveResult removes fields based on options
func filterBulkMoveResult(result *service.BulkMoveResult, opts *service.ResponseOptions) *service.BulkMoveResult {
	if opts == nil {
		return result
	}

	resultCopy := *result

	if opts.Minimal {
		// Keep only essential fields
		resultCopy.GameState = nil
	} else if resultCopy.GameState != nil {
		resultCopy.GameState = filterGameState(resultCopy.GameState, opts)
	}

	return &resultCopy
}

func toTOON(v any) string {
	// Convert to JSON-compatible format first (handles custom types)
	jsonBytes, _ := json.Marshal(v)
	var data any
	json.Unmarshal(jsonBytes, &data)

	// Then marshal to TOON
	s, err := toon.MarshalString(data)
	if err != nil {
		return fmt.Sprintf("error: failed to marshal TOON: %v", err)
	}
	return s
}

func toTOONWithOptions(v any, opts *service.ResponseOptions) string {
	// Convert to JSON-compatible format first (handles custom types)
	jsonBytes, _ := json.Marshal(v)
	var data any
	json.Unmarshal(jsonBytes, &data)

	if opts != nil && !opts.IncludeHistory {
		pruneHistoryFields(data)
	}

	s, err := toon.MarshalString(data)
	if err != nil {
		return fmt.Sprintf("error: failed to marshal TOON: %v", err)
	}
	return s
}

func pruneHistoryFields(v any) {
	switch node := v.(type) {
	case map[string]any:
		delete(node, "move_history")
		delete(node, "current_moves")
		delete(node, "current_moves_count")
		for _, child := range node {
			pruneHistoryFields(child)
		}
	case []any:
		for _, child := range node {
			pruneHistoryFields(child)
		}
	}
}

func errResult(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
}

func (s *Server) broadcastToSession(sessionID string, state *engine.GameState) {
	if s.hub == nil {
		return
	}
	s.hub.BroadcastToSession(sessionID, state)
}

func (s *Server) handleGameState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	state, err := s.svc.GetGameState(ctx, str(req, "session_id"))
	if err != nil {
		return errResult(err)
	}
	opts := extractResponseOptions(req)
	filteredState := filterGameState(state, opts)
	return mcp.NewToolResultText(toTOONWithOptions(filteredState, opts)), nil
}

func (s *Server) handleMove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := str(req, "session_id")
	result, err := s.svc.Move(ctx, sessionID, str(req, "direction"), boolParam(req, "reset"))
	if err != nil {
		return errResult(err)
	}
	s.broadcastToSession(sessionID, result.GameState)
	opts := extractResponseOptions(req)
	filteredResult := filterMoveResult(result, opts)
	return mcp.NewToolResultText(toTOONWithOptions(filteredResult, opts)), nil
}

func (s *Server) handleBulkMove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := str(req, "session_id")
	raw, _ := req.GetArguments()["moves"].([]interface{})
	moves := make([]string, 0, len(raw))
	for _, m := range raw {
		if mv, ok := m.(string); ok {
			moves = append(moves, mv)
		}
	}
	result, err := s.svc.BulkMove(ctx, sessionID, moves, boolParam(req, "reset"))
	if err != nil {
		return errResult(err)
	}
	s.broadcastToSession(sessionID, result.GameState)
	opts := extractResponseOptions(req)
	filteredResult := filterBulkMoveResult(result, opts)
	return mcp.NewToolResultText(toTOONWithOptions(filteredResult, opts)), nil
}

func (s *Server) handleReset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := str(req, "session_id")
	state, err := s.svc.Reset(ctx, sessionID)
	if err != nil {
		return errResult(err)
	}
	s.broadcastToSession(sessionID, state)
	opts := extractResponseOptions(req)
	filteredState := filterGameState(state, opts)
	return mcp.NewToolResultText(toTOONWithOptions(filteredState, opts)), nil
}

func (s *Server) handleMoveHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := service.HistoryOptions{}
	if l, ok := req.GetArguments()["limit"].(float64); ok {
		opts.Limit = int(l)
	}
	history, err := s.svc.GetMoveHistory(ctx, str(req, "session_id"), opts)
	if err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(toTOON(history)), nil
}

func (s *Server) handleCreateSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.svc.CreateSession(ctx, str(req, "map_name"))
	if err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(toTOON(session)), nil
}

func (s *Server) handleGetSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.svc.GetSession(ctx, str(req, "session_id"))
	if err != nil {
		return errResult(err)
	}
	opts := extractResponseOptions(req)
	filteredSession := filterSessionInfo(session, opts)
	return mcp.NewToolResultText(toTOONWithOptions(filteredSession, opts)), nil
}

func (s *Server) handleListSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessions, err := s.svc.ListSessions(ctx)
	if err != nil {
		return errResult(err)
	}
	// Filter out GameState from list_sessions (just lists sessions)
	filtered := make([]*service.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		sessCopy := *sess
		sessCopy.GameState = nil
		filtered = append(filtered, &sessCopy)
	}
	return mcp.NewToolResultText(toTOON(filtered)), nil
}

func (s *Server) handleListMaps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	maps, err := s.svc.ListMaps(ctx)
	if err != nil {
		return errResult(err)
	}
	var sb strings.Builder
	for _, m := range maps {
		sb.WriteString(fmt.Sprintf("- %s\n", m.MapID))
	}
	return mcp.NewToolResultText(sb.String() + "\n" + toTOON(maps)), nil
}

func (s *Server) handleGetMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := str(req, "name")
	if name == "" {
		return errResult(fmt.Errorf("name is required"))
	}
	cfg, err := s.svc.LoadMap(ctx, name)
	if err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(toTOON(cfg)), nil
}

func (s *Server) handleCreateMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := checkAdminKey(ctx); err != nil {
		return errResult(err)
	}
	args := req.GetArguments()
	name := str(req, "name")
	if name == "" {
		return errResult(fmt.Errorf("name is required"))
	}
	cfg := &engine.GameConfig{
		Name:              name,
		Description:       str(req, "description"),
		GridSize:          intParam(req, "grid_size"),
		MaxBattery:        intParam(req, "max_battery"),
		StartingBattery:   intParam(req, "starting_battery"),
		WallCrashEndsGame: boolParam(req, "wall_crash_ends_game"),
		Legend:            map[string]string{"R": "road", "H": "home", "P": "park", "S": "supercharger", "W": "water", "B": "building"},
	}
	if rows, ok := args["layout"].([]interface{}); ok {
		cfg.Layout = make([]string, len(rows))
		for i, r := range rows {
			cfg.Layout[i], _ = r.(string)
		}
	}
	if err := s.svc.SaveMap(ctx, name, cfg); err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q created", name)), nil
}

func (s *Server) handleUpdateMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := checkAdminKey(ctx); err != nil {
		return errResult(err)
	}
	args := req.GetArguments()
	name := str(req, "name")
	if name == "" {
		return errResult(fmt.Errorf("name is required"))
	}
	cfg, err := s.svc.LoadMap(ctx, name)
	if err != nil {
		return errResult(fmt.Errorf("map %q not found: %w", name, err))
	}
	if v, ok := args["description"].(string); ok {
		cfg.Description = v
	}
	if v, ok := args["max_battery"].(float64); ok {
		cfg.MaxBattery = int(v)
	}
	if v, ok := args["starting_battery"].(float64); ok {
		cfg.StartingBattery = int(v)
	}
	if v, ok := args["wall_crash_ends_game"].(bool); ok {
		cfg.WallCrashEndsGame = v
	}
	if rows, ok := args["layout"].([]interface{}); ok {
		cfg.Layout = make([]string, len(rows))
		for i, r := range rows {
			cfg.Layout[i], _ = r.(string)
		}
	}
	if err := s.svc.SaveMap(ctx, name, cfg); err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q updated", name)), nil
}

func (s *Server) handleDeleteMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := checkAdminKey(ctx); err != nil {
		return errResult(err)
	}
	name := str(req, "name")
	if name == "" {
		return errResult(fmt.Errorf("name is required"))
	}
	if err := s.svc.DeleteMap(ctx, name); err != nil {
		return errResult(err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("map %q deleted", name)), nil
}
