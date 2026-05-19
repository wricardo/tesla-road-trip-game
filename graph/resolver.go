package graph

import (
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
)

// Resolver is the gqlgen dependency injection root.
type Resolver struct {
	Service service.GameService
	Hub     *websocket.Hub
}

func NewResolver(gameService service.GameService, hub *websocket.Hub) *Resolver {
	return &Resolver{Service: gameService, Hub: hub}
}
