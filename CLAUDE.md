# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Tesla Road Trip Game — grid-based multi-session game server in Go where players control a Tesla to collect parks while managing battery. Features: per-session configuration, GraphQL API (gqlgen), REST API, WebSocket, MCP server, SvelteKit frontend, and ngrok tunneling.

## Development Commands

Server uses default port 8080. Use `make` for all common tasks.

### Build and Run
```bash
make build          # Build binary → ./statefullgame
make run            # Build + run (port 8080)
make dev            # Same as run, explicit port 8080
./statefullgame -port 9090              # Custom port
./statefullgame -ngrok                  # With ngrok tunnel
./statefullgame stdio-mcp               # MCP stdio mode
```

### Testing
```bash
make test                   # All Go tests
make test-verbose            # Verbose output
make test-coverage           # Coverage report → coverage.html
go test ./game/engine/...    # Single package
go test -run TestName ./...  # Single test
make validate                # Validate all configs
```

### Code Quality
```bash
make verify         # fmt-check + vet-safe + lint (fast CI)
make fmt            # Format (gofmt -s + goimports)
make lint           # golangci-lint
make vet-safe       # go vet core packages (skips flaky test pkg)
```

### Frontend (SvelteKit + Tailwind)
```bash
cd frontend
npm install
npm run dev         # Dev server (Vite, proxies to Go backend)
npm run build       # Static build → frontend/build/
npm run check       # Type-check
```

### GraphQL Code Generation
```bash
go run github.com/99designs/gqlgen generate   # Regenerate from schema.graphqls
```
Schema lives in `graph/schema.graphqls`. Do not edit `graph/generated/` directly.

### MCP Server
```bash
make claude-game         # Claude with HTTP MCP (./mcp.json)
make claude-game-stdin   # Claude with stdio MCP (./mcp-stdin.json)
./statefullgame stdio-mcp  # Standalone stdio MCP
```

### Config Analysis Tool
```bash
cd cmd/analyze && go run .   # Prints heuristics: dimensions, battery, parks, chargers, reachability
```

## Architecture

### Package Structure
```
main.go                  Entry point — wires all packages, handles flags, starts server
api/                     REST HTTP handlers (GET/POST /api, sessions, saves, maps)
game/
  config/                Config file loading and management (maps/*.json)
  engine/                Core game logic: movement, battery, victory, cell types
  service/               GameService interface + implementation (facade over engine+session)
  session/               Session lifecycle, in-memory manager, file persistence
graph/
  schema.graphqls        GraphQL schema (source of truth)
  generated/             gqlgen-generated code (do not edit)
  resolver.go            Resolver wiring
  schema.resolvers.go    Mutation/Query/Subscription implementations
transport/
  mcp/                   MCP server (tools: game_state, move, bulk_move, create_session, etc.)
  websocket/             WebSocket hub — broadcasts state updates to session subscribers
validate/                Config winnability checks
cmd/analyze/             CLI: config heuristics analyzer
frontend/                SvelteKit app (Tailwind, Svelte 5)
  src/routes/
    +page.svelte         Home / game selection
    multi/               Multi-player lobby
    watch/[id]/          Spectator view
    learn/               Tutorial/learn page
    admin/               Admin panel
    lobby/               Session lobby
```

### Key Interfaces
- `game/service.GameService` — primary facade used by all transport layers (REST, GraphQL, MCP)
- `game/session.Manager` — session CRUD + persistence
- `game/engine` — stateless game logic; takes/returns `GameState`

### Data Flow
1. **Session creation**: client → REST or GraphQL → `GameService.CreateSession` → `session.Manager` → persisted to `sessions/`
2. **Move**: client → REST (`/api`) or GraphQL mutation (`move`/`bulkMove`) → `GameService.Move` → `engine.ApplyMove` → state saved → WebSocket broadcast
3. **MCP**: stdio or HTTP `/mcp` → MCP tools → same `GameService` interface

### GraphQL Endpoints
- `/graphql` — GraphQL HTTP endpoint + playground (GET opens playground, POST executes)
- `/ws` — WebSocket (upgrades for real-time subscriptions, also accepts `?sessionId=`)

### REST Endpoints
- `GET/POST /api` — game state / move (`?sessionId=`)
- `POST /api/sessions` — create session (`{"map_name":"easy"}`)
- `GET /api/sessions/{id}` — session state
- `GET /api/maps`, `GET /api/saves`, `GET /api/history`
- `GET /llms.txt` — LLM-readable server guide (auto-generated, includes public URL)

### Configuration System

Configs at `maps/*.json`. Grid cell characters:
- `R` = road (passable) — **often hidden between B/W, look carefully**
- `H` = home base (passable, full battery charge)
- `P` = park (objective)
- `S` = supercharger (passable, full battery charge)
- `B` = building (impassable)
- `W` = water (impassable)

Battery: move costs 1, H/S restore to max. Victory: visit all parks.

**⚠️ R is visually similar to B in monospace. Re-parse any "fully blocked" row character by character before concluding it's impassable.**

### ngrok Integration
`-ngrok` flag starts an ngrok tunnel. Auth token via `-ngrok-auth` flag or `NGROK_AUTHTOKEN` env var. Domain via `-ngrok-domain`. Public URL is served in `/llms.txt`.

## AI Strategy Guidelines (Gameplay)

When playing the Tesla game, ask: where is the nearest charger/home? How much battery remains? Any walls nearby?

### Navigation
- Map grid character-by-character before planning routes
- Identify obstacle-free rows/columns ("golden corridors") for long-distance travel
- When direct routes are blocked, try perpendicular approaches
- Maintain 3+ battery buffer when far from charging

### Systematic Play
1. **Analyze**: map all parks, chargers, obstacle clusters
2. **Plan**: section-based routes through safe corridors
3. **Execute**: `bulkMove` for known safe paths, single `move` near obstacles
4. **Refine**: when blocked, re-parse that row, try alternative approach angle

### Common Pitfalls
- Assuming a row is fully blocked without character-by-character check
- Depleting battery without a clear path to a charger
- Confusing `R` (road) with `B` (building) — they look similar in monospace

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
