package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	cellSize          = 40
	headerHeight      = 80 // Taller header for multi-session stats
	screenWidth       = 800
	screenHeight      = 720
	baseURL           = "http://localhost:8080"
	animationDuration = 150 * time.Millisecond // Smooth animation duration
	crashDuration     = 400 * time.Millisecond // Crash animation duration
)

// ScreenType represents different screens in the app
type ScreenType int

const (
	ScreenWelcome ScreenType = iota
	ScreenGame
)

// Car colors for different sessions
var carColors = []color.RGBA{
	{255, 100, 100, 255}, // Red
	{100, 100, 255, 255}, // Blue
	{100, 255, 100, 255}, // Green
	{255, 255, 100, 255}, // Yellow
	{255, 100, 255, 255}, // Magenta
	{100, 255, 255, 255}, // Cyan
	{255, 165, 0, 255},   // Orange
	{128, 0, 128, 255},   // Purple
	{255, 192, 203, 255}, // Pink
}

// Cell represents a grid cell
type Cell struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Visited bool   `json:"visited,omitempty"`
}

// GameState represents the state from the Tesla game server
type GameState struct {
	Grid        [][]Cell `json:"grid"`
	PlayerPos   Position `json:"player_pos"`
	Battery     int      `json:"battery"`
	MaxBattery  int      `json:"max_battery"`
	Score       int      `json:"score"`
	GameOver    bool     `json:"game_over"`
	Victory     bool     `json:"victory"`
	Message     string   `json:"message"`
	ConfigName  string   `json:"config_name"`
	MoveHistory []Move   `json:"move_history,omitempty"`
}

// Move represents a single move in history
type Move struct {
	Action     string   `json:"action"`
	MoveNumber int      `json:"move_number"`
	Battery    int      `json:"battery"`
	Success    bool     `json:"success"`
	FromPos    Position `json:"from_position"`
	ToPos      Position `json:"to_position"`
}

// Position represents player position
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WSMessage represents WebSocket message wrapper
type WSMessage struct {
	SessionID string     `json:"session_id"`
	GameState *GameState `json:"game_state,omitempty"`
	Event     string     `json:"event,omitempty"`
}

// SessionData holds data for a single session
type SessionData struct {
	sessionID     string
	state         *GameState
	wsConn        *websocket.Conn
	lastUpdate    time.Time
	prevPos       Position  // Previous position for interpolation
	targetPos     Position  // Target position for interpolation
	moveStartTime time.Time // When the move started
	animationTime float64   // Animation progress 0.0 to 1.0
	crashTime     time.Time // When a crash happened
	isCrashing    bool      // Currently showing crash animation
}

// SessionListItem represents a session from the server
type SessionListItem struct {
	ID         string `json:"id"`
	ConfigName string `json:"config_name"`
	CreatedAt  string `json:"created_at"`
	Battery    int    `json:"battery"`
	Score      int    `json:"score"`
	Victory    bool   `json:"victory"`
	GameOver   bool   `json:"game_over"`
}

// ConfigListItem represents a game configuration
type ConfigListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Game represents the desktop game client
type Game struct {
	sessions         []*SessionData
	activeSession    int // index of currently active session
	stateMutex       sync.RWMutex
	currentScreen    ScreenType
	welcomeScreen    *WelcomeScreen
	selectedSessions map[string]bool // session IDs selected to play
}

// WelcomeScreen manages the welcome screen state
type WelcomeScreen struct {
	availableSessions []SessionListItem
	availableConfigs  []ConfigListItem
	selectedConfigs   map[string]bool // for creating new sessions
	scrollOffset      int
	cursorPos         int
	loading           bool
	errorMsg          string
	newSessionConfig  string // selected config for new session
}

