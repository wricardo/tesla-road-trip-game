package graph

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/graph/model"
)

func timeString(t time.Time) string { return t.Format(time.RFC3339) }

func toPosition(p engine.Position) *model.Position { return &model.Position{X: p.X, Y: p.Y} }

func toSession(s *service.SessionInfo) *model.Session {
	if s == nil {
		return nil
	}
	var dn *string
	if s.DisplayName != "" {
		dn = &s.DisplayName
	}
	gs := toGameState(s.GameState)
	gm := toGameMap(s.GameMap)
	if gm != nil && s.GameState != nil {
		gameMapLayoutPolicy.Store(gm, gridPolicy{fogEnabled: s.GameState.FogEnabled, gridPassword: s.GameState.GridPassword})
	}
	return &model.Session{ID: s.ID, DisplayName: dn, MapName: s.MapName, CreatedAt: timeString(s.CreatedAt), LastAccessedAt: timeString(s.LastAccessedAt), GameState: gs, GameMap: gm}
}

func toUnifiedSession(s *service.SessionInfo) *model.UnifiedSession {
	if s == nil {
		return nil
	}
	gs := toGameState(s.GameState)
	gm := toGameMap(s.GameMap)
	if gm != nil && s.GameState != nil {
		gameMapLayoutPolicy.Store(gm, gridPolicy{fogEnabled: s.GameState.FogEnabled, gridPassword: s.GameState.GridPassword})
	}
	return &model.UnifiedSession{SessionID: s.ID, CreatedAt: timeString(s.CreatedAt), LastAccessedAt: timeString(s.LastAccessedAt), GameState: gs, GameMap: gm}
}

type gridPolicy struct {
	fogEnabled   bool
	gridPassword string
}

var gameStateGridPolicy sync.Map // map[*model.GameState]gridPolicy
var gameMapLayoutPolicy sync.Map // map[*model.GameMap]gridPolicy

func toGameState(gs *engine.GameState) *model.GameState {
	if gs == nil {
		return nil
	}
	grid := make([][]*model.Cell, len(gs.Grid))
	for y := range gs.Grid {
		grid[y] = make([]*model.Cell, len(gs.Grid[y]))
		for x, c := range gs.Grid[y] {
			dirs := make([]string, len(c.AllowedDirections))
			copy(dirs, c.AllowedDirections)
			grid[y][x] = &model.Cell{Type: string(c.Type), Visited: c.Visited, ID: c.ID, AllowedDirections: dirs}
		}
	}
	visitedKeys := make([]string, 0, len(gs.VisitedParks))
	for k := range gs.VisitedParks {
		visitedKeys = append(visitedKeys, k)
	}
	sort.Strings(visitedKeys)
	visited := make([]*model.VisitedPark, 0, len(visitedKeys))
	for _, k := range visitedKeys {
		visited = append(visited, &model.VisitedPark{ID: k, Visited: gs.VisitedParks[k]})
	}
	local := buildLocalViewGrid(gs)
	out := &model.GameState{Grid: grid, PlayerPos: toPosition(gs.PlayerPos), Battery: gs.Battery, MaxBattery: gs.MaxBattery, Score: gs.Score, VisitedParks: visited, Message: gs.Message, GameOver: gs.GameOver, Victory: gs.Victory, MapName: gs.MapName, MoveHistory: toMoveHistory(gs.MoveHistory), TotalMoves: gs.TotalMoves, NearbyGrid: local, CurrentMoves: toMoveHistory(gs.CurrentMoves), CurrentMovesCount: gs.CurrentMovesCount, BatteryRisk: gs.BatteryRisk, FogEnabled: gs.FogEnabled, FogRadius: gs.FogRadius, MoveDelayMs: gs.MoveDelayMs}
	gameStateGridPolicy.Store(out, gridPolicy{fogEnabled: gs.FogEnabled, gridPassword: gs.GridPassword})
	return out
}

