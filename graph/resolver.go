package graph

import (
	"context"
	"errors"
	"net/http"
	"os"

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

// HTTPRequestKey is the context key for the *http.Request injected by the
// withHTTPRequest middleware in main.go. Exported so main can use the same key.
type HTTPRequestKey struct{}

// checkAdminKey returns an error if ADMIN_API_KEY is set and the request's
// X-Admin-Key header does not match. No-ops when ADMIN_API_KEY is unset.
func checkAdminKey(ctx context.Context) error {
	required := os.Getenv("ADMIN_API_KEY")
	if required == "" {
		return nil
	}
	r, _ := ctx.Value(HTTPRequestKey{}).(*http.Request)
	if r == nil {
		return errors.New("admin operation requires X-Admin-Key header")
	}
	if r.Header.Get("X-Admin-Key") != required {
		return errors.New("forbidden: invalid or missing X-Admin-Key")
	}
	return nil
}