// NewGame creates a new game instance with initial sessions
func NewGame(sessionIDs []string) *Game {
	g := &Game{
		sessions:         make([]*SessionData, 0),
		activeSession:    0,
		currentScreen:    ScreenWelcome,
		selectedSessions: make(map[string]bool),
		welcomeScreen: &WelcomeScreen{
			availableSessions: make([]SessionListItem, 0),
			availableConfigs:  make([]ConfigListItem, 0),
			selectedConfigs:   make(map[string]bool),
			cursorPos:         0,
			scrollOffset:      0,
		},
	}

	// If session IDs provided, skip welcome screen and go straight to game
	if len(sessionIDs) > 0 {
		for _, sid := range sessionIDs {
			g.addSession(sid)
		}
		g.currentScreen = ScreenGame
	} else {
		// Load available sessions and configs for welcome screen
		g.loadWelcomeData()
	}

	return g
}

// addSession adds a new session to the game with optional config
func (g *Game) addSession(sessionID string) {
	session := &SessionData{
		sessionID:  sessionID,
		lastUpdate: time.Now(),
	}

	// If no session ID provided, create one with same config as first session
	if sessionID == "" {
		configName := ""
		if len(g.sessions) > 0 && g.sessions[0].state != nil {
			configName = g.sessions[0].state.ConfigName
		}
		if err := g.createSessionWithConfig(session, configName); err != nil {
			log.Printf("Failed to create session: %v", err)
			return
		}
	}

	g.sessions = append(g.sessions, session)

	// Connect to WebSocket
	if err := g.connectWebSocket(session); err != nil {
		log.Printf("Failed to connect WebSocket for %s: %v (falling back to polling)", session.sessionID, err)
	} else {
		// Start WebSocket listener
		go g.listenWebSocket(session)
	}

	// Initial state fetch
	g.fetchGameState(session)
}

// createSession creates a new game session
func (g *Game) createSession(session *SessionData) error {
	return g.createSessionWithConfig(session, "")
}

// createSessionWithConfig creates a new game session with specific config
func (g *Game) createSessionWithConfig(session *SessionData, configName string) error {
	url := fmt.Sprintf("%s/api/sessions", baseURL)

	payload := "{}"
	if configName != "" {
		payload = fmt.Sprintf(`{"config_name":"%s"}`, configName)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse session response: %v (body: %s)", err, string(body))
	}

	session.sessionID = result.SessionID
	log.Printf("Created new session: %s (config: %s)", session.sessionID, configName)
	return nil
}

// connectWebSocket establishes WebSocket connection
func (g *Game) connectWebSocket(session *SessionData) error {
	if session.sessionID == "" {
		return fmt.Errorf("no session ID set")
	}

	wsURL := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	q := wsURL.Query()
	q.Set("session", session.sessionID)
	wsURL.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		return err
	}

	session.wsConn = conn
	log.Printf("WebSocket connected for session %s", session.sessionID)
	return nil
}

// listenWebSocket listens for WebSocket updates
func (g *Game) listenWebSocket(session *SessionData) {
	defer func() {
		if session.wsConn != nil {
			session.wsConn.Close()
		}
	}()

	for {
		_, message, err := session.wsConn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for %s: %v", session.sessionID, err)
			return
		}

		// WebSocket sends wrapped message
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("WebSocket JSON parse error: %v", err)
			continue
		}

		if wsMsg.GameState == nil {
			log.Printf("WebSocket message has no game_state field")
			continue
		}

		g.stateMutex.Lock()
		// Check if position changed for animation
		if session.state != nil {
			oldPos := session.state.PlayerPos
			newPos := wsMsg.GameState.PlayerPos
			oldMoves := len(session.state.MoveHistory)
			newMoves := len(wsMsg.GameState.MoveHistory)

			if oldPos.X != newPos.X || oldPos.Y != newPos.Y {
				// Position changed - start move animation
				session.prevPos = oldPos
				session.targetPos = newPos
				session.moveStartTime = time.Now()
				session.animationTime = 0.0
				session.isCrashing = false
			} else if newMoves > oldMoves {
				// Move was attempted but position didn't change - CRASH!
				session.crashTime = time.Now()
				session.isCrashing = true
			}
		} else {
			// First state - no animation
			session.targetPos = wsMsg.GameState.PlayerPos
			session.prevPos = wsMsg.GameState.PlayerPos
			session.animationTime = 1.0
		}
		session.state = wsMsg.GameState
		session.lastUpdate = time.Now()
		g.stateMutex.Unlock()
	}
}

