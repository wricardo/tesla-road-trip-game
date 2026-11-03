package graph

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/graph/model"
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

// adminAuthorized reports whether the current request carries valid admin
// credentials (X-Admin-Key matches ADMIN_API_KEY, or the local-dev bypass).
// Unlike checkAdminKey it never errors, so callers can use it as an override.
func adminAuthorized(ctx context.Context) bool {
	required := os.Getenv("ADMIN_API_KEY")
	if required == "" {
		return os.Getenv("ALLOW_UNAUTHENTICATED_ADMIN") == "true"
	}
	r, _ := ctx.Value(HTTPRequestKey{}).(*http.Request)
	return r != nil && r.Header.Get("X-Admin-Key") == required
}

// ErrBuiltinReadOnly is returned when a caller tries to modify a shipped map
// without admin credentials.
var ErrBuiltinReadOnly = errors.New("forbidden: built-in maps can only be modified with an admin key")

// annotateMapOwnership fills GameMap.builtin, which is derived from where the
// map file lives. Session payloads embed a map too, so every conversion goes
// through here. `obj` must come straight from toGameMap (never from createMap).
func (r *Resolver) annotateMapOwnership(ctx context.Context, obj *model.GameMap, mapName string) *model.GameMap {
	if obj == nil {
		return nil
	}
	obj.Builtin = r.mapBuiltin(ctx, mapName)
	return obj
}

// annotateSessionMap fills the embedded GameMap.builtin for session payloads.
func (r *Resolver) annotateSessionMap(ctx context.Context, session *model.Session, info *service.SessionInfo) *model.Session {
	if session == nil || info == nil {
		return session
	}
	r.annotateMapOwnership(ctx, session.GameMap, info.MapName)
	return session
}

// mapBuiltin reports whether name is a shipped map, tolerating lookups on
// names that no longer resolve.
func (r *Resolver) mapBuiltin(ctx context.Context, name string) bool {
	builtin, exists, err := r.Service.MapAuthorization(ctx, name)
	return err == nil && exists && builtin
}

// authorizeMapMutation enforces the map write policy:
//   - built-in maps: admin key required.
//   - user-created maps: the edit token returned by createMap is required.
//   - an admin key always overrides, so operators can repair any map.
//
// It returns allowBuiltin=true only when the admin override was used, which the
// caller then passes through to the service.
func (r *Resolver) authorizeMapMutation(ctx context.Context, name string, editKey *string) (allowBuiltin bool, err error) {
	builtin, exists, err := r.Service.MapAuthorization(ctx, name)
	if err != nil || !exists {
		return false, fmt.Errorf("map %q not found", name)
	}
	if adminAuthorized(ctx) {
		return true, nil
	}
	if builtin {
		return false, ErrBuiltinReadOnly
	}
	key := ""
	if editKey != nil {
		key = strings.TrimSpace(*editKey)
	}
	if err := r.Service.VerifyMapEditToken(ctx, name, key); err != nil {
		return false, err
	}
	return false, nil
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
