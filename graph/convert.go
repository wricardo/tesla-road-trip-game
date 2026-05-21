package graph

import (
	"sort"
	"strings"
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
	return &model.Session{ID: s.ID, MapName: s.MapName, CreatedAt: timeString(s.CreatedAt), LastAccessedAt: timeString(s.LastAccessedAt), GameState: toGameState(s.GameState), GameMap: toGameMap(s.GameMap)}
}

func toUnifiedSession(s *service.SessionInfo) *model.UnifiedSession {
	if s == nil {
		return nil
	}
	return &model.UnifiedSession{SessionID: s.ID, CreatedAt: timeString(s.CreatedAt), LastAccessedAt: timeString(s.LastAccessedAt), GameState: toGameState(s.GameState), GameMap: toGameMap(s.GameMap)}
}

func toGameState(gs *engine.GameState) *model.GameState {
	if gs == nil {
		return nil
	}
	grid := make([][]*model.Cell, len(gs.Grid))
	for y := range gs.Grid {
		grid[y] = make([]*model.Cell, len(gs.Grid[y]))
		for x, c := range gs.Grid[y] {
			grid[y][x] = &model.Cell{Type: string(c.Type), Visited: c.Visited, ID: c.ID}
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
	local := make([]*model.SurroundingCell, len(gs.LocalView))
	for i, c := range gs.LocalView {
		local[i] = &model.SurroundingCell{X: c.X, Y: c.Y, Type: string(c.Type)}
	}
	return &model.GameState{Grid: grid, PlayerPos: toPosition(gs.PlayerPos), Battery: gs.Battery, MaxBattery: gs.MaxBattery, Score: gs.Score, VisitedParks: visited, Message: gs.Message, GameOver: gs.GameOver, Victory: gs.Victory, MapName: gs.MapName, MoveHistory: toMoveHistory(gs.MoveHistory), TotalMoves: gs.TotalMoves, LocalView: local, CurrentMoves: toMoveHistory(gs.CurrentMoves), CurrentMovesCount: gs.CurrentMovesCount, LocalView3x3: gs.LocalView3x3, BatteryRisk: gs.BatteryRisk}
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
	m := c.Messages
	return &model.GameMap{Name: c.Name, Description: c.Description, GridSize: c.GridSize, MaxBattery: c.MaxBattery, StartingBattery: c.StartingBattery, Layout: c.Layout, Legend: legend, WallCrashEndsGame: c.WallCrashEndsGame, Messages: &model.MapMessages{Welcome: m.Welcome, HomeCharge: m.HomeCharge, SuperchargerCharge: m.SuperchargerCharge, ParkVisited: m.ParkVisited, ParkAlreadyVisited: m.ParkAlreadyVisited, Victory: m.Victory, OutOfBattery: m.OutOfBattery, Stranded: m.Stranded, CantMove: m.CantMove, BatteryStatus: m.BatteryStatus, HitWall: m.HitWall}}
}

func fromGameMapInput(in model.GameMapInput) *engine.GameConfig {
	legend := make(map[string]string, len(in.Legend))
	for _, e := range in.Legend {
		if e != nil {
			legend[e.Key] = e.Value
		}
	}
	c := &engine.GameConfig{Name: in.Name, Description: in.Description, GridSize: in.GridSize, MaxBattery: in.MaxBattery, StartingBattery: in.StartingBattery, Layout: in.Layout, Legend: legend, WallCrashEndsGame: in.WallCrashEndsGame}
	if in.Messages != nil {
		c.Messages.Welcome = in.Messages.Welcome
		c.Messages.HomeCharge = in.Messages.HomeCharge
		c.Messages.SuperchargerCharge = in.Messages.SuperchargerCharge
		c.Messages.ParkVisited = in.Messages.ParkVisited
		c.Messages.ParkAlreadyVisited = in.Messages.ParkAlreadyVisited
		c.Messages.Victory = in.Messages.Victory
		c.Messages.OutOfBattery = in.Messages.OutOfBattery
		c.Messages.Stranded = in.Messages.Stranded
		c.Messages.CantMove = in.Messages.CantMove
		c.Messages.BatteryStatus = in.Messages.BatteryStatus
		c.Messages.HitWall = in.Messages.HitWall
	}
	return c
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
	return &model.BulkMoveResult{MovesExecuted: r.MovesExecuted, TotalMoves: r.TotalMoves, RequestedMoves: r.RequestedMoves, Success: r.Success, GameState: toGameState(r.GameState), Events: toEvents(r.Events), StoppedReason: r.StoppedReason, StopReasonCode: r.StopReasonCode, StoppedOnMove: r.StoppedOnMove, Truncated: r.Truncated, Limit: r.Limit, StartPos: toPosition(r.StartPos), EndPos: toPosition(r.EndPos), StartBattery: r.StartBattery, EndBattery: r.EndBattery, ScoreDelta: r.ScoreDelta, Steps: steps, AttemptedTo: toAttempt(r.AttemptedTo), GameOver: r.GameOver, GameOverCode: r.GameOverCode, Message: r.Message, PossibleMoves: r.PossibleMoves, LocalView3x3: r.LocalView3x3, BatteryRisk: r.BatteryRisk}
}
func toHistory(h *service.HistoryResponse) *model.HistoryResponse {
	if h == nil {
		return nil
	}
	return &model.HistoryResponse{Moves: toMoveHistory(h.Moves), TotalMoves: h.TotalMoves, Page: h.Page, PageSize: h.PageSize, TotalPages: h.TotalPages, HasNext: h.HasNext, HasPrevious: h.HasPrevious}
}
func directionString(d model.Direction) string { return strings.ToLower(string(d)) }