// fetchGameState gets the current game state from the server
func (g *Game) fetchGameState(session *SessionData) error {
	if session.sessionID == "" {
		return fmt.Errorf("no session ID set")
	}

	url := fmt.Sprintf("%s/api/sessions/%s/state", baseURL, session.sessionID)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var state GameState
	if err := json.Unmarshal(body, &state); err != nil {
		return fmt.Errorf("failed to parse JSON: %v (body: %s)", err, string(body))
	}

	g.stateMutex.Lock()
	// Check if position changed for animation
	if session.state != nil {
		oldPos := session.state.PlayerPos
		newPos := state.PlayerPos
		oldMoves := len(session.state.MoveHistory)
		newMoves := len(state.MoveHistory)

		if oldPos.X != newPos.X || oldPos.Y != newPos.Y {
			// Position changed - start move animation
			session.prevPos = oldPos
			session.targetPos = newPos
			session.moveStartTime = time.Now()
			session.animationTime = 0.0
			session.isCrashing = false
		} else if newMoves > oldMoves {
			// Move was attempted but position didn't change - CRASH!
			session.crashTime = time.Now()
			session.isCrashing = true
		}
	} else {
		// First state - no animation
		session.targetPos = state.PlayerPos
		session.prevPos = state.PlayerPos
		session.animationTime = 1.0
	}
	session.state = &state
	session.lastUpdate = time.Now()
	g.stateMutex.Unlock()

	return nil
}

