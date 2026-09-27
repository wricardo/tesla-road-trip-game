package graph

import (
	"context"

	"github.com/wricardo/tesla-road-trip-game/graph/model"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

// forwardStateUpdates converts hub updates into GraphQL models until ctx is
// done. Every send also selects on ctx.Done: gqlgen stops reading the returned
// channel once the subscription ends, so an unconditional send would block
// this goroutine (and the states it holds) forever.
func forwardStateUpdates(ctx context.Context, updates <-chan *websocket.StateUpdate, buffer int) <-chan *model.GameState {
	out := make(chan *model.GameState, buffer)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case u, ok := <-updates:
				if !ok {
					return
				}
				select {
				case out <- websocket.Convert(u, toGameState):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