func buildLocalViewGrid(gs *engine.GameState) [][]*model.Cell {
	if gs == nil {
		return nil
	}
	radius := 1
	if gs.FogEnabled && gs.FogRadius > 0 {
		radius = gs.FogRadius
	}
	size := radius*2 + 1
	local := make([][]*model.Cell, 0, size)
	px, py := gs.PlayerPos.X, gs.PlayerPos.Y
	for dy := -radius; dy <= radius; dy++ {
		row := make([]*model.Cell, 0, size)
		for dx := -radius; dx <= radius; dx++ {
			x, y := px+dx, py+dy
			if y < 0 || y >= len(gs.Grid) || x < 0 || x >= len(gs.Grid[y]) {
				row = append(row, &model.Cell{Type: string(engine.Building), Visited: false, ID: "", AllowedDirections: nil})
				continue
			}
			c := gs.Grid[y][x]
			dirs := make([]string, len(c.AllowedDirections))
			copy(dirs, c.AllowedDirections)
			row = append(row, &model.Cell{Type: string(c.Type), Visited: c.Visited, ID: c.ID, AllowedDirections: dirs})
		}
		local = append(local, row)
	}
	return local
}

func toMoveHistory(entries []engine.MoveHistoryEntry) []*model.MoveHistoryEntry {
	out := make([]*model.MoveHistoryEntry, len(entries))
	for i, e := range entries {
		out[i] = &model.MoveHistoryEntry{Action: e.Action, FromPosition: toPosition(e.FromPosition), ToPosition: toPosition(e.ToPosition), Battery: e.Battery, Timestamp: int(e.Timestamp), Success: e.Success, MoveNumber: e.MoveNumber}
	}
	return out
}

func toGameMap(c *engine.GameConfig) *model.GameMap {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, len(c.Legend))
	for k := range c.Legend {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	legend := make([]*model.LegendEntry, 0, len(keys))
	for _, k := range keys {
		legend = append(legend, &model.LegendEntry{Key: k, Value: c.Legend[k]})
	}
	cellConfigKeys := make([]string, 0, len(c.CellConfigs))
	for k := range c.CellConfigs {
		cellConfigKeys = append(cellConfigKeys, k)
	}
	sort.Strings(cellConfigKeys)
	cellConfigs := make([]*model.CellConfigEntry, 0, len(cellConfigKeys))
	for _, k := range cellConfigKeys {
		cc := c.CellConfigs[k]
		dirs := make([]string, len(cc.AllowedDirections))
		copy(dirs, cc.AllowedDirections)
		cellConfigs = append(cellConfigs, &model.CellConfigEntry{Key: k, Type: cc.Type, AllowedDirections: dirs})
	}
	return &model.GameMap{Name: c.Name, Description: c.Description, GridSize: c.GridSize, MaxBattery: c.MaxBattery, StartingBattery: c.StartingBattery, Layout: c.Layout, Legend: legend, CellConfigs: cellConfigs, WallCrashEndsGame: c.WallCrashEndsGame}
}

func fromGameMapInput(in model.GameMapInput) *engine.GameConfig {
	legend := make(map[string]string, len(in.Legend))
	for _, e := range in.Legend {
		if e != nil {
			legend[e.Key] = e.Value
		}
	}
	cellConfigs := make(map[string]engine.CellConfig, len(in.CellConfigs))
	for _, cc := range in.CellConfigs {
		if cc != nil {
			dirs := make([]string, len(cc.AllowedDirections))
			copy(dirs, cc.AllowedDirections)
			cellConfigs[cc.Key] = engine.CellConfig{Type: cc.Type, AllowedDirections: dirs}
		}
	}
	return &engine.GameConfig{Name: in.Name, Description: in.Description, GridSize: in.GridSize, MaxBattery: in.MaxBattery, StartingBattery: in.StartingBattery, Layout: in.Layout, Legend: legend, CellConfigs: cellConfigs, WallCrashEndsGame: in.WallCrashEndsGame}
}