// loadWelcomeData fetches available sessions and configs from server
func (g *Game) loadWelcomeData() {
	g.welcomeScreen.loading = true
	g.welcomeScreen.errorMsg = ""

	// Fetch available sessions
	resp, err := http.Get(fmt.Sprintf("%s/api/sessions", baseURL))
	if err != nil {
		g.welcomeScreen.errorMsg = fmt.Sprintf("Error loading sessions: %v", err)
		g.welcomeScreen.loading = false
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var sessionsResp struct {
		Sessions []SessionListItem `json:"sessions"`
	}
	if err := json.Unmarshal(body, &sessionsResp); err == nil {
		g.welcomeScreen.availableSessions = sessionsResp.Sessions
	}

	// Fetch available configs
	resp, err = http.Get(fmt.Sprintf("%s/api/configs", baseURL))
	if err != nil {
		g.welcomeScreen.errorMsg = fmt.Sprintf("Error loading configs: %v", err)
		g.welcomeScreen.loading = false
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	var configsResp struct {
		Configs []ConfigListItem `json:"configs"`
	}
	if err := json.Unmarshal(body, &configsResp); err == nil {
		g.welcomeScreen.availableConfigs = configsResp.Configs
	}

	g.welcomeScreen.loading = false
}

// createNewSessionFromWelcome creates a new session with selected config
func (g *Game) createNewSessionFromWelcome() error {
	configName := g.welcomeScreen.newSessionConfig
	url := fmt.Sprintf("%s/api/sessions", baseURL)

	payload := "{}"
	if configName != "" {
		payload = fmt.Sprintf(`{"config_name":"%s"}`, configName)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse session response: %v", err)
	}

	// Add to selected sessions
	g.selectedSessions[result.SessionID] = true
	log.Printf("Created new session: %s (config: %s)", result.SessionID, configName)

	// Reload session list
	g.loadWelcomeData()
	return nil
}

// startGameWithSelectedSessions transitions to game screen with selected sessions
func (g *Game) startGameWithSelectedSessions() {
	if len(g.selectedSessions) == 0 {
		g.welcomeScreen.errorMsg = "Please select at least one session"
		return
	}

	// Create sessions for each selected ID
	for sessionID := range g.selectedSessions {
		g.addSession(sessionID)
	}

	// Switch to game screen
	g.currentScreen = ScreenGame
}

// sendAction sends a move action to the server for active session
func (g *Game) sendAction(action string) error {
	if len(g.sessions) == 0 {
		return fmt.Errorf("no sessions available")
	}

	session := g.sessions[g.activeSession]
	if session.sessionID == "" {
		return fmt.Errorf("no session ID set")
	}

	var url string
	var payload string

	if action == "reset" {
		url = fmt.Sprintf("%s/api/sessions/%s/reset", baseURL, session.sessionID)
		payload = "{}"
	} else {
		url = fmt.Sprintf("%s/api/sessions/%s/move", baseURL, session.sessionID)
		payload = fmt.Sprintf(`{"direction":"%s"}`, action)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return g.fetchGameState(session)
}

// Update updates game logic
func (g *Game) Update() error {
	// Route to appropriate screen update
	switch g.currentScreen {
	case ScreenWelcome:
		return g.updateWelcomeScreen()
	case ScreenGame:
		return g.updateGameScreen()
	}
	return nil
}

// updateWelcomeScreen handles welcome screen input
func (g *Game) updateWelcomeScreen() error {
	ws := g.welcomeScreen

	// Refresh data with F5
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.loadWelcomeData()
	}

	// Navigate with arrow keys
	totalItems := len(ws.availableSessions)
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		ws.cursorPos++
		if ws.cursorPos >= totalItems {
			ws.cursorPos = totalItems - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		ws.cursorPos--
		if ws.cursorPos < 0 {
			ws.cursorPos = 0
		}
	}

	// Toggle selection with Space
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if ws.cursorPos < len(ws.availableSessions) {
			sessionID := ws.availableSessions[ws.cursorPos].ID
			g.selectedSessions[sessionID] = !g.selectedSessions[sessionID]
			if !g.selectedSessions[sessionID] {
				delete(g.selectedSessions, sessionID)
			}
		}
	}

	// Cycle through configs with Tab
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if len(ws.availableConfigs) > 0 {
			// Find current config index
			currentIdx := -1
			for i, cfg := range ws.availableConfigs {
				if cfg.Name == ws.newSessionConfig {
					currentIdx = i
					break
				}
			}
			// Move to next
			currentIdx++
			if currentIdx >= len(ws.availableConfigs) {
				ws.newSessionConfig = "" // No config (default)
			} else {
				ws.newSessionConfig = ws.availableConfigs[currentIdx].Name
			}
		}
	}

	// Create new session with N
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if err := g.createNewSessionFromWelcome(); err != nil {
			ws.errorMsg = fmt.Sprintf("Failed to create session: %v", err)
		}
	}

	// Start game with Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.startGameWithSelectedSessions()
	}

	// Back to game screen with Escape (if sessions exist)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && len(g.sessions) > 0 {
		g.currentScreen = ScreenGame
	}

	return nil
}

// updateGameScreen handles game screen input
func (g *Game) updateGameScreen() error {
	if len(g.sessions) == 0 {
		return nil
	}

	// Update animation progress for all sessions
	g.stateMutex.Lock()
	for _, session := range g.sessions {
		if session.animationTime < 1.0 {
			elapsed := time.Since(session.moveStartTime)
			session.animationTime = float64(elapsed) / float64(animationDuration)
			if session.animationTime > 1.0 {
				session.animationTime = 1.0
			}
		}

		// End crash animation after duration
		if session.isCrashing && time.Since(session.crashTime) > crashDuration {
			session.isCrashing = false
		}
	}
	g.stateMutex.Unlock()

	// Poll all sessions if WebSocket is not connected
	for _, session := range g.sessions {
		if session.wsConn == nil {
			if session.state == nil || time.Since(session.lastUpdate) > 500*time.Millisecond {
				if err := g.fetchGameState(session); err != nil {
					log.Printf("Error fetching state for %s: %v", session.sessionID, err)
				}
			}
		}
	}

	// Session switching with number keys (1-9)
	for i := ebiten.Key1; i <= ebiten.Key9; i++ {
		if inpututil.IsKeyJustPressed(i) {
			sessionIdx := int(i - ebiten.Key1)
			if sessionIdx < len(g.sessions) {
				g.activeSession = sessionIdx
				log.Printf("Switched to session %d: %s", sessionIdx+1, g.sessions[sessionIdx].sessionID)
			}
		}
	}

	// Add new session with N key
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if len(g.sessions) < 9 {
			g.addSession("")
			log.Printf("Added new session (total: %d)", len(g.sessions))
		}
	}

	// Handle keyboard input for active session
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.sendAction("up")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.sendAction("down")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.sendAction("left")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.sendAction("right")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.sendAction("reset")
	}

	// Return to welcome screen with Escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.currentScreen = ScreenWelcome
		g.loadWelcomeData()
	}

	return nil
}

