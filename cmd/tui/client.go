package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(server string) *Client {
	return &Client{
		endpoint:   server + "/graphql",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) do(query string, vars map[string]any, result any) error {
	body, err := json.Marshal(gqlRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer resp.Body.Close()

	var gqlResp gqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return err
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("%s", gqlResp.Errors[0].Message)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(gqlResp.Data, result)
}

// --- Domain types ---

type MapInfo struct {
	MapId       string `json:"mapId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GridSize    int    `json:"gridSize"`
	MaxBattery  int    `json:"maxBattery"`
}

type Session struct {
	ID             string    `json:"id"`
	DisplayName    *string   `json:"displayName"`
	MapName        string    `json:"mapName"`
	CreatedAt      string    `json:"createdAt"`
	LastAccessedAt string    `json:"lastAccessedAt"`
	GameState      GameState `json:"gameState"`
}

type GameState struct {
	Battery     int      `json:"battery"`
	MaxBattery  int      `json:"maxBattery"`
	Score       int      `json:"score"`
	Victory     bool     `json:"victory"`
	GameOver    bool     `json:"gameOver"`
	TotalMoves  int      `json:"totalMoves"`
	Message     string   `json:"message"`
	MapName     string   `json:"mapName"`
	BatteryRisk string   `json:"batteryRisk"`
	PlayerPos   Position `json:"playerPos"`
	Grid        [][]Cell `json:"grid"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Cell struct {
	Type              string   `json:"type"`
	Visited           bool     `json:"visited"`
	ID                string   `json:"id"`
	AllowedDirections []string `json:"allowedDirections"`
}

// --- API methods ---

func (c *Client) Sessions() ([]Session, error) {
	var result struct {
		Sessions struct {
			Sessions []Session `json:"sessions"`
		} `json:"sessions"`
	}
	err := c.do(`
		query {
			sessions(limit: 50) {
				sessions {
					id displayName mapName createdAt lastAccessedAt
					gameState { battery maxBattery score victory gameOver totalMoves message }
				}
			}
		}
	`, nil, &result)
	return result.Sessions.Sessions, err
}

func (c *Client) Maps() ([]MapInfo, error) {
	var result struct {
		Maps []MapInfo `json:"maps"`
	}
	err := c.do(`query { maps { mapId name description gridSize maxBattery } }`, nil, &result)
	return result.Maps, err
}

func (c *Client) CreateSession(mapID string) (string, error) {
	var result struct {
		CreateSession struct {
			ID string `json:"id"`
		} `json:"createSession"`
	}
	err := c.do(`
		mutation CreateSession($mapID: String) {
			createSession(mapID: $mapID) { id }
		}
	`, map[string]any{"mapID": mapID}, &result)
	return result.CreateSession.ID, err
}

func (c *Client) UpdateSession(id, displayName string) error {
	return c.do(`
		mutation UpdateSession($id: ID!, $displayName: String!) {
			updateSession(id: $id, displayName: $displayName) { id }
		}
	`, map[string]any{"id": id, "displayName": displayName}, nil)
}

func (c *Client) GameState(sessionID string) (*GameState, error) {
	var result struct {
		GameState GameState `json:"gameState"`
	}
	err := c.do(`
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName batteryRisk
				playerPos { x y }
				grid { type visited id allowedDirections }
			}
		}
	`, map[string]any{"sessionID": sessionID}, &result)
	return &result.GameState, err
}

func (c *Client) Move(sessionID, direction string) (*GameState, error) {
	var result struct {
		Move struct {
			GameState GameState `json:"gameState"`
		} `json:"move"`
	}
	err := c.do(`
		mutation Move($sessionID: ID!, $direction: Direction!) {
			move(sessionID: $sessionID, direction: $direction) {
				gameState {
					battery maxBattery score victory gameOver totalMoves message mapName batteryRisk
					playerPos { x y }
					grid { type visited id allowedDirections }
				}
			}
		}
	`, map[string]any{"sessionID": sessionID, "direction": direction}, &result)
	return &result.Move.GameState, err
}

func (c *Client) Reset(sessionID string) (*GameState, error) {
	var result struct {
		Reset GameState `json:"reset"`
	}
	err := c.do(`
		mutation Reset($sessionID: ID!) {
			reset(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName batteryRisk
				playerPos { x y }
				grid { type visited id allowedDirections }
			}
		}
	`, map[string]any{"sessionID": sessionID}, &result)
	return &result.Reset, err
}

func (c *Client) DeleteSession(id string) error {
	return c.do(`
		mutation DeleteSession($id: ID!) {
			deleteSession(id: $id) { message }
		}
	`, map[string]any{"id": id}, nil)
}

type MoveHistoryEntry struct {
	FromPosition Position `json:"fromPosition"`
	ToPosition   Position `json:"toPosition"`
	Success      bool     `json:"success"`
}

// AllHistory fetches all move history for a session across pages.
func (c *Client) AllHistory(sessionID string) ([]MoveHistoryEntry, error) {
	var all []MoveHistoryEntry
	for page := 1; ; page++ {
		var result struct {
			History struct {
				Moves   []MoveHistoryEntry `json:"moves"`
				HasNext bool               `json:"hasNext"`
			} `json:"history"`
		}
		err := c.do(`
			query History($sessionID: ID!, $page: Int!) {
				history(sessionID: $sessionID, page: $page, limit: 200, order: ASC) {
					moves { fromPosition { x y } toPosition { x y } success }
					hasNext
				}
			}
		`, map[string]any{"sessionID": sessionID, "page": page}, &result)
		if err != nil {
			return all, err
		}
		all = append(all, result.History.Moves...)
		if !result.History.HasNext {
			break
		}
	}
	return all, nil
}
