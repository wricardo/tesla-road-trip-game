package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Cell struct {
	Type    string `json:"type"`
	Visited bool   `json:"visited,omitempty"`
	ID      string `json:"id,omitempty"`
}

type GameState struct {
	Grid         [][]Cell        `json:"grid"`
	PlayerPos    Position        `json:"player_pos"`
	Battery      int             `json:"battery"`
	MaxBattery   int             `json:"max_battery"`
	Score        int             `json:"score"`
	GameOver     bool            `json:"game_over"`
	Victory      bool            `json:"victory"`
	Message      string          `json:"message"`
	VisitedParks map[string]bool `json:"visited_parks"`
	ConfigName   string          `json:"config_name"`
}

type SessionResponse struct {
	ID         string     `json:"id"`
	ConfigName string     `json:"config_name"`
	GameState  *GameState `json:"game_state"`
}

type MoveRequest struct {
	Direction  string   `json:"direction,omitempty"`
	Directions []string `json:"directions,omitempty"`
	Reset      bool     `json:"reset,omitempty"`
}

type Client struct {
	baseURL   string
	sessionID string
	client    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) CreateSession(configName string) (*GameState, error) {
	var reqBody []byte
	var err error

	if configName != "" {
		reqBody, err = json.Marshal(map[string]string{"config_name": configName})
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	resp, err := c.client.Post(c.baseURL+"/api/sessions", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create session failed: %s - %s", resp.Status, string(body))
	}

	var session SessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("parse session response: %w", err)
	}

	c.sessionID = session.ID
	return session.GameState, nil
}

func (c *Client) GetState() (*GameState, error) {
	url := fmt.Sprintf("%s/api/sessions/%s", c.baseURL, c.sessionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	defer resp.Body.Close()

	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}

	return session.GameState, nil
}

func (c *Client) Move(direction string) (*GameState, error) {
	req := MoveRequest{Direction: direction}
	return c.executeMove(req)
}

func (c *Client) BulkMove(directions []string) (*GameState, error) {
	req := MoveRequest{Directions: directions}
	return c.executeMove(req)
}

type ResetResponse struct {
	Message string     `json:"message"`
	State   *GameState `json:"state"`
}

func (c *Client) Reset() (*GameState, error) {
	url := fmt.Sprintf("%s/api/sessions/%s/reset", c.baseURL, c.sessionID)
	resp, err := c.client.Post(url, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("reset: %w", err)
	}
	defer resp.Body.Close()

	var resetResp ResetResponse
	if err := json.NewDecoder(resp.Body).Decode(&resetResp); err != nil {
		return nil, fmt.Errorf("parse reset response: %w", err)
	}

	return resetResp.State, nil
}

type MoveResponse struct {
	Success   bool       `json:"success"`
	GameState *GameState `json:"game_state"`
	Message   string     `json:"message"`
}

func (c *Client) executeMove(req MoveRequest) (*GameState, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal move: %w", err)
	}

	url := fmt.Sprintf("%s/api/sessions/%s/move", c.baseURL, c.sessionID)
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("execute move: %w", err)
	}
	defer resp.Body.Close()

	var moveResp MoveResponse
	if err := json.NewDecoder(resp.Body).Decode(&moveResp); err != nil {
		return nil, fmt.Errorf("parse move response: %w", err)
	}

	// Check if move actually succeeded even if success=false
	// Some messages like "Home sweet home!" are informational, not errors
	// If we got a valid game state back and it's not game over, treat as success
	if !moveResp.Success {
		if moveResp.GameState != nil && !moveResp.GameState.GameOver {
			// Move partially succeeded - return state without error
			return moveResp.GameState, nil
		}
		return moveResp.GameState, fmt.Errorf("move failed: %s", moveResp.Message)
	}

	return moveResp.GameState, nil
}