// Draw renders the game
func (g *Game) Draw(screen *ebiten.Image) {
	// Route to appropriate screen renderer
	switch g.currentScreen {
	case ScreenWelcome:
		g.drawWelcomeScreen(screen)
	case ScreenGame:
		g.drawGameScreen(screen)
	}
}

// drawWelcomeScreen renders the welcome/session selection screen
func (g *Game) drawWelcomeScreen(screen *ebiten.Image) {
	ws := g.welcomeScreen

	// Clear screen
	screen.Fill(color.RGBA{20, 20, 30, 255})

	y := 20
	ebitenutil.DebugPrintAt(screen, "=== TESLA ROAD TRIP - SESSION SELECT ===", 200, y)
	y += 30

	if ws.loading {
		ebitenutil.DebugPrintAt(screen, "Loading sessions...", 20, y)
		return
	}

	if ws.errorMsg != "" {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("ERROR: %s", ws.errorMsg), 20, y)
		y += 20
	}

	// Session list
	ebitenutil.DebugPrintAt(screen, "Available Sessions:", 20, y)
	y += 20

	if len(ws.availableSessions) == 0 {
		ebitenutil.DebugPrintAt(screen, "  No sessions found. Press N to create one.", 20, y)
		y += 20
	} else {
		for i, session := range ws.availableSessions {
			cursor := "  "
			if i == ws.cursorPos {
				cursor = "> "
			}

			checkbox := "[ ]"
			if g.selectedSessions[session.ID] {
				checkbox = "[X]"
			}

			status := ""
			if session.Victory {
				status = " VICTORY"
			} else if session.GameOver {
				status = " GAME OVER"
			}

			line := fmt.Sprintf("%s%s %s | %s | Battery:%d Score:%d%s",
				cursor, checkbox, session.ID, session.ConfigName,
				session.Battery, session.Score, status)

			ebitenutil.DebugPrintAt(screen, line, 20, y)
			y += 15
		}
	}

	y += 20
	ebitenutil.DebugPrintAt(screen, "─────────────────────────────────────────", 20, y)
	y += 20

	// New session creation
	ebitenutil.DebugPrintAt(screen, "Create New Session:", 20, y)
	y += 20

	configDisplay := "default"
	if ws.newSessionConfig != "" {
		configDisplay = ws.newSessionConfig
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("  Selected Config: %s", configDisplay), 20, y)
	y += 15

	ebitenutil.DebugPrintAt(screen, "  Available Configs:", 20, y)
	y += 15
	for _, cfg := range ws.availableConfigs {
		marker := "  "
		if cfg.Name == ws.newSessionConfig {
			marker = "→ "
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("    %s%s - %s", marker, cfg.Name, cfg.Description), 20, y)
		y += 15
	}

	y += 20
	ebitenutil.DebugPrintAt(screen, "─────────────────────────────────────────", 20, y)
	y += 20

	// Selected sessions summary
	selectedCount := len(g.selectedSessions)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Selected: %d session(s)", selectedCount), 20, y)
	y += 20

	// Controls
	y += 10
	ebitenutil.DebugPrintAt(screen, "CONTROLS:", 20, y)
	y += 20
	ebitenutil.DebugPrintAt(screen, "  ↑/↓      - Navigate sessions", 20, y)
	y += 15
	ebitenutil.DebugPrintAt(screen, "  SPACE    - Toggle session selection", 20, y)
	y += 15
	ebitenutil.DebugPrintAt(screen, "  TAB      - Cycle config for new session", 20, y)
	y += 15
	ebitenutil.DebugPrintAt(screen, "  N        - Create new session with selected config", 20, y)
	y += 15
	ebitenutil.DebugPrintAt(screen, "  ENTER    - Start game with selected sessions", 20, y)
	y += 15
	ebitenutil.DebugPrintAt(screen, "  F5       - Refresh session list", 20, y)
	y += 15
	if len(g.sessions) > 0 {
		ebitenutil.DebugPrintAt(screen, "  ESC      - Back to game", 20, y)
	}
}

