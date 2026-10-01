# Tesla Road Trip Game Server

[![CI](https://github.com/wricardo/tesla-road-trip-game/actions/workflows/ci.yml/badge.svg)](https://github.com/wricardo/tesla-road-trip-game/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/wricardo/tesla-road-trip-game)](https://goreportcard.com/report/github.com/wricardo/tesla-road-trip-game)

Grid-based, multi-session game server. Players drive a Tesla around a map to visit every park without running out of battery. Go backend (GraphQL via gqlgen, MCP over Streamable HTTP, WebSocket updates) with a SvelteKit web UI and a terminal client.

## Requirements

| Tool | Version | Needed for |
|------|---------|-----------|
| Go | 1.25+ (see `go.mod`) | server, TUI, tests |
| Node.js + npm | 20.19+ or 22.12+ | rebuilding the web UI (optional) |
| make | any | convenience targets (optional) |

A prebuilt copy of the web UI is committed in `static/`, so Node is only needed if you change the frontend.

## Quick Start

```bash
git clone https://github.com/wricardo/tesla-road-trip-game.git
cd tesla-road-trip-game
make build              # or: go build -o tesla-road-trip .
./tesla-road-trip       # http://localhost:8000
```

Open http://localhost:8000 to play in the browser.

The server must be started from the repo root: it reads maps from `./maps`, writes sessions to `./sessions`, and serves the UI from `./frontend/build` (if built) or `./static`.

### Full setup (with frontend)

```bash
make setup              # go mod download + npm ci in frontend/
make run                # build frontend + server, run on :8000
```

### Server options

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8000` | HTTP port |
| `-host` | `localhost` | Bind address (use `0.0.0.0` to expose on LAN) |
| `-config-dir` | `maps` | Map directory (env `CONFIG_DIR`) |
| `-sessions-dir` | `sessions` | Session persistence directory (env `SESSIONS_DIR`) |
| `-public-url` | request host | Base URL rendered in `/llms.txt` |
| `-debug` | `false` | Debug logging |

Port 8000 is often taken. Check with `lsof -i :8000` and pass `-port 9191` if needed.

### Environment (`.env`)

The server loads `.env` from the working directory if present. See [.env.example](.env.example):

```bash
cp .env.example .env
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `GRAPHQL_INTROSPECTION` | `true` | GraphQL schema introspection |
| `GRAPHQL_PLAYGROUND` | `true` | `/playground` UI |
| `MCP_ENABLED` | `true` | `/mcp` endpoint |
| `ADMIN_API_KEY` | unset | Required for map create/update/delete (`X-Admin-Key` header) |
| `ALLOW_UNAUTHENTICATED_ADMIN` | `false` | Local-only bypass for admin operations |
| `ALLOWED_ORIGINS` | all | WebSocket origin allowlist (comma-separated) |

## Endpoints

| Path | Purpose |
|------|---------|
| `/` | Web UI |
| `/graphql` | GraphQL (HTTP + graphql-ws subscriptions) |
| `/playground` | GraphQL playground |
| `/mcp` | MCP server (Streamable HTTP) — tools in [MCP_TOOLS.md](MCP_TOOLS.md) |
| `/llms.txt` | Guide for LLM agents |
| `/api/sessions` | Legacy `GET` list / `POST` create (`{"map_name":"easy"}`) |
| `/ws?session=<id>` | Legacy WebSocket state stream |

## Clients

- **Web UI** — served at `/`.
- **TUI** — `make build-tui && ./tesla-road-trip-tui -server http://localhost:8000` (add `-session <id>` to jump into a session).
- **Claude Code / MCP** — with the server running on :8000, `make claude-game` starts Claude with [mcp.json](mcp.json). For other MCP clients, point them at `http://localhost:8000/mcp`.
- **Any GraphQL client** — e.g. [gqlcli](https://github.com/wricardo/gqlcli), `curl`, or `/playground`.

## 🎲 Game Rules

### Objective
Navigate your Tesla to visit all parks (P) while managing battery life and avoiding obstacles.

### Mechanics
- **Movement**: Each move consumes 1 battery unit
- **Charging**: Restore battery at home tiles (H) or superchargers (S)
- **Obstacles**: Cannot move through water (W) or buildings (B)
- **Victory**: Collect all parks to win
- **Game Over**: Battery depleted with no reachable charging stations

### Grid Legend
- `T` - Tesla (your position)
- `R` - Road (passable)
- `H` - Home (passable, charging station)
- `P` - Park (passable, collectible objective)
- `S` - Supercharger (passable, charging station)
- `W` - Water (impassable obstacle)
- `B` - Building (impassable obstacle)
- `✓` - Visited park

## 📡 GraphQL API Reference

GraphQL endpoint: `http://localhost:8000/graphql`  
Interactive playground: `http://localhost:8000/playground`  
Subscription WebSocket endpoint: `ws://localhost:8000/graphql`  
LLM quick guide: `http://localhost:8000/llms.txt`

List maps:

```graphql
query {
  maps { mapId name description gridSize maxBattery }
}
```

Create a session:

```graphql
mutation {
  createSession(mapID: "easy") {
    id
    mapName
    gameState { playerPos { x y } battery score message }
  }
}
```

Move:

```graphql
mutation Move($sessionID: ID!) {
  move(sessionID: $sessionID, direction: RIGHT) {
    success
    message
    gameState { playerPos { x y } battery score victory gameOver }
  }
}
```

Subscribe to session updates:

```graphql
subscription Watch($sessionID: ID!) {
  sessionUpdated(sessionID: $sessionID) {
    playerPos { x y }
    battery
    score
    victory
    gameOver
  }
}
```


See [docs/graphql.md](docs/graphql.md) for the full API reference.

### GraphQL Response Enhancements

The `move` mutation returns:
- `step`: compact summary of the move
  - Fields: `dir`, `from { x y }`, `to { x y }`, `tileChar`, `tileType`, `batteryBefore`, `batteryAfter`, `success`
- `attemptedTo`: present when move metadata is available for the attempted target
  - Fields: `x`, `y`, `tileChar`, `tileType`, `passable`
- `gameState` includes:
  - `localView3x3`: three short strings centered on player (T in center)
  - `batteryRisk`: human-readable battery risk label

The `bulkMove` mutation adds:
- Summary fields: `requestedMoves`, `movesExecuted`, `stoppedReason`, `stopReasonCode`, `stoppedOnMove`, `truncated`, `limit`
- Start/end snapshot: `startPos`, `endPos`, `startBattery`, `endBattery`, `scoreDelta`
- `steps`: compact per-step entries for this call only
- `attemptedTo`: failed/attempted target metadata when available
- Decision aids: `possibleMoves`, `localView3x3`, `batteryRisk`

Notes:
- `totalMoves` is the GraphQL field name for cumulative session moves.
- Bulk responses expose both `requestedMoves` and `movesExecuted` so agents can detect truncation or blocked routes.


## Maps

Maps live in `maps/*.json` (schema: [map-schema.json](map-schema.json), docs: [docs/config-schema.md](docs/config-schema.md)). List them at runtime with the `maps` GraphQL query. Validate all maps with:

```bash
make validate
```

`cd cmd/analyze && go run .` prints heuristics (size, battery, parks, chargers, reachability) per map.

## Development

```bash
make help           # list targets
make test           # go test ./...
make test-coverage  # coverage.html
make verify         # gofmt check + go vet + golangci-lint (run `make tools` first)
make fmt            # gofmt -s + goimports
make dev-live       # backend :9090 + Vite frontend :5173 with live reload
```

Frontend only:

```bash
cd frontend
npm ci
npm run dev:local   # proxies to backend on :9090
npm test
npm run build       # → frontend/build (served by the Go server when present)
```

After changing `graph/schema.graphqls`, regenerate with `go run github.com/99designs/gqlgen generate`. Do not edit `graph/generated/`.

To update the committed UI fallback after frontend changes: `make build-frontend && rm -rf static && cp -R frontend/build static`.

### Layout

```
main.go              entry point, flags, wiring
api/                 HTTP routes, SPA serving, legacy REST
game/config          map loading
game/engine          game rules (movement, battery, victory)
game/service         GameService facade used by all transports
game/session         session lifecycle + file persistence
graph/               GraphQL schema + resolvers (generated/ is gqlgen output)
transport/mcp        MCP tools
transport/websocket  WebSocket hub
validate/            map winnability checks
cmd/tui              terminal client
cmd/analyze          map heuristics CLI
cmd/wsdebug          WebSocket debugging tool
bruteforcer/         standalone solver (own go.mod)
frontend/            SvelteKit + Tailwind source
static/              prebuilt UI fallback
maps/                map JSON files
docs/                additional docs (docs/archive is historical)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
