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
	"strings"
	"time"
)

// Internal game state (matches systematic_strategy.go expectations)
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
	PlayerPos    Position        `json:"playerPos"`
	Battery      int             `json:"battery"`
	MaxBattery   int             `json:"maxBattery"`
	Score        int             `json:"score"`
	GameOver     bool            `json:"gameOver"`
	Victory      bool            `json:"victory"`
	Message      string          `json:"message"`
	VisitedParks map[string]bool `json:"visitedParks"`
	MapName      string          `json:"mapName"`
}

// GraphQL response types

type gqlCell struct {
	Type    string `json:"type"`
	Visited bool   `json:"visited"`
	ID      string `json:"id"`
}

type gqlPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type gqlVisitedPark struct {
	ID      string `json:"id"`
	Visited bool   `json:"visited"`
}

type gqlGameState struct {
	Grid         [][]gqlCell      `json:"grid"`
	PlayerPos    gqlPosition      `json:"playerPos"`
	Battery      int              `json:"battery"`
	MaxBattery   int              `json:"maxBattery"`
	Score        int              `json:"score"`
	GameOver     bool             `json:"gameOver"`
	Victory      bool             `json:"victory"`
	Message      string           `json:"message"`
	VisitedParks []gqlVisitedPark `json:"visitedParks"`
	MapName      string           `json:"mapName"`
}

func toGameState(g gqlGameState) *GameState {
	grid := make([][]Cell, len(g.Grid))
	for y, row := range g.Grid {
		grid[y] = make([]Cell, len(row))
		for x, c := range row {
			grid[y][x] = Cell{Type: c.Type, Visited: c.Visited, ID: c.ID}
		}
	}
	visited := make(map[string]bool)
	for _, p := range g.VisitedParks {
		if p.Visited {
			visited[p.ID] = true
		}
	}
	return &GameState{
		Grid:         grid,
		PlayerPos:    Position{X: g.PlayerPos.X, Y: g.PlayerPos.Y},
		Battery:      g.Battery,
		MaxBattery:   g.MaxBattery,
		Score:        g.Score,
		GameOver:     g.GameOver,
		Victory:      g.Victory,
		Message:      g.Message,
		VisitedParks: visited,
		MapName:      g.MapName,
	}
}

// GraphQL client

type Client struct {
	endpoint  string
	sessionID string
	http      *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		endpoint: strings.TrimRight(baseURL, "/") + "/graphql",
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage  `json:"data"`
	Errors []map[string]any `json:"errors,omitempty"`
}

func (c *Client) do(query string, vars map[string]any) (json.RawMessage, error) {
	body, _ := json.Marshal(gqlRequest{Query: query, Variables: vars})
	resp, err := c.http.Post(c.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var gr gqlResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %.200s)", err, raw)
	}
	if len(gr.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %v", gr.Errors[0]["message"])
	}
	return gr.Data, nil
}

const gameStateFields = `
  grid { type visited id }
  playerPos { x y }
  battery maxBattery score gameOver victory message mapName
  visitedParks { id visited }
`

