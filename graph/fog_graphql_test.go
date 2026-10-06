package graph

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/wricardo/tesla-road-trip-game/game/config"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/game/session"
	"github.com/wricardo/tesla-road-trip-game/graph/generated"
)

type gqlResp struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func newTestGraphQLServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg, err := config.NewManager("../maps")
	if err != nil {
		t.Fatalf("config.NewManager: %v", err)
	}
	sessions := session.NewManager()
	svc := service.NewGameService(sessions, cfg)
	resolver := NewResolver(svc, nil)
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	return httptest.NewServer(h)
}

func gqlPost(t *testing.T, url, query string, variables map[string]any) gqlResp {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		t.Fatalf("marshal gql body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gql request failed: %v", err)
	}
	defer resp.Body.Close()

	var out gqlResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode gql response: %v", err)
	}
	return out
}

// Every nearbyGrid cell must carry its own map coordinates, so a client never
// has to derive them from playerPos and fogRadius.
func TestGraphQL_NearbyGridCellsCarryCoordinates(t *testing.T) {
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	create := gqlPost(t, ts.URL, `mutation { createSession(mapID:"easy", fogEnabled:true, fogRadius:2, moveDelayMs:0) { id } }`, nil)
	if len(create.Errors) > 0 {
		t.Fatalf("createSession errors: %+v", create.Errors)
	}
	var created struct {
		CreateSession struct{ ID string } `json:"createSession"`
	}
	if err := json.Unmarshal(create.Data, &created); err != nil {
		t.Fatal(err)
	}

	moved := gqlPost(t, ts.URL,
		`mutation($sid: ID!) { bulkMove(sessionID:$sid, moves:[UP]) { gameState { playerPos { x y } nearbyGrid { x y } } } }`,
		map[string]any{"sid": created.CreateSession.ID},
	)
	if len(moved.Errors) > 0 {
		t.Fatalf("bulkMove errors: %+v", moved.Errors)
	}
	var out struct {
		BulkMove struct {
			GameState struct {
				PlayerPos  struct{ X, Y int }     `json:"playerPos"`
				NearbyGrid [][]struct{ X, Y int } `json:"nearbyGrid"`
			} `json:"gameState"`
		} `json:"bulkMove"`
	}
	if err := json.Unmarshal(moved.Data, &out); err != nil {
		t.Fatal(err)
	}
	gs := out.BulkMove.GameState
	if len(gs.NearbyGrid) != 5 {
		t.Fatalf("want 5 rows for radius 2, got %d", len(gs.NearbyGrid))
	}
	for j, row := range gs.NearbyGrid {
		for i, c := range row {
			if c.X != gs.PlayerPos.X-2+i || c.Y != gs.PlayerPos.Y-2+j {
				t.Fatalf("nearbyGrid[%d][%d] = (%d,%d), want (%d,%d)", j, i, c.X, c.Y, gs.PlayerPos.X-2+i, gs.PlayerPos.Y-2+j)
			}
		}
	}
}

func TestGraphQL_GridPasswordWhenFogEnabled(t *testing.T) {
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	create := gqlPost(t, ts.URL,
		`mutation($map: String!, $r: Int!, $pw: String!) {
			createSession(mapID:$map, fogEnabled:true, fogRadius:$r, gridPassword:$pw) { id }
		}`,
		map[string]any{"map": "easy", "r": 2, "pw": "letmein"},
	)
	if len(create.Errors) > 0 {
		t.Fatalf("createSession errors: %+v", create.Errors)
	}
	var createData struct {
		CreateSession struct {
			ID string `json:"id"`
		} `json:"createSession"`
	}
	if err := json.Unmarshal(create.Data, &createData); err != nil {
		t.Fatalf("unmarshal create data: %v", err)
	}
	if createData.CreateSession.ID == "" {
		t.Fatalf("expected session id")
	}
	sid := createData.CreateSession.ID

	withoutPassword := gqlPost(t, ts.URL,
		`query($sid: ID!) { gameState(sessionID:$sid) { grid { type } } }`,
		map[string]any{"sid": sid},
	)
	if len(withoutPassword.Errors) == 0 {
		t.Fatalf("expected error when querying grid without password")
	}

	wrongPassword := gqlPost(t, ts.URL,
		`query($sid: ID!, $pw: String!) { gameState(sessionID:$sid) { grid(password:$pw) { type } } }`,
		map[string]any{"sid": sid, "pw": "wrong"},
	)
	if len(wrongPassword.Errors) == 0 {
		t.Fatalf("expected error with wrong grid password")
	}

	layoutWithoutPassword := gqlPost(t, ts.URL,
		`query($sid: ID!) { session(id:$sid) { gameMap { layout } } }`,
		map[string]any{"sid": sid},
	)
	if len(layoutWithoutPassword.Errors) == 0 {
		t.Fatalf("expected error when querying session gameMap layout without password")
	}

	layoutWithPassword := gqlPost(t, ts.URL,
		`query($sid: ID!, $pw: String!) { session(id:$sid) { gameMap { layout(password:$pw) } } }`,
		map[string]any{"sid": sid, "pw": "letmein"},
	)
	if len(layoutWithPassword.Errors) > 0 {
		t.Fatalf("unexpected errors with correct layout password: %+v", layoutWithPassword.Errors)
	}

	goodPassword := gqlPost(t, ts.URL,
		`query($sid: ID!, $pw: String!) { gameState(sessionID:$sid) { grid(password:$pw) { type } nearbyGrid { type } } }`,
		map[string]any{"sid": sid, "pw": "letmein"},
	)
	if len(goodPassword.Errors) > 0 {
		t.Fatalf("unexpected errors with correct password: %+v", goodPassword.Errors)
	}
	var okData struct {
		GameState struct {
			Grid [][]struct {
				Type string `json:"type"`
			} `json:"grid"`
			NearbyGrid [][]struct {
				Type string `json:"type"`
			} `json:"nearbyGrid"`
		} `json:"gameState"`
	}
	if err := json.Unmarshal(goodPassword.Data, &okData); err != nil {
		t.Fatalf("unmarshal success data: %v", err)
	}
	if len(okData.GameState.Grid) == 0 || len(okData.GameState.Grid[0]) == 0 {
		t.Fatalf("expected full grid data")
	}
	if got := len(okData.GameState.NearbyGrid); got != 5 {
		t.Fatalf("expected nearbyGrid height 5 for radius=2, got %d", got)
	}
	if got := len(okData.GameState.NearbyGrid[0]); got != 5 {
		t.Fatalf("expected nearbyGrid width 5 for radius=2, got %d", got)
	}
}

