// Command envcheck replays action traces against the real game engine so the
// Python training environment can be diffed against it.
//
//	go run ./cmd/envcheck < traces.json
//
// Input:  [{"map_file":"maps/easy.json","actions":["up","right",...]}]
// Output: [{"states":[{x,y,battery,score,game_over,victory,visited:[...]},...]}]
// The first state is the initial one; one state follows each action.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

type trace struct {
	MapFile string   `json:"map_file"`
	Actions []string `json:"actions"`
}

type snapshot struct {
	X        int      `json:"x"`
	Y        int      `json:"y"`
	Battery  int      `json:"battery"`
	Score    int      `json:"score"`
	GameOver bool     `json:"game_over"`
	Victory  bool     `json:"victory"`
	Visited  []string `json:"visited"`
}

type result struct {
	Error  string     `json:"error,omitempty"`
	States []snapshot `json:"states,omitempty"`
}

func snapshotOf(state *engine.GameState) snapshot {
	visited := make([]string, 0, len(state.VisitedParks))
	for id := range state.VisitedParks {
		visited = append(visited, id)
	}
	sort.Strings(visited)
	return snapshot{
		X:        state.PlayerPos.X,
		Y:        state.PlayerPos.Y,
		Battery:  state.Battery,
		Score:    state.Score,
		GameOver: state.GameOver,
		Victory:  state.Victory,
		Visited:  visited,
	}
}

func run(t trace) result {
	raw, err := os.ReadFile(t.MapFile)
	if err != nil {
		return result{Error: err.Error()}
	}
	var cfg engine.GameConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return result{Error: fmt.Sprintf("decode %s: %v", t.MapFile, err)}
	}
	eng, err := engine.NewEngineForLoad(&cfg)
	if err != nil {
		return result{Error: err.Error()}
	}
	out := result{States: []snapshot{snapshotOf(eng.GetState())}}
	for _, a := range t.Actions {
		if eng.IsGameOver() {
			break
		}
		eng.Move(a)
		out.States = append(out.States, snapshotOf(eng.GetState()))
	}
	return out
}

func main() {
	var traces []trace
	if err := json.NewDecoder(os.Stdin).Decode(&traces); err != nil {
		fmt.Fprintln(os.Stderr, "decode traces:", err)
		os.Exit(1)
	}
	results := make([]result, len(traces))
	for i, t := range traces {
		results[i] = run(t)
	}
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, "encode results:", err)
		os.Exit(1)
	}
}
