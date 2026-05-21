package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

// Server serves WebSocket and static UI routes.
// Game operations are exposed through GraphQL at /graphql.
type Server struct {
	service service.GameService
	hub     *websocket.Hub
	router  *mux.Router
}

// NewServer creates a new API server
func NewServer(gameService service.GameService, hub *websocket.Hub) *Server {
	s := &Server{
		service: gameService,
		hub:     hub,
		router:  mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures non-REST routes.
func (s *Server) setupRoutes() {
	// Legacy session list/create routes retained for compatibility with existing tests/tools.
	// Game operations are exposed through GraphQL.
	s.router.HandleFunc("/api/sessions", s.handleCreateSession).Methods(http.MethodPost)
	s.router.HandleFunc("/api/sessions", s.handleListSessions).Methods(http.MethodGet)

	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Serve built SvelteKit frontend with SPA fallback.
	// Prefer ./frontend/build (dev), fall back to ./static (production bundle).
	uiDir := "./frontend/build"
	if _, err := os.Stat(uiDir); err != nil {
		uiDir = "./static"
	}
	fs := http.FileServer(http.Dir(uiDir))
	s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(uiDir, filepath.Clean("/"+r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Response helpers
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Session Handlers

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MapID   string `json:"map_id,omitempty"`
		MapName string `json:"map_name,omitempty"`
	}

	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	mapID := req.MapID
	if mapID == "" && req.MapName != "" {
		mapID = req.MapName
	}

	session, err := s.service.CreateSession(r.Context(), mapID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, session)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.service.ListSessions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	sortBy := query.Get("sort")    // "created", "accessed" (default)
	order := query.Get("order")    // "asc", "desc" (default: "desc")
	limitStr := query.Get("limit") // number of sessions to return

	// Set defaults
	if sortBy == "" {
		sortBy = "accessed"
	}
	if order == "" {
		order = "desc"
	}

	// Sort sessions
	sort.Slice(sessions, func(i, j int) bool {
		var ti, tj time.Time
		if sortBy == "created" {
			ti, tj = sessions[i].CreatedAt, sessions[j].CreatedAt
		} else { // "accessed"
			ti, tj = sessions[i].LastAccessedAt, sessions[j].LastAccessedAt
		}

		if order == "asc" {
			return ti.Before(tj)
		}
		return ti.After(tj) // desc
	})

	// Apply limit if specified
	limit := len(sessions)
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l < len(sessions) {
			limit = l
		}
	}
	sessions = sessions[:limit]

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(sessions),
		"total":    len(sessions),
		"sessions": sessions,
		"sort":     sortBy,
		"order":    order,
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := s.service.GetSession(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, session)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	err := s.service.DeleteSession(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Session %s deleted", sessionID),
	})
}

// Game Operation Handlers

func (s *Server) handleGetGameState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	state, err := s.service.GetGameState(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, state)
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		Direction string `json:"direction"`
		Reset     bool   `json:"reset,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.service.Move(r.Context(), sessionID, req.Direction, req.Reset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast to WebSocket clients
	if s.hub != nil {
		s.hub.BroadcastToSession(sessionID, result.GameState)
	}

	// Compact server log for observability
	if result.Step != nil {
		s := result.Step
		status := "FAIL"
		if result.Success {
			status = "OK"
		}
		fmt.Printf("[MOVE] session=%s %s (%d,%d)->(%d,%d) tile=%s batt=%d status=%s\n",
			sessionID, s.Dir, s.From.X, s.From.Y, s.To.X, s.To.Y, s.TileChar, s.BatteryAfter, status)
	} else if result.AttemptedTo != nil {
		a := result.AttemptedTo
		fmt.Printf("[MOVE] session=%s BLOCKED attempt=(%d,%d) tile=%s type=%s\n",
			sessionID, a.X, a.Y, a.TileChar, a.TileType)
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleBulkMove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		Moves []string `json:"moves"`
		Reset bool     `json:"reset,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.service.BulkMove(r.Context(), sessionID, req.Moves, req.Reset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast to WebSocket clients
	if s.hub != nil {
		s.hub.BroadcastToSession(sessionID, result.GameState)
	}

	// Compact server log for observability
	requested := result.RequestedMoves
	if requested == 0 {
		requested = result.TotalMoves
	}
	stop := result.StopReasonCode
	if stop == "" && result.StoppedReason != "" {
		stop = "stopped"
	}
	fmt.Printf("[BULK] session=%s exec=%d/%d stop=%s end=(%d,%d) batt=%d scoreΔ=%d\n",
		sessionID, result.MovesExecuted, requested, stop, result.GameState.PlayerPos.X, result.GameState.PlayerPos.Y, result.GameState.Battery, result.ScoreDelta)

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	state, err := s.service.Reset(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// Broadcast to WebSocket clients
	if s.hub != nil {
		s.hub.BroadcastToSession(sessionID, state)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Game reset successfully",
		"state":   state,
	})
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	// Parse query parameters
	opts := service.HistoryOptions{
		Page:  1,
		Limit: 20,
		Order: "desc",
	}

	query := r.URL.Query()
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			opts.Page = p
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			opts.Limit = l
		}
	}

	if order := query.Get("order"); order == "asc" || order == "desc" {
		opts.Order = order
	}

	history, err := s.service.GetMoveHistory(r.Context(), sessionID, opts)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// Map Handlers