func main() {
	serverURL := flag.String("url", "http://localhost:8080", "Game server URL")
	configName := flag.String("config", "", "Game configuration name (default, easy, medium_maze)")
	continueSession := flag.String("continue", "", "Resume playing an existing session by ID")
	maxMoves := flag.Int("max-moves", 3000, "Maximum moves per attempt")
	maxAttempts := flag.Int("max-attempts", 100, "Maximum attempts before giving up")
	verbose := flag.Bool("v", false, "Verbose output")
	delayMs := flag.Int("delay", 0, "Delay between moves in milliseconds (0 = no delay)")
	flag.Parse()

	log.Printf("Connecting to game server at %s", *serverURL)
	client := NewClient(*serverURL)

	var state *GameState
	var err error
	var totalParks int

	// Check for saved session ID
	sessionFile := ".session"
	savedSessionID := ""

	if *continueSession != "" {
		// Use explicitly provided session
		savedSessionID = *continueSession
	} else {
		// Try to load saved session
		if data, err := os.ReadFile(sessionFile); err == nil {
			savedSessionID = string(bytes.TrimSpace(data))
		}
	}

	if savedSessionID != "" {
		// Resume existing session
		client.sessionID = savedSessionID
		log.Printf("🔄 Resuming session: %s", client.sessionID)
		state, err = client.GetState()
		if err != nil {
			log.Printf("⚠️  Failed to resume session (may be expired): %v", err)
			log.Printf("Creating new session...")
			savedSessionID = "" // Force create new
		} else {
			totalParks = countTotalParks(state)
			log.Printf("Session resumed - Grid: %dx%d, Parks: %d, Battery: %d/%d",
				len(state.Grid[0]), len(state.Grid), totalParks, state.Battery, state.MaxBattery)
		}
	}

	if savedSessionID == "" {
		// Create new session
		state, err = client.CreateSession(*configName)
		if err != nil {
			log.Fatalf("Failed to create session: %v", err)
		}
		log.Printf("✨ Session created: %s", client.sessionID)
		totalParks = countTotalParks(state)
		log.Printf("Grid size: %dx%d, Parks to collect: %d, Battery: %d/%d",
			len(state.Grid[0]), len(state.Grid), totalParks, state.Battery, state.MaxBattery)

		// Save session ID for next run
		if err := os.WriteFile(sessionFile, []byte(client.sessionID), 0644); err != nil {
			log.Printf("Warning: Failed to save session ID: %v", err)
		}
	}

	// RESET the game state at the beginning of each run
	log.Printf("🔄 Resetting game state...")
	state, err = client.Reset()
	if err != nil {
		log.Fatalf("Failed to reset game: %v", err)
	}
	log.Printf("Game reset - Position: (%d,%d), Battery: %d/%d",
		state.PlayerPos.X, state.PlayerPos.Y, state.Battery, state.MaxBattery)

	// Initialize systematic strategy
	systematicStrategy := NewSystematicStrategy(state)

	// Keep trying until victory or max attempts
	attemptNum := 0
	for attemptNum < *maxAttempts {
		attemptNum++

		// Reset the game for this attempt
		if attemptNum > 1 {
			state, err = client.Reset()
			if err != nil {
				log.Printf("Failed to reset: %v", err)
				break
			}
		}

		// Reset strategy for new attempt
		systematicStrategy.Reset()

		log.Printf("\n=== 🎮 Attempt %d/%d ===", attemptNum, *maxAttempts)

		// Try to complete the game - use single moves for reliability
		moveCount := 0
		for !state.Victory && !state.GameOver && moveCount < *maxMoves {
			if *verbose && moveCount%50 == 0 {
				log.Printf("Position: (%d,%d), Battery: %d/%d, Parks: %d/%d",
					state.PlayerPos.X, state.PlayerPos.Y,
					state.Battery, state.MaxBattery,
					len(state.VisitedParks), totalParks)
			}

			// Get next move from strategy
			direction := systematicStrategy.NextMove(state)
			if direction == "" {
				log.Printf("⚠️  No valid moves available")
				break
			}

			// Execute single move
			newState, err := client.Move(direction)
			if err != nil {
				// Even on "error", check if we got a valid state back
				if newState != nil && !newState.GameOver {
					// Move succeeded despite error message (e.g., charging message)
					state = newState
					moveCount++
					continue
				}

				if *verbose {
					log.Printf("Move failed: %v", err)
				}
				if newState != nil {
					state = newState
				}
				// Try to continue
				continue
			}
			state = newState
			moveCount++

			// Add delay if specified
			if *delayMs > 0 {
				time.Sleep(time.Duration(*delayMs) * time.Millisecond)
			}
		}

		parksCollected := len(state.VisitedParks)
		log.Printf("Attempt %d: Moves=%d, Parks=%d/%d, Battery=%d/%d",
			attemptNum, moveCount, parksCollected, totalParks, state.Battery, state.MaxBattery)

		// Show progress
		if parksCollected > 0 {
			log.Printf("✅ Collected parks: %v", state.VisitedParks)
		}

		// Check if we won
		if state.Victory {
			log.Printf("\n🎉 VICTORY! Game won in attempt %d with %d moves!", attemptNum, moveCount)
			log.Printf("Session: %s", client.sessionID)
			os.Exit(0)
		}
	}

	// Failed to win after all attempts
	log.Printf("\n❌ Failed to win after %d attempts", attemptNum)
	log.Printf("Session: %s", client.sessionID)
	os.Exit(1)
}

func countTotalParks(state *GameState) int {
	count := 0
	for _, row := range state.Grid {
		for _, cell := range row {
			if cell.Type == "park" {
				count++
			}
		}
	}
	return count
}
