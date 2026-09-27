package graph

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

// waitGoroutines polls until the goroutine count drops to at most want.
func waitGoroutines(t *testing.T, want int) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		n := runtime.NumGoroutine()
		if n <= want || time.Now().After(deadline) {
			return n
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestSubscriptionForwardersExitWhenReaderStalls reproduces the production
// leak: a stalled client stops draining the subscription channel, then its
// context is cancelled. gqlgen never reads the channel again, so the forwarder
// must exit on ctx.Done instead of blocking on the send forever.
func TestSubscriptionForwardersExitWhenReaderStalls(t *testing.T) {
	hub := websocket.NewHub()
	r := &subscriptionResolver{&Resolver{Hub: hub}}
	before := runtime.NumGoroutine()

	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		if _, err := r.LobbyUpdated(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := r.SessionUpdated(ctx, "s"); err != nil {
			t.Fatal(err)
		}
		// Overfill every buffer without ever reading the returned channels.
		for j := 0; j < 100; j++ {
			hub.BroadcastToSession("s", &engine.GameState{Battery: j})
		}
		cancel()
	}

	if after := waitGoroutines(t, before); after > before {
		t.Fatalf("leaked goroutines: before=%d after=%d", before, after)
	}
}

func TestSubscriptionUpdatesAreSharedSnapshots(t *testing.T) {
	hub := websocket.NewHub()
	r := &subscriptionResolver{&Resolver{Hub: hub}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, _ := r.LobbyUpdated(ctx)
	b, _ := r.SessionUpdated(ctx, "s")

	live := &engine.GameState{Battery: 7, MoveHistory: []engine.MoveHistoryEntry{{Action: "up"}}}
	hub.BroadcastToSession("s", live)
	live.Battery = 0 // mutation after broadcast must not be observed
	live.MoveHistory[0].Action = "down"

	ga, gb := <-a, <-b
	if ga != gb {
		t.Fatal("expected subscribers of one broadcast to share one converted state")
	}
	if ga.Battery != 7 || ga.MoveHistory[0].Action != "up" {
		t.Fatalf("subscriber observed live state mutation: battery=%d action=%q", ga.Battery, ga.MoveHistory[0].Action)
	}
}
