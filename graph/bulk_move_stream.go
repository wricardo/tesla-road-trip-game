package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/graph/model"
)

func (r *mutationResolver) bulkMoveWithStepBroadcast(ctx context.Context, sessionID string, dirs []string, reset bool, stepDelay time.Duration) (*model.BulkMoveResult, error) {
	startState, err := r.Service.GetGameState(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	requestedMoves := len(dirs)
	limitedDirs := dirs
	truncated := false
	limit := 0
	if len(limitedDirs) > engine.MaxBulkMoves {
		truncated = true
		limit = engine.MaxBulkMoves
		limitedDirs = limitedDirs[:engine.MaxBulkMoves]
	}

	modelResult := &model.BulkMoveResult{
		MovesExecuted:  0,
		TotalMoves:     len(limitedDirs),
		RequestedMoves: requestedMoves,
		Success:        true,
		Events:         []*model.GameEvent{},
		StoppedReason:  "",
		StopReasonCode: "",
		StoppedOnMove:  0,
		Truncated:      truncated,
		Limit:          limit,
		StartPos:       toPosition(startState.PlayerPos),
		EndPos:         toPosition(startState.PlayerPos),
		StartBattery:   startState.Battery,
		EndBattery:     startState.Battery,
		ScoreDelta:     0,
		Steps:          []*model.StepInfo{},
		GameOver:       startState.GameOver,
		GameOverCode:   "",
		Message:        startState.Message,
		PossibleMoves:  []string{},
		BatteryRisk:    "",
	}

	finalState := startState
	for i, dir := range limitedDirs {
		moveResult, moveErr := r.Service.Move(ctx, sessionID, dir, reset && i == 0)
		if moveErr != nil {
			return nil, moveErr
		}
		finalState = moveResult.GameState
		if finalState == nil {
			return nil, fmt.Errorf("bulkMove step %d returned empty game state", i+1)
		}

		if moveResult.Step != nil {
			modelResult.Steps = append(modelResult.Steps, &model.StepInfo{
				Idx:           i + 1,
				Dir:           moveResult.Step.Dir,
				From:          toPosition(moveResult.Step.From),
				To:            toPosition(moveResult.Step.To),
				TileChar:      moveResult.Step.TileChar,
				TileType:      moveResult.Step.TileType,
				BatteryBefore: moveResult.Step.BatteryBefore,
				BatteryAfter:  moveResult.Step.BatteryAfter,
				Success:       moveResult.Step.Success,
				Charged:       moveResult.Step.Charged,
				Park:          moveResult.Step.Park,
				Victory:       moveResult.Step.Victory,
			})
		}

		if moveResult.Success {
			modelResult.MovesExecuted++
		} else {
			modelResult.Success = false
			modelResult.StoppedReason = moveResult.Message
			modelResult.StoppedOnMove = i + 1
			if moveResult.AttemptedTo != nil {
				modelResult.AttemptedTo = &model.AttemptInfo{
					X:        moveResult.AttemptedTo.X,
					Y:        moveResult.AttemptedTo.Y,
					TileChar: moveResult.AttemptedTo.TileChar,
					TileType: moveResult.AttemptedTo.TileType,
					Passable: moveResult.AttemptedTo.Passable,
				}
			}
		}

		if r.Hub != nil {
			r.Hub.BroadcastToSession(sessionID, finalState)
		}

		if !moveResult.Success || finalState.GameOver {
			break
		}
		if i < len(limitedDirs)-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(stepDelay):
			}
		}
	}

	modelResult.GameState = toGameState(finalState)
	modelResult.EndPos = toPosition(finalState.PlayerPos)
	modelResult.EndBattery = finalState.Battery
	modelResult.ScoreDelta = finalState.Score - startState.Score
	modelResult.GameOver = finalState.GameOver
	modelResult.Message = finalState.Message

	if modelResult.StoppedReason == "" && modelResult.GameOver {
		modelResult.StoppedReason = "game_over"
		modelResult.StopReasonCode = "game_over"
		modelResult.GameOverCode = "game_over"
	}
	if finalState.Victory {
		modelResult.GameOverCode = "victory"
		if modelResult.StopReasonCode == "" {
			modelResult.StopReasonCode = "victory"
		}
	}

	return modelResult, nil
}