func (s *Server) handleListMaps(w http.ResponseWriter, r *http.Request) {
	maps, err := s.service.ListMaps(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, maps)
}

func (s *Server) handleGetMap(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mapName := vars["name"]

	mapName = strings.TrimSuffix(mapName, ".json")

	config, err := s.service.LoadMap(r.Context(), mapName)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, config)
}

func (s *Server) handleCreateMap(w http.ResponseWriter, r *http.Request) {
	var gameConfig engine.GameConfig

	if err := json.NewDecoder(r.Body).Decode(&gameConfig); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if gameConfig.Name == "" {
		respondError(w, http.StatusBadRequest, "Map name is required")
		return
	}

	if err := s.service.SaveMap(r.Context(), gameConfig.Name, &gameConfig); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save map: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Map saved successfully",
		"map_id":  gameConfig.Name,
	})
}

// Unified Sessions Handler

func (s *Server) handleUnifiedSessions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Get sessions based on query parameters
	var sessions []*service.SessionInfo

	if sessionIDs := query.Get("sessionIds"); sessionIDs != "" {
		// Get specific sessions by IDs
		ids := strings.Split(sessionIDs, ",")
		sessions = make([]*service.SessionInfo, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				session, err := s.service.GetSession(r.Context(), id)
				if err == nil {
					sessions = append(sessions, session)
				}
			}
		}
	} else if mapName := query.Get("mapName"); mapName != "" {
		allSessions, err := s.service.ListSessions(r.Context())
		if err == nil {
			sessions = make([]*service.SessionInfo, 0)
			for _, session := range allSessions {
				if session.MapName == mapName {
					sessions = append(sessions, session)
				}
			}
		}
	} else {
		// Get all sessions
		allSessions, err := s.service.ListSessions(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sessions = allSessions
	}

	mapName := ""
	totalParks := 0

	if len(sessions) > 0 {
		mapName = sessions[0].MapName

		if sessions[0].GameMap != nil && sessions[0].GameMap.Layout != nil {
			for _, row := range sessions[0].GameMap.Layout {
				for _, cell := range row {
					if cell == 'P' {
						totalParks++
					}
				}
			}
		}
	}

	response := map[string]interface{}{
		"map_name":    mapName,
		"total_parks": totalParks,
		"sessions":    make([]map[string]interface{}, 0, len(sessions)),
	}

	for _, session := range sessions {
		sessionData := map[string]interface{}{
			"session_id":    session.ID,
			"map_name":      session.MapName,
			"game_state":    session.GameState,
			"created_at":    session.CreatedAt,
			"last_accessed": session.LastAccessedAt,
		}
		response["sessions"] = append(response["sessions"].([]map[string]interface{}), sessionData)
	}

	respondJSON(w, http.StatusOK, response)
}

// WebSocket Handler

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "session parameter required", http.StatusBadRequest)
		return
	}

	// Verify session exists
	_, err := s.service.GetSession(context.Background(), sessionID)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket. httptest.ResponseRecorder does not implement
	// http.Hijacker; return an explicit 500 so unit tests can assert that the
	// request reached the upgrade path without requiring a real network socket.
	if _, ok := w.(http.Hijacker); !ok {
		http.Error(w, "websocket upgrade unavailable", http.StatusInternalServerError)
		return
	}
	s.hub.ServeWS(w, r, sessionID)
}

// Health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
