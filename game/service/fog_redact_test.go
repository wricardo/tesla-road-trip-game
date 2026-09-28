package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

// Fog sessions must not leak the map through transports that serialize
// service structs (MCP, REST): no grid, no layout, never the password.
func TestRedactFogJSON_HidesMapForFogSessions(t *testing.T) {
	layout := []string{"BBB", "BHP", "BBB"}
	fogState := &engine.GameState{FogEnabled: true, GridPassword: "s3cret", Grid: [][]engine.Cell{{{Type: engine.Road}}}}
	cases := map[string]any{
		"session with state": &SessionInfo{ID: "a", FogEnabled: true, GameState: fogState, GameMap: &engine.GameConfig{Layout: layout}},
		"session list entry": []*SessionInfo{{ID: "b", FogEnabled: true, GameMap: &engine.GameConfig{Layout: layout}}},
		"bare state":         fogState,
		"unified entry":      map[string]any{"game_state": fogState, "game_map": &engine.GameConfig{Layout: layout}},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(payload)
			var data any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			RedactFogJSON(data)
			out, _ := json.Marshal(data)
			for _, leak := range []string{"s3cret", `"grid"`, `"layout"`} {
				if strings.Contains(string(out), leak) {
					t.Fatalf("fog payload leaked %s: %s", leak, out)
				}
			}
		})
	}
}

func TestRedactFogJSON_KeepsMapWithoutFog(t *testing.T) {
	info := &SessionInfo{ID: "a", GameState: &engine.GameState{Grid: [][]engine.Cell{{{Type: engine.Road}}}}, GameMap: &engine.GameConfig{Layout: []string{"R"}}}
	raw, _ := json.Marshal(info)
	var data any
	_ = json.Unmarshal(raw, &data)
	RedactFogJSON(data)
	out, _ := json.Marshal(data)
	if !strings.Contains(string(out), `"grid"`) || !strings.Contains(string(out), `"layout"`) {
		t.Fatalf("non-fog payload lost map data: %s", out)
	}
}
