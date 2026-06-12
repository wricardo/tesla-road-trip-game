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

// checkAdminKey requires X-Admin-Key for admin mutations. ADMIN_API_KEY must be
// configured; local development can opt out explicitly with
// ALLOW_UNAUTHENTICATED_ADMIN=true.
func checkAdminKey(ctx context.Context) error {
	required := os.Getenv("ADMIN_API_KEY")
	if required == "" {
		if os.Getenv("ALLOW_UNAUTHENTICATED_ADMIN") == "true" {
			return nil
		}
		return errors.New("admin operation disabled: set ADMIN_API_KEY and send X-Admin-Key")
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