func TestGraphQL_CreateSessionFogValidation(t *testing.T) {
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	badRadius := gqlPost(t, ts.URL,
		`mutation { createSession(mapID:"easy", fogEnabled:true, fogRadius:0, gridPassword:"pw") { id } }`,
		nil,
	)
	if len(badRadius.Errors) == 0 {
		t.Fatalf("expected createSession to fail when fog radius is invalid")
	}
}

func TestGraphQL_MapQueryPassword(t *testing.T) {
	pw := loadUIMapPassword()
	if pw == "" {
		t.Fatalf("expected non-empty uiMapPassword in frontend/src/lib/config/ui-auth.json")
	}
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	without := gqlPost(t, ts.URL,
		`query { map(name:"easy") { name layout } }`,
		nil,
	)
	if len(without.Errors) == 0 {
		t.Fatalf("expected map query to fail without UI map password")
	}

	wrong := gqlPost(t, ts.URL,
		`query($pw: String!) { map(name:"easy", password:$pw) { name layout } }`,
		map[string]any{"pw": "wrong"},
	)
	if len(wrong.Errors) == 0 {
		t.Fatalf("expected map query to fail with wrong UI map password")
	}

	good := gqlPost(t, ts.URL,
		`query($pw: String!) { map(name:"easy", password:$pw) { name layout } }`,
		map[string]any{"pw": pw},
	)
	if len(good.Errors) > 0 {
		t.Fatalf("unexpected map query errors with correct password: %+v", good.Errors)
	}
}

func TestGraphQL_GridNoPasswordWhenFogDisabled(t *testing.T) {
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	create := gqlPost(t, ts.URL,
		`mutation { createSession(mapID:"easy", fogEnabled:false) { id } }`,
		nil,
	)
	if len(create.Errors) > 0 {
		t.Fatalf("createSession errors: %+v", create.Errors)
	}
	var createData struct {
		CreateSession struct {
			ID string `json:"id"`
		} `json:"createSession"`
	}
	if err := json.Unmarshal(create.Data, &createData); err != nil {
		t.Fatalf("unmarshal create data: %v", err)
	}

	query := gqlPost(t, ts.URL,
		`query($sid: ID!) { gameState(sessionID:$sid) { grid { type } nearbyGrid { type } } session(id:$sid) { gameMap { layout } } }`,
		map[string]any{"sid": createData.CreateSession.ID},
	)
	if len(query.Errors) > 0 {
		t.Fatalf("unexpected errors for fog-disabled grid: %+v", query.Errors)
	}
	var data struct {
		GameState struct {
			NearbyGrid [][]struct {
				Type string `json:"type"`
			} `json:"nearbyGrid"`
		} `json:"gameState"`
	}
	if err := json.Unmarshal(query.Data, &data); err != nil {
		t.Fatalf("unmarshal query data: %v", err)
	}
	if got := len(data.GameState.NearbyGrid); got != 3 {
		t.Fatalf("expected default nearbyGrid height 3, got %d", got)
	}
}

func TestGraphQL_FogWithoutPasswordGeneratesOne(t *testing.T) {
	ts := newTestGraphQLServer(t)
	defer ts.Close()

	create := gqlPost(t, ts.URL,
		`mutation { createSession(mapID:"easy", fogEnabled:true, fogRadius:2) { id generatedGridPassword } }`,
		nil,
	)
	if len(create.Errors) > 0 {
		t.Fatalf("createSession errors: %+v", create.Errors)
	}
	var data struct {
		CreateSession struct {
			ID                    string  `json:"id"`
			GeneratedGridPassword *string `json:"generatedGridPassword"`
		} `json:"createSession"`
	}
	if err := json.Unmarshal(create.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.CreateSession.GeneratedGridPassword == nil || *data.CreateSession.GeneratedGridPassword == "" {
		t.Fatalf("expected a generated password in the create response")
	}
	sid, pw := data.CreateSession.ID, *data.CreateSession.GeneratedGridPassword

	unlocked := gqlPost(t, ts.URL,
		`query($sid: ID!, $pw: String!) { gameState(sessionID:$sid) { grid(password:$pw) { type } } }`,
		map[string]any{"sid": sid, "pw": pw},
	)
	if len(unlocked.Errors) > 0 {
		t.Fatalf("generated password should unlock the grid: %+v", unlocked.Errors)
	}

	later := gqlPost(t, ts.URL,
		`query($sid: ID!) { session(id:$sid) { generatedGridPassword } }`,
		map[string]any{"sid": sid},
	)
	if len(later.Errors) > 0 || string(later.Data) != `{"session":{"generatedGridPassword":null}}` {
		t.Fatalf("password must not be readable after creation: data=%s errors=%+v", later.Data, later.Errors)
	}
}