func (c *Client) CreateSession(mapName string) (*GameState, error) {
	q := `mutation CreateSession($mapName: String) {
		createSession(mapName: $mapName) {
			id
			gameState {` + gameStateFields + `}
		}
	}`
	data, err := c.do(q, map[string]any{"mapName": mapName})
	if err != nil {
		return nil, err
	}
	var r struct {
		CreateSession struct {
			ID        string       `json:"id"`
			GameState gqlGameState `json:"gameState"`
		} `json:"createSession"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse createSession: %w", err)
	}
	c.sessionID = r.CreateSession.ID
	return toGameState(r.CreateSession.GameState), nil
}

func (c *Client) GetState() (*GameState, error) {
	q := `query GetState($id: ID!) {
		session(id: $id) {
			gameState {` + gameStateFields + `}
		}
	}`
	data, err := c.do(q, map[string]any{"id": c.sessionID})
	if err != nil {
		return nil, err
	}
	var r struct {
		Session struct {
			GameState gqlGameState `json:"gameState"`
		} `json:"session"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return toGameState(r.Session.GameState), nil
}

func (c *Client) Move(direction string) (*GameState, error) {
	q := `mutation Move($sessionID: ID!, $direction: Direction!) {
		move(sessionID: $sessionID, direction: $direction) {
			success message
			gameState {` + gameStateFields + `}
		}
	}`
	data, err := c.do(q, map[string]any{
		"sessionID": c.sessionID,
		"direction": strings.ToUpper(direction),
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Move struct {
			Success   bool         `json:"success"`
			Message   string       `json:"message"`
			GameState gqlGameState `json:"gameState"`
		} `json:"move"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse move: %w", err)
	}
	gs := toGameState(r.Move.GameState)
	if !r.Move.Success && gs.GameOver {
		return gs, fmt.Errorf("move failed: %s", r.Move.Message)
	}
	return gs, nil
}

func (c *Client) Reset() (*GameState, error) {
	q := `mutation Reset($sessionID: ID!) {
		reset(sessionID: $sessionID) {` + gameStateFields + `}
	}`
	data, err := c.do(q, map[string]any{"sessionID": c.sessionID})
	if err != nil {
		return nil, err
	}
	var r struct {
		Reset gqlGameState `json:"reset"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse reset: %w", err)
	}
	return toGameState(r.Reset), nil
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

func main() {
	serverURL := flag.String("url", "http://localhost:8080", "Game server URL")
	configName := flag.String("config", "easy", "Map name")
	continueSession := flag.String("continue", "", "Resume existing session ID")
	maxMoves := flag.Int("max-moves", 3000, "Max moves per attempt")
	maxAttempts := flag.Int("max-attempts", 100, "Max attempts")
	verbose := flag.Bool("v", false, "Verbose output")
	delayMs := flag.Int("delay", 0, "Delay between moves (ms)")
	flag.Parse()

	log.Printf("Connecting to %s/graphql", *serverURL)
	client := NewClient(*serverURL)

	const sessionFile = ".session"
	savedSessionID := *continueSession
	if savedSessionID == "" {
		if data, err := os.ReadFile(sessionFile); err == nil {
			savedSessionID = strings.TrimSpace(string(data))
		}
	}

	var state *GameState
	var err error

	if savedSessionID != "" {
		client.sessionID = savedSessionID
		log.Printf("Resuming session: %s", client.sessionID)
		state, err = client.GetState()
		if err != nil {
			log.Printf("Resume failed: %v — creating new session", err)
			savedSessionID = ""
		}
	}

	if savedSessionID == "" {
		state, err = client.CreateSession(*configName)
		if err != nil {
			log.Fatalf("Create session failed: %v", err)
		}
		log.Printf("Session created: %s", client.sessionID)
		_ = os.WriteFile(sessionFile, []byte(client.sessionID), 0644)
	}

	totalParks := countTotalParks(state)
	log.Printf("Grid: %dx%d  Parks: %d  Battery: %d/%d",
		len(state.Grid[0]), len(state.Grid), totalParks, state.Battery, state.MaxBattery)

	// Reset before starting
	state, err = client.Reset()
	if err != nil {
		log.Fatalf("Reset failed: %v", err)
	}

	strategy := NewSystematicStrategy(state)

	for attempt := 1; attempt <= *maxAttempts; attempt++ {
		if attempt > 1 {
			state, err = client.Reset()
			if err != nil {
				log.Printf("Reset failed: %v", err)
				break
			}
			strategy.Reset()
		}

		log.Printf("\n=== Attempt %d/%d ===", attempt, *maxAttempts)

		for moveCount := 0; !state.Victory && !state.GameOver && moveCount < *maxMoves; moveCount++ {
			if *verbose && moveCount%50 == 0 {
				log.Printf("  pos=(%d,%d) battery=%d/%d parks=%d/%d",
					state.PlayerPos.X, state.PlayerPos.Y,
					state.Battery, state.MaxBattery,
					len(state.VisitedParks), totalParks)
			}

			dir := strategy.NextMove(state)
			if dir == "" {
				log.Printf("No valid move available")
				break
			}

			newState, err := client.Move(dir)
			if err != nil {
				if newState != nil && !newState.GameOver {
					state = newState
					continue
				}
				if newState != nil {
					state = newState
				}
				continue
			}
			state = newState

			if *delayMs > 0 {
				time.Sleep(time.Duration(*delayMs) * time.Millisecond)
			}
		}

		log.Printf("Attempt %d: parks=%d/%d battery=%d/%d",
			attempt, len(state.VisitedParks), totalParks, state.Battery, state.MaxBattery)

		if state.Victory {
			log.Printf("\nVICTORY! Won on attempt %d. Session: %s", attempt, client.sessionID)
			os.Exit(0)
		}
	}

	log.Printf("\nFailed after %d attempts. Session: %s", *maxAttempts, client.sessionID)
	os.Exit(1)
}
