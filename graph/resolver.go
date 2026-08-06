package graph

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"

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

type uiAuthConfig struct {
	UIMapPassword string `json:"uiMapPassword"`
}

var (
	//go:embed ui-auth.json
	embeddedUIAuthJSON []byte
	uiAuthOnce         sync.Once
	uiMapPassword      string
)

func loadUIMapPassword() string {
	uiAuthOnce.Do(func() {
		var cfg uiAuthConfig
		if err := json.Unmarshal(embeddedUIAuthJSON, &cfg); err != nil {
			uiMapPassword = ""
			return
		}
		uiMapPassword = strings.TrimSpace(cfg.UIMapPassword)
	})
	return uiMapPassword
}

// checkUIMapPassword gates privileged UI map reads (map query) behind the shared ui-auth.json password.
// If the config password is empty/unset, access is allowed for backward compatibility.
func checkUIMapPassword(password *string) error {
	required := loadUIMapPassword()
	if required == "" {
		return nil
	}
	if password == nil || *password != required {
		return errors.New("forbidden: invalid or missing map password")
	}
	return nil
}
