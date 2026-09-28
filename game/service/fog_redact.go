package service

// RedactFogJSON strips map knowledge that fog mode is meant to hide from a
// generic JSON value (the result of json.Unmarshal into `any`). Transports
// that serialize service structs directly (MCP, legacy REST) run their payload
// through this before responding: any object with "fog_enabled": true (a game
// state or a SessionInfo) loses "grid" and "game_map.layout", and so does any
// object whose "game_state" has fog enabled.
//
// The grid password itself is never serialized (engine.GameState tags it
// json:"-"). GraphQL enforces the same policy in its field resolvers.
func RedactFogJSON(v any) {
	switch node := v.(type) {
	case map[string]any:
		fog, _ := node["fog_enabled"].(bool)
		if gs, ok := node["game_state"].(map[string]any); ok {
			if f, _ := gs["fog_enabled"].(bool); f {
				fog = true
			}
		}
		if fog {
			delete(node, "grid")
			if gm, ok := node["game_map"].(map[string]any); ok {
				delete(gm, "layout")
			}
		}
		for _, child := range node {
			RedactFogJSON(child)
		}
	case []any:
		for _, child := range node {
			RedactFogJSON(child)
		}
	}
}