// drawGameScreen renders the game screen
func (g *Game) drawGameScreen(screen *ebiten.Image) {
	g.stateMutex.RLock()
	defer g.stateMutex.RUnlock()

	if len(g.sessions) == 0 {
		ebitenutil.DebugPrint(screen, "No sessions available. Press ESC to go to session select.")
		return
	}

	// Use first session's grid as reference (all should have same map)
	refSession := g.sessions[0]
	if refSession.state == nil {
		ebitenutil.DebugPrint(screen, "Loading...")
		return
	}

	// Draw header with all session stats
	g.drawSessionStats(screen)

	// Build park collection map: park_id -> list of session indices that collected it
	parkCollectors := make(map[string][]int)
	for idx, session := range g.sessions {
		if session.state == nil {
			continue
		}
		for _, row := range session.state.Grid {
			for _, cell := range row {
				if cell.Type == "park" && cell.Visited {
					parkCollectors[cell.ID] = append(parkCollectors[cell.ID], idx)
				}
			}
		}
	}

	// Draw the grid once (all cars share same map)
	gridOffsetY := headerHeight
	for y, row := range refSession.state.Grid {
		for x, cell := range row {
			// Base cell color
			cellColor := getCellColor(cell.Type, false)
			ebitenutil.DrawRect(screen,
				float64(x*cellSize),
				float64(y*cellSize+gridOffsetY),
				cellSize-1, cellSize-1, cellColor)

			// If it's a park, show who collected it
			if cell.Type == "park" {
				collectors := parkCollectors[cell.ID]
				if len(collectors) > 0 {
					// Show player numbers who collected this park
					collectorText := ""
					for _, sessionIdx := range collectors {
						collectorText += fmt.Sprintf("%d", sessionIdx+1)
					}
					ebitenutil.DebugPrintAt(screen,
						collectorText,
						x*cellSize+10,
						y*cellSize+gridOffsetY+12)
				}
			}
		}
	}

	// Draw trails for each car (before drawing the cars themselves)
	for idx, session := range g.sessions {
		if session.state == nil || len(session.state.MoveHistory) == 0 {
			continue
		}

		carColor := carColors[idx%len(carColors)]
		history := session.state.MoveHistory
		maxTrailLength := 200 // Show last 200 moves

		// Start from the most recent moves
		startIdx := 0
		if len(history) > maxTrailLength {
			startIdx = len(history) - maxTrailLength
		}

		// Draw trail dots with fading opacity
		for i := startIdx; i < len(history); i++ {
			move := history[i]
			if !move.Success {
				continue // Skip failed moves
			}

			// Calculate opacity based on age (newer = more opaque)
			age := float64(i - startIdx)
			maxAge := float64(len(history) - startIdx)
			opacity := (age / maxAge) * 0.4 // Fade from 0 to 0.4 alpha

			// Create faded color
			trailColor := color.RGBA{
				R: carColor.R,
				G: carColor.G,
				B: carColor.B,
				A: uint8(opacity * 255),
			}

			// Draw small trail dot at the ToPos
			dotSize := 6.0
			dotX := float64(move.ToPos.X*cellSize) + float64(cellSize)/2 - dotSize/2
			dotY := float64(move.ToPos.Y*cellSize) + float64(gridOffsetY) + float64(cellSize)/2 - dotSize/2

			ebitenutil.DrawRect(screen, dotX, dotY, dotSize, dotSize, trailColor)
		}
	}

	// Draw all cars on the grid with smooth interpolation
	for idx, session := range g.sessions {
		if session.state == nil {
			continue
		}

		// Interpolate position for smooth animation
		t := session.animationTime
		if t > 1.0 {
			t = 1.0
		}

		// Linear interpolation between previous and target position
		displayX := float64(session.prevPos.X)*(1.0-t) + float64(session.targetPos.X)*t
		displayY := float64(session.prevPos.Y)*(1.0-t) + float64(session.targetPos.Y)*t

		// Get color for this car
		carColor := carColors[idx%len(carColors)]

		// Crash animation: shake and flash
		var shakeX, shakeY float64
		if session.isCrashing {
			crashProgress := time.Since(session.crashTime).Seconds() / crashDuration.Seconds()
			// Shake effect (dampening over time)
			shakeIntensity := 4.0 * (1.0 - crashProgress)
			shakeX = shakeIntensity * math.Sin(crashProgress*40) // Fast shake
			shakeY = shakeIntensity * math.Cos(crashProgress*40)

			// Flash red color
			flashAmount := (1.0 - crashProgress) * 0.7
			carColor.R = uint8(float64(carColor.R)*(1.0-flashAmount) + 255*flashAmount)
		}

		// Draw car with session number (interpolated position + shake)
		screenX := displayX*float64(cellSize) + 3 + shakeX
		screenY := displayY*float64(cellSize) + float64(gridOffsetY) + 3 + shakeY

		ebitenutil.DrawRect(screen,
			screenX,
			screenY,
			cellSize-6,
			cellSize-6,
			carColor)

		// Draw session number on car
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("%d", idx+1),
			int(screenX)+9,
			int(screenY)+9)
	}

	// Footer controls
	ebitenutil.DebugPrintAt(screen, "1-9: Switch Car | N: New Car | Arrow/WASD: Move | R: Reset | ESC: Menu", 10, screenHeight-20)
}

