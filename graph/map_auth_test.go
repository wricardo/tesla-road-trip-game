package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newOwnedMapServer builds a GraphQL server over a temp maps dir where
// "classic" is built-in, so the whole authorization policy can be exercised.
func newOwnedMapServer(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	if err := os.MkdirAll(builtin, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	copyMapFile(t, "../maps/classic.json", filepath.Join(builtin, "classic.json"))
	t.Setenv("CUSTOM_MAPS_DIR", filepath.Join(root, "custom"))
	t.Setenv("ADMIN_API_KEY", "")

	return newTestGraphQLServerAt(t, builtin)
}

func copyMapFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

const createMapMutation = `mutation($name: String!, $map: GameMapInput!) {
  createMap(name: $name, map: $map) { name builtin editKey }
}`

const updateMapMutation = `mutation($name: String!, $patch: GameMapPatchInput!, $editKey: String) {
  updateMap(name: $name, patch: $patch, editKey: $editKey) { name description builtin }
}`

const deleteMapMutation = `mutation($name: String!, $editKey: String) {
  deleteMap(name: $name, editKey: $editKey) { message }
}`

func mapInput(name string) map[string]any {
	return map[string]any{
		"name":            name,
		"description":     "test map",
		"gridSize":        5,
		"maxBattery":      10,
		"startingBattery": 10,
		"layout":          []string{"BBBBB", "BRHPB", "BRRSB", "BPPPB", "BBBBB"},
		"legend":          []any{map[string]any{"key": "R", "value": "road"}, map[string]any{"key": "H", "value": "home"}, map[string]any{"key": "P", "value": "park"}, map[string]any{"key": "S", "value": "supercharger"}, map[string]any{"key": "W", "value": "water"}, map[string]any{"key": "B", "value": "building"}},
		"cellConfigs":     []any{},
	}
}

func TestCreateMapNeedsNoCredentialAndReturnsEditKey(t *testing.T) {
	url := newOwnedMapServer(t)

	resp := gqlPost(t, url, createMapMutation, map[string]any{"name": "freebie", "map": mapInput("freebie")})
	if len(resp.Errors) > 0 {
		t.Fatalf("createMap without admin key should succeed, got %v", resp.Errors)
	}

	var out struct {
		CreateMap struct {
			Name    string  `json:"name"`
			Builtin bool    `json:"builtin"`
			EditKey *string `json:"editKey"`
		} `json:"createMap"`
	}
	decodeData(t, resp.Data, &out)

	if out.CreateMap.Builtin {
		t.Error("a newly created map must not be reported as built-in")
	}
	if out.CreateMap.EditKey == nil || *out.CreateMap.EditKey == "" {
		t.Fatal("createMap must return an editKey")
	}
}

func TestEditKeyAuthorizesUpdateAndDelete(t *testing.T) {
	url := newOwnedMapServer(t)

	created := gqlPost(t, url, createMapMutation, map[string]any{"name": "mine", "map": mapInput("mine")})
	var createOut struct {
		CreateMap struct {
			EditKey *string `json:"editKey"`
		} `json:"createMap"`
	}
	decodeData(t, created.Data, &createOut)
	editKey := *createOut.CreateMap.EditKey

	// Wrong token is rejected.
	bad := gqlPost(t, url, updateMapMutation, map[string]any{
		"name": "mine", "patch": map[string]any{"description": "nope"}, "editKey": "wrong",
	})
	if len(bad.Errors) == 0 {
		t.Fatal("updateMap with a bad editKey should fail")
	}

	// Missing token is rejected too.
	missing := gqlPost(t, url, updateMapMutation, map[string]any{
		"name": "mine", "patch": map[string]any{"description": "nope"},
	})
	if len(missing.Errors) == 0 {
		t.Fatal("updateMap without an editKey should fail")
	}

	// Correct token works.
	ok := gqlPost(t, url, updateMapMutation, map[string]any{
		"name": "mine", "patch": map[string]any{"description": "updated"}, "editKey": editKey,
	})
	if len(ok.Errors) > 0 {
		t.Fatalf("updateMap with the right editKey should succeed, got %v", ok.Errors)
	}

	del := gqlPost(t, url, deleteMapMutation, map[string]any{"name": "mine", "editKey": editKey})
	if len(del.Errors) > 0 {
		t.Fatalf("deleteMap with the right editKey should succeed, got %v", del.Errors)
	}
}

func TestBuiltinMapRejectsTokenButAllowsAdmin(t *testing.T) {
	url := newOwnedMapServer(t)

	// No credentials at all.
	blocked := gqlPost(t, url, updateMapMutation, map[string]any{
		"name": "classic", "patch": map[string]any{"description": "hacked"},
	})
	if len(blocked.Errors) == 0 {
		t.Fatal("updating a built-in map without an admin key should fail")
	}

	// A made-up token must not be enough either.
	tokenTry := gqlPost(t, url, updateMapMutation, map[string]any{
		"name": "classic", "patch": map[string]any{"description": "hacked"}, "editKey": "guess",
	})
	if len(tokenTry.Errors) == 0 {
		t.Fatal("a token must never unlock a built-in map")
	}

	// Deleting a built-in map is refused as well.
	del := gqlPost(t, url, deleteMapMutation, map[string]any{"name": "classic"})
	if len(del.Errors) == 0 {
		t.Fatal("deleting a built-in map without an admin key should fail")
	}
}

func TestAdminKeyOverridesOwnership(t *testing.T) {
	url := newOwnedMapServer(t)
	t.Setenv("ADMIN_API_KEY", "s3cret")

	req := newAdminRequest(t, url, updateMapMutation, map[string]any{
		"name": "classic", "patch": map[string]any{"description": "admin edit"},
	}, "s3cret")
	resp := doGQL(t, req)
	if len(resp.Errors) > 0 {
		t.Fatalf("admin should be able to edit a built-in map, got %v", resp.Errors)
	}
}

func TestCreateMapRejectsDuplicateName(t *testing.T) {
	url := newOwnedMapServer(t)

	first := gqlPost(t, url, createMapMutation, map[string]any{"name": "dup", "map": mapInput("dup")})
	if len(first.Errors) > 0 {
		t.Fatalf("first create should succeed: %v", first.Errors)
	}

	second := gqlPost(t, url, createMapMutation, map[string]any{"name": "dup", "map": mapInput("dup")})
	if len(second.Errors) == 0 {
		t.Fatal("creating a map with an existing name should fail")
	}
	if !strings.Contains(second.Errors[0].Message, "already exists") {
		t.Errorf("expected an 'already exists' error, got %q", second.Errors[0].Message)
	}

	// Built-in names are protected from creation overwrites too.
	builtinTry := gqlPost(t, url, createMapMutation, map[string]any{"name": "classic", "map": mapInput("classic")})
	if len(builtinTry.Errors) == 0 {
		t.Fatal("createMap must not overwrite a built-in map")
	}
}

// Session payloads embed a map, so they must report ownership correctly too.
func TestSessionPayloadReportsMapOwnership(t *testing.T) {
	url := newOwnedMapServer(t)

	builtinSession := gqlPost(t, url, `mutation { createSession(mapID: "classic") { mapName gameMap { name builtin } } }`, nil)
	if len(builtinSession.Errors) > 0 {
		t.Fatalf("createSession: %v", builtinSession.Errors)
	}
	var builtinOut struct {
		CreateSession struct {
			GameMap struct {
				Builtin bool `json:"builtin"`
			} `json:"gameMap"`
		} `json:"createSession"`
	}
	decodeData(t, builtinSession.Data, &builtinOut)
	if !builtinOut.CreateSession.GameMap.Builtin {
		t.Error("a session on a built-in map should report gameMap.builtin=true")
	}

	// A user-created map must report builtin=false, even though the session
	// carries a copy of the config rather than a name lookup.
	if created := gqlPost(t, url, createMapMutation, map[string]any{"name": "mine", "map": mapInput("mine")}); len(created.Errors) > 0 {
		t.Fatalf("createMap: %v", created.Errors)
	}
	mineSession := gqlPost(t, url, `mutation { createSession(mapID: "mine") { gameMap { builtin } } }`, nil)
	if len(mineSession.Errors) > 0 {
		t.Fatalf("createSession on custom map: %v", mineSession.Errors)
	}
	var mineOut struct {
		CreateSession struct {
			GameMap struct {
				Builtin bool `json:"builtin"`
			} `json:"gameMap"`
		} `json:"createSession"`
	}
	decodeData(t, mineSession.Data, &mineOut)
	if mineOut.CreateSession.GameMap.Builtin {
		t.Error("a session on a user-created map should report gameMap.builtin=false")
	}
}