// applyPatch merges non-nil patch fields onto an existing GameConfig (in place).
func applyPatch(cfg *engine.GameConfig, patch model.GameMapPatchInput) {
	if patch.Name != nil {
		cfg.Name = *patch.Name
	}
	if patch.Description != nil {
		cfg.Description = *patch.Description
	}
	if patch.GridSize != nil {
		cfg.GridSize = *patch.GridSize
	}
	if patch.MaxBattery != nil {
		cfg.MaxBattery = *patch.MaxBattery
	}
	if patch.StartingBattery != nil {
		cfg.StartingBattery = *patch.StartingBattery
	}
	if patch.Layout != nil {
		cfg.Layout = patch.Layout
	}
	if patch.Legend != nil {
		legend := make(map[string]string, len(patch.Legend))
		for _, e := range patch.Legend {
			if e != nil {
				legend[e.Key] = e.Value
			}
		}
		cfg.Legend = legend
	}
	if patch.CellConfigs != nil {
		cellConfigs := make(map[string]engine.CellConfig, len(patch.CellConfigs))
		for _, cc := range patch.CellConfigs {
			if cc != nil {
				dirs := make([]string, len(cc.AllowedDirections))
				copy(dirs, cc.AllowedDirections)
				cellConfigs[cc.Key] = engine.CellConfig{Type: cc.Type, AllowedDirections: dirs}
			}
		}
		cfg.CellConfigs = cellConfigs
	}
	if patch.WallCrashEndsGame != nil {
		cfg.WallCrashEndsGame = *patch.WallCrashEndsGame
	}
}

func toMapInfo(c *service.MapInfo) *model.MapInfo {
	if c == nil {
		return nil
	}
	return &model.MapInfo{Filename: c.Filename, MapID: c.MapID, Name: c.Name, Description: c.Description, GridSize: c.GridSize, MaxBattery: c.MaxBattery}
}
func toAttempt(a *service.AttemptInfo) *model.AttemptInfo {
	if a == nil {
		return nil
	}
	return &model.AttemptInfo{X: a.X, Y: a.Y, TileChar: a.TileChar, TileType: a.TileType, Passable: a.Passable}
}
func toStep(s service.StepInfo) *model.StepInfo {
	return &model.StepInfo{Idx: s.Idx, Dir: s.Dir, From: toPosition(s.From), To: toPosition(s.To), TileChar: s.TileChar, TileType: s.TileType, BatteryBefore: s.BatteryBefore, BatteryAfter: s.BatteryAfter, Success: s.Success, Charged: s.Charged, Park: s.Park, Victory: s.Victory}
}
func toStepPtr(s *service.StepInfo) *model.StepInfo {
	if s == nil {
		return nil
	}
	return toStep(*s)
}
func toEvents(events []service.GameEvent) []*model.GameEvent {
	out := make([]*model.GameEvent, len(events))
	for i, e := range events {
		out[i] = &model.GameEvent{Type: e.Type, Message: e.Message, Timestamp: timeString(e.Timestamp), Position: toPosition(e.Position)}
	}
	return out
}
func toMoveResult(r *service.MoveResult) *model.MoveResult {
	if r == nil {
		return nil
	}
	return &model.MoveResult{Success: r.Success, GameState: toGameState(r.GameState), Message: r.Message, Events: toEvents(r.Events), Step: toStepPtr(r.Step), AttemptedTo: toAttempt(r.AttemptedTo)}
}
func toBulkResult(r *service.BulkMoveResult) *model.BulkMoveResult {
	if r == nil {
		return nil
	}
	steps := make([]*model.StepInfo, len(r.Steps))
	for i, s := range r.Steps {
		steps[i] = toStep(s)
	}
	return &model.BulkMoveResult{MovesExecuted: r.MovesExecuted, TotalMoves: r.TotalMoves, RequestedMoves: r.RequestedMoves, Success: r.Success, GameState: toGameState(r.GameState), Events: toEvents(r.Events), StoppedReason: r.StoppedReason, StopReasonCode: r.StopReasonCode, StoppedOnMove: r.StoppedOnMove, Truncated: r.Truncated, Limit: r.Limit, StartPos: toPosition(r.StartPos), EndPos: toPosition(r.EndPos), StartBattery: r.StartBattery, EndBattery: r.EndBattery, ScoreDelta: r.ScoreDelta, Steps: steps, AttemptedTo: toAttempt(r.AttemptedTo), GameOver: r.GameOver, GameOverCode: r.GameOverCode, Message: r.Message, PossibleMoves: r.PossibleMoves, BatteryRisk: r.BatteryRisk}
}
func toHistory(h *service.HistoryResponse) *model.HistoryResponse {
	if h == nil {
		return nil
	}
	return &model.HistoryResponse{Moves: toMoveHistory(h.Moves), TotalMoves: h.TotalMoves, Page: h.Page, PageSize: h.PageSize, TotalPages: h.TotalPages, HasNext: h.HasNext, HasPrevious: h.HasPrevious}
}
func directionString(d model.Direction) string { return strings.ToLower(string(d)) }