// drawSessionStats draws stats for all sessions in header
func (g *Game) drawSessionStats(screen *ebiten.Image) {
	headerY := 5
	for idx, session := range g.sessions {
		if session.state == nil {
			continue
		}

		y := headerY + (idx * 15)
		carColor := carColors[idx%len(carColors)]

		// Draw color indicator
		ebitenutil.DrawRect(screen, 5, float64(y), 10, 10, carColor)

		// Session info
		activeMarker := ""
		if idx == g.activeSession {
			activeMarker = ">>>"
		}

		connStatus := "POLL"
		if session.wsConn != nil {
			connStatus = "WS"
		}

		totalMoves := len(session.state.MoveHistory)

		info := fmt.Sprintf("%s [%d] %s [%s] BAT:%d/%d MV:%d SC:%d",
			activeMarker,
			idx+1,
			session.sessionID,
			connStatus,
			session.state.Battery,
			session.state.MaxBattery,
			totalMoves,
			session.state.Score)

		if session.state.Victory {
			info += " VICTORY!"
		} else if session.state.GameOver {
			info += " GAME OVER"
		}

		ebitenutil.DebugPrintAt(screen, info, 20, y)
	}
}

// Layout returns the game screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// getCellColor returns the color for each cell type
func getCellColor(cellType string, visited bool) color.Color {
	switch cellType {
	case "road":
		return color.RGBA{128, 128, 128, 255} // Gray for road
	case "home":
		return color.RGBA{0, 200, 0, 255} // Green for home
	case "supercharger":
		return color.RGBA{255, 0, 0, 255} // Red for supercharger
	case "water":
		return color.RGBA{0, 100, 200, 255} // Blue for water
	case "building":
		return color.RGBA{100, 50, 0, 255} // Brown for building
	case "park":
		if visited {
			return color.RGBA{100, 100, 100, 255} // Gray for collected parks
		}
		return color.RGBA{255, 165, 0, 255} // Orange for uncollected parks
	default:
		return color.RGBA{50, 50, 50, 255} // Dark gray for unknown
	}
}

func main() {
	// Accept multiple session IDs as arguments
	sessionIDs := []string{}
	if len(os.Args) > 1 {
		sessionIDs = os.Args[1:]
	}

	game := NewGame(sessionIDs)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Tesla Road Trip - Multi-Session Desktop Client")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
