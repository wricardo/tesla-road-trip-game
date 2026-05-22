// Command tesla-road-trip starts the Tesla Road Trip Game server.
//
// It supports two modes:
//  1. "server" (default) – runs the HTTP server exposing GraphQL, WebSocket, and an /mcp HTTP endpoint
//  2. "stdio-mcp" – runs an MCP stdio server and spins up an internal HTTP API if none is available
//
// Flags control host/port, config directory, debug logging, version output,
// and optional ngrok tunneling for easy external access during development.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/joho/godotenv"
	"github.com/wricardo/tesla-road-trip-game/api"
	"github.com/wricardo/tesla-road-trip-game/game/config"
	"github.com/wricardo/tesla-road-trip-game/game/service"
	"github.com/wricardo/tesla-road-trip-game/game/session"
	"github.com/wricardo/tesla-road-trip-game/graph"
	"github.com/wricardo/tesla-road-trip-game/graph/generated"
	mcptransport "github.com/wricardo/tesla-road-trip-game/transport/mcp"
	"github.com/wricardo/tesla-road-trip-game/transport/websocket"
	"golang.ngrok.com/ngrok"
	ngrokConfig "golang.ngrok.com/ngrok/config"
)

// Version information
const (
	Version = "2.0.0"
	AppName = "Tesla Road Trip Game Server"
)

// Configuration flags control how the server starts and which services are enabled.
var (
	port         = flag.Int("port", 8080, "HTTP server port")
	host         = flag.String("host", "localhost", "HTTP server host")
	configDir    = flag.String("config-dir", getConfigDirDefault(), "Directory containing game configurations")
	debug        = flag.Bool("debug", false, "Enable debug logging")
	version      = flag.Bool("version", false, "Show version information")
	ngrokEnabled = flag.Bool("ngrok", false, "Enable ngrok tunnel")
	ngrokAuth    = flag.String("ngrok-auth", "", "Ngrok auth token (or use NGROK_AUTHTOKEN env var)")
	ngrokDomain  = flag.String("ngrok-domain", "", "Custom ngrok domain (optional)")
	publicURL    = flag.String("public-url", "", "Public base URL served in llms.txt (e.g. https://myserver.com). Defaults to http://<host>:<port>")
)

var llmsTxtTemplate = template.Must(template.New("llms").Parse(`# Tesla Road Trip Game — LLM Guide

Grid-based navigation game. Control a Tesla, collect all parks, manage battery. Win by visiting every park.

GraphQL endpoint:       POST {{.BaseURL}}/graphql
GraphQL subscriptions:  ws(s)://<host>/graphql
Playground:             GET  {{.BaseURL}}/playground
MCP endpoint:           POST {{.BaseURL}}/mcp  (Streamable HTTP transport)
Introspection:          enabled — query __schema/__type or use Playground docs
GraphQL CLI client:     https://github.com/wricardo/gqlcli

---

## Quick Start

### 1. List available maps

` + "```" + `graphql
query {
  maps {
    mapId
    name
    description
    gridSize
    maxBattery
  }
}
` + "```" + `

### 2. Create a session

` + "```" + `graphql
mutation {
  createSession(mapID: "easy") {
    id
    mapName
    gameState {
      battery
      maxBattery
      playerPos { x y }
      grid { type visited id }
      victory
      gameOver
    }
  }
}
` + "```" + `

### 3. Get current game state

` + "```" + `graphql
query {
  gameState(sessionID: "SESSION_ID") {
    playerPos { x y }
    battery
    maxBattery
    batteryRisk
    score
    victory
    gameOver
    message
    visitedParks { id visited }
    localView3x3
    grid { type visited id }
  }
}
` + "```" + `

` + "`" + `localView3x3` + "`" + ` — 3-row ASCII snapshot centered on the player. Quick spatial awareness without parsing the full grid.
` + "`" + `batteryRisk` + "`" + ` — one of: ` + "`" + `SAFE` + "`" + `, ` + "`" + `LOW` + "`" + `, ` + "`" + `CAUTION` + "`" + `, ` + "`" + `DANGER` + "`" + `, ` + "`" + `CRITICAL` + "`" + `, ` + "`" + `WARNING` + "`" + `, ` + "`" + `UNKNOWN` + "`" + `.

### 4. Move (single)

` + "```" + `graphql
mutation {
  move(sessionID: "SESSION_ID", direction: RIGHT) {
    success
    message
    gameState { playerPos { x y } battery victory gameOver }
    attemptedTo { x y tileChar tileType passable }
    step { tileType charged park batteryAfter victory }
  }
}
` + "```" + `

Directions: ` + "`" + `UP` + "`" + ` ` + "`" + `DOWN` + "`" + ` ` + "`" + `LEFT` + "`" + ` ` + "`" + `RIGHT` + "`" + `
Optional ` + "`" + `reset: true` + "`" + ` resets before executing.

### 5. Bulk move (sequence)

` + "```" + `graphql
mutation {
  bulkMove(sessionID: "SESSION_ID", moves: [UP, UP, RIGHT, RIGHT, DOWN]) {
    movesExecuted
    requestedMoves
    success
    stoppedReason
    stopReasonCode
    truncated
    limit
    startPos { x y }
    endPos { x y }
    startBattery
    endBattery
    scoreDelta
    gameOver
    gameOverCode
    victory
    message
    possibleMoves
    localView3x3
    batteryRisk
    steps {
      idx dir
      from { x y } to { x y }
      tileChar tileType
      batteryBefore batteryAfter
      success charged park victory
    }
    gameState {
      playerPos { x y }
      battery
      victory
      gameOver
      visitedParks { id visited }
    }
  }
}
` + "```" + `

Stops early on: wall collision (if ` + "`" + `wallCrashEndsGame=true` + "`" + `), battery depletion, or victory.
Max 50 moves per call. Check ` + "`" + `stoppedReason` + "`" + ` / ` + "`" + `stopReasonCode` + "`" + ` / ` + "`" + `truncated` + "`" + ` before continuing.
Optional ` + "`" + `reset: true` + "`" + ` resets before executing.

### 5a. Long route — reset + chained bulkMoves in one request

GraphQL aliases execute mutation fields serially. Reset and run a full route in one round trip:

` + "```" + `graphql
mutation {
  reset(sessionID: "SESSION_ID") { battery score }

  c1: bulkMove(sessionID: "SESSION_ID", moves: [LEFT,LEFT,UP,UP,RIGHT,RIGHT,RIGHT,UP,UP]) {
    movesExecuted success stoppedReason
    gameState { playerPos { x y } battery victory gameOver }
  }

  c2: bulkMove(sessionID: "SESSION_ID", moves: [RIGHT,RIGHT,DOWN,DOWN,LEFT,LEFT,LEFT,DOWN]) {
    movesExecuted success stoppedReason
    gameState { playerPos { x y } battery victory gameOver }
  }
}
` + "```" + `

Each alias resumes from where the previous left off. Check ` + "`" + `stoppedReason` + "`" + ` on each segment.

### 6. Reset session

` + "```" + `graphql
mutation {
  reset(sessionID: "SESSION_ID") {
    playerPos { x y }
    battery
    maxBattery
    score
    gameOver
    victory
    message
  }
}
` + "```" + `

---

## All Queries

` + "```" + `graphql
# Single session
query {
  session(id: "SESSION_ID") {
    id mapName createdAt lastAccessedAt
    gameState { playerPos { x y } battery score victory gameOver }
    gameMap { name gridSize maxBattery }
  }
}

# All sessions (sort: CREATED|ACCESSED, order: ASC|DESC)
query {
  sessions(sort: ACCESSED, order: DESC, limit: 20) {
    count total
    sessions { id mapName lastAccessedAt gameState { victory gameOver score battery } }
  }
}

# Sessions grouped by map
query {
  unifiedSessions(mapName: "easy") {
    mapName count
    sessions {
      sessionId createdAt lastAccessedAt
      gameState { playerPos { x y } battery score victory gameOver }
      gameMap { name gridSize maxBattery }
    }
  }
}

# Full game state
query { gameState(sessionID: "SESSION_ID") { ... } }

# Move history (paginated)
query {
  history(sessionID: "SESSION_ID", page: 1, limit: 20, order: DESC) {
    totalMoves totalPages hasNext hasPrevious page pageSize
    moves {
      moveNumber action battery success timestamp
      fromPosition { x y }
      toPosition { x y }
    }
  }
}

# Map list
query {
  maps { mapId name description gridSize maxBattery }
}

# Full map details
query {
  map(name: "easy") {
    name description gridSize maxBattery startingBattery
    wallCrashEndsGame
    layout
    legend { key value }
    messages {
      welcome homeCharge superchargerCharge parkVisited
      victory outOfBattery stranded cantMove hitWall
    }
  }
}
` + "```" + `

---

## All Mutations

` + "```" + `graphql
# Create session (use mapID or mapName)
mutation { createSession(mapID: "easy") { id mapName gameState { battery playerPos { x y } } } }

# Delete session
mutation { deleteSession(id: "SESSION_ID") { message } }

# Move (single)
mutation { move(sessionID: "SESSION_ID", direction: UP) { success message gameState { battery victory gameOver playerPos { x y } } } }

# Bulk move
mutation { bulkMove(sessionID: "SESSION_ID", moves: [UP, RIGHT, DOWN]) { movesExecuted success stoppedReason victory gameOver gameState { battery playerPos { x y } } } }

# Reset
mutation { reset(sessionID: "SESSION_ID") { battery score victory gameOver playerPos { x y } } }

# Create map
mutation {
  createMap(name: "mymap", map: {
    name: "My Map"
    description: "Custom map"
    gridSize: 10
    maxBattery: 20
    startingBattery: 20
    wallCrashEndsGame: true
    layout: [
      "BBBBBBBBBB",
      "BRRRRRRRRB",
      "BRPRRRRPRB",
      "BRRRHRRRRB",
      "BSRRRRRRRB",
      "BRRRRRRRRB",
      "BRPRRRRPRB",
      "BRRRRRRRRB",
      "BRRRRRRRRB",
      "BBBBBBBBBB"
    ]
    legend: [
      { key: "R", value: "road" }
      { key: "H", value: "home" }
      { key: "P", value: "park" }
      { key: "S", value: "supercharger" }
      { key: "B", value: "building" }
      { key: "W", value: "water" }
    ]
    messages: {
      welcome: "Welcome!" homeCharge: "Charged!" superchargerCharge: "Supercharged!"
      parkVisited: "Park %d!" parkAlreadyVisited: "Already visited"
      victory: "Victory! %d parks!" outOfBattery: "Out of battery!"
      stranded: "Stranded!" cantMove: "Can't move there!" batteryStatus: "Battery: %d/%d"
      hitWall: "Hit wall!"
    }
  }) { name gridSize maxBattery }
}

# Update map (partial — omitted fields unchanged)
mutation {
  updateMap(name: "mymap", patch: {
    description: "Updated description"
    maxBattery: 25
    startingBattery: 25
  }) { name description maxBattery startingBattery }
}
` + "```" + `

---

## Subscriptions

` + "```" + `graphql
# Real-time updates for a session (WebSocket)
subscription {
  sessionUpdated(sessionID: "SESSION_ID") {
    battery maxBattery score victory gameOver totalMoves message mapName
    playerPos { x y }
    grid { type visited id }
    currentMoves { fromPosition { x y } toPosition { x y } success }
  }
}

# Lobby updates (any session change)
subscription {
  lobbyUpdated {
    battery score victory gameOver mapName playerPos { x y }
  }
}
` + "```" + `

WebSocket URL: ` + "`" + `ws://<host>/graphql` + "`" + ` (` + "`" + `wss://` + "`" + ` over TLS). Uses graphql-ws protocol.

---

## MCP (Model Context Protocol)

MCP endpoint: ` + "`" + `{{.BaseURL}}/mcp` + "`" + ` (Streamable HTTP transport)

Available tools:

| tool            | description                                         |
|-----------------|-----------------------------------------------------|
| game_state      | Get current game state for a session                |
| move            | Move one step (direction: up/down/left/right)       |
| bulk_move       | Execute multiple moves at once                      |
| reset_game      | Reset session to initial state                      |
| move_history    | Get paginated move history for a session            |
| create_session  | Create a new game session                           |
| get_session     | Get session details                                 |
| list_sessions   | List all active sessions                            |
| list_maps       | List available maps                                 |
| get_map         | Get full map details (layout, battery, messages)    |
| create_map      | Create a new map                                    |
| update_map      | Partially update a map (omitted fields unchanged)   |
| delete_map      | Permanently delete a map                            |

To use MCP in Claude Code, add to ` + "`" + `mcp.json` + "`" + `:

` + "```" + `json
{
  "mcpServers": {
    "tesla-game": {
      "type": "http",
      "url": "{{.BaseURL}}/mcp"
    }
  }
}
` + "```" + `

---

## Grid Cell Types

| char | type         | passable | effect                 |
|------|--------------|----------|------------------------|
| R    | road         | yes      | none                   |
| H    | home         | yes      | charges battery to max |
| S    | supercharger | yes      | charges battery to max |
| P    | park         | yes      | collect to score / win |
| B    | building     | no       | impassable             |
| W    | water        | no       | impassable             |

Grid: ` + "`" + `grid[y][x]` + "`" + ` (row-major). ` + "`" + `Cell.type` + "`" + ` uses names above. ` + "`" + `Cell.id` + "`" + ` is a coordinate string.
Victory when ` + "`" + `victory: true` + "`" + ` — all parks collected.

---

## Battery

- Each move costs 1 battery.
- H or S restores to ` + "`" + `maxBattery` + "`" + `.
- ` + "`" + `battery: 0` + "`" + ` → ` + "`" + `gameOver: true` + "`" + `, ` + "`" + `gameOverCode: "battery"` + "`" + `.
- ` + "`" + `batteryRisk` + "`" + ` summarizes current risk level.

---

## Usage Notes

1. Use ` + "`" + `maps` + "`" + ` to pick a map → ` + "`" + `createSession` + "`" + ` to start.
2. Use ` + "`" + `gameState` + "`" + ` for snapshots; request only fields you need.
3. Prefer ` + "`" + `bulkMove` + "`" + ` over repeated ` + "`" + `move` + "`" + ` calls — max 50 moves per call.
4. After mutations: check ` + "`" + `success` + "`" + `, ` + "`" + `gameOver` + "`" + `, ` + "`" + `victory` + "`" + `, ` + "`" + `stoppedReason` + "`" + `, ` + "`" + `batteryRisk` + "`" + `.
5. Chain ` + "`" + `bulkMove` + "`" + ` aliases in one mutation request for long routes.
6. Introspection enabled — use Playground at ` + "`" + `{{.BaseURL}}/playground` + "`" + ` to explore schema.
`))

// getConfigDirDefault returns the default configuration directory.
// It first honors the CONFIG_DIR environment variable, then falls back to "maps".
func getConfigDirDefault() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}
	return "maps"
}

// envBool reads an env var as a boolean. Returns defaultVal if the var is unset.
// Recognized true values: "true", "1", "yes". Everything else is false.
func envBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch v {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// withHTTPRequest is middleware that stores the *http.Request in context so
// GraphQL resolvers can read headers (e.g. X-Admin-Key).
func withHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), graph.HTTPRequestKey{}, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [MODE]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s v%s\n\n", AppName, Version)
		fmt.Fprintf(os.Stderr, "Available modes:\n")
		fmt.Fprintf(os.Stderr, "  server, http     Run HTTP server with GraphQL and WebSocket (default)\n")
		fmt.Fprintf(os.Stderr, "  stdio-mcp        Disabled: MCP transport needs GraphQL migration\n")
		fmt.Fprintf(os.Stderr, "  mcp-stdio        Alias for stdio-mcp\n")
		fmt.Fprintf(os.Stderr, "  mcp              Alias for stdio-mcp\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Run HTTP server on default port 8080\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 9090         # Run HTTP server on port 9090\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s stdio-mcp          # Run MCP stdio server\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s mcp -port 9090     # Run MCP stdio server with internal HTTP on port 9090\n", os.Args[0])
	}
}

// main parses flags, initializes services, and starts the selected mode.
func main() {
	// Load .env file if it exists (ignore error if not found)
	if err := godotenv.Load(); err != nil {
		// Only log if it's not a "file not found" error
		if !os.IsNotExist(err) {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	} else {
		log.Println("Loaded environment variables from .env file")
	}

	flag.Parse()

	// Show version if requested
	if *version {
		fmt.Printf("%s v%s\n", AppName, Version)
		os.Exit(0)
	}

	// Setup logging
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	// Determine mode from command
	args := flag.Args()
	mode := "server" // default
	if len(args) > 0 {
		mode = args[0]
	}

	log.Printf("Starting %s v%s (mode: %s)", AppName, Version, mode)

	// Initialize services
	gameService, err := initializeServices()
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	switch mode {
	case "stdio-mcp", "mcp-stdio", "mcp":
		log.Fatalf("stdio MCP is disabled because the REST API was removed; migrate MCP transport to GraphQL first")
		return

	case "server", "http":
		// Run HTTP server with GraphQL and WebSocket
		runHTTPServer(gameService)

	default:
		log.Fatalf("Unknown mode: %s. Use 'server' (default) or 'stdio-mcp'", mode)
	}
}

// runHTTPServer starts the HTTP server with GraphQL, WebSocket hub, and an /mcp proxy endpoint.
// If ngrok is enabled (via flag or environment), it also provisions a public tunnel.
func runHTTPServer(gameService service.GameService) {
	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create static/WebSocket server
	apiServer := api.NewServer(gameService, hub)

	// Setup HTTP server address
	addr := fmt.Sprintf("%s:%d", *host, *port)

	// Create main router.
	mainRouter := http.NewServeMux()

	// Feature gates from environment variables.
	// All default to enabled to preserve backward compatibility.
	introspectionEnabled := envBool("GRAPHQL_INTROSPECTION", true)
	playgroundEnabled := envBool("GRAPHQL_PLAYGROUND", true)
	mcpEnabled := envBool("MCP_ENABLED", true)

	// GraphQL is the public game API.
	graphqlResolver := graph.NewResolver(gameService, hub)
	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver}))
	if introspectionEnabled {
		gqlSrv.Use(extension.Introspection{})
		log.Println("GraphQL introspection: enabled (set GRAPHQL_INTROSPECTION=false to disable)")
	} else {
		log.Println("GraphQL introspection: disabled")
	}
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(&transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader:              websocket.DefaultUpgrader(),
	})
	mainRouter.Handle("/graphql", withHTTPRequest(gqlSrv))
	if playgroundEnabled {
		mainRouter.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))
		log.Println("GraphQL playground: enabled (set GRAPHQL_PLAYGROUND=false to disable)")
	} else {
		log.Println("GraphQL playground: disabled")
	}

	// /llms.txt — rendered from template so the server URL is always correct.
	baseURL := *publicURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s", addr)
	}
	mainRouter.HandleFunc("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := llmsTxtTemplate.Execute(&buf, struct{ BaseURL string }{BaseURL: baseURL}); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(buf.Bytes())
	})

	// MCP HTTP endpoint (Streamable HTTP transport).
	if mcpEnabled {
		mcpSrv := mcptransport.NewServer(gameService)
		mainRouter.Handle("/mcp", mcpSrv.Handler())
		log.Println("MCP endpoint: enabled at /mcp (set MCP_ENABLED=false to disable)")
	} else {
		log.Println("MCP endpoint: disabled")
	}

	// Mount static UI and WebSocket routes at root.
	mainRouter.Handle("/", apiServer)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mainRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start regular HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("HTTP server listening on %s", addr)
		log.Printf("GraphQL API: http://%s/graphql", addr)
		log.Printf("GraphQL playground: http://%s/playground", addr)
		log.Printf("WebSocket: ws://%s/ws?session=<session_id>", addr)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Check if ngrok should be enabled (from flag or environment)
	ngrokShouldRun := *ngrokEnabled
	if !ngrokShouldRun {
		// Check environment variable if flag not set
		if envEnabled := os.Getenv("NGROK_ENABLED"); envEnabled == "true" || envEnabled == "1" {
			ngrokShouldRun = true
		}
	}

	// Start ngrok tunnel if enabled
	if ngrokShouldRun {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Get auth token from flag or environment (support both naming conventions)
			authToken := *ngrokAuth
			if authToken == "" {
				authToken = os.Getenv("NGROK_AUTHTOKEN")
				if authToken == "" {
					authToken = os.Getenv("NGROK_AUTH_TOKEN") // Also support underscore version
				}
			}

			if authToken == "" {
				log.Println("WARNING: Ngrok enabled but no auth token provided (use --ngrok-auth, NGROK_AUTHTOKEN, or NGROK_AUTH_TOKEN env var)")
				return
			}

			log.Println("Starting ngrok tunnel...")

			// Get domain from flag or environment
			domain := *ngrokDomain
			if domain == "" {
				domain = os.Getenv("NGROK_DOMAIN")
			}

			// Configure ngrok endpoint
			var tunnel ngrokConfig.Tunnel
			if domain != "" {
				tunnel = ngrokConfig.HTTPEndpoint(ngrokConfig.WithDomain(domain))
				log.Printf("Using custom ngrok domain: %s", domain)
			} else {
				tunnel = ngrokConfig.HTTPEndpoint()
			}

			// Start ngrok tunnel
			tun, err := ngrok.Listen(ctx,
				tunnel,
				ngrok.WithAuthtoken(authToken),
			)
			if err != nil {
				log.Printf("Failed to start ngrok tunnel: %v", err)
				return
			}
			defer func() {
				if err := tun.Close(); err != nil {
					log.Printf("Failed to close ngrok tunnel: %v", err)
				}
			}()

			ngrokURL := tun.URL()
			log.Printf("🚀 Ngrok tunnel established: %s", ngrokURL)
			log.Printf("  GraphQL API (ngrok): %s/graphql", ngrokURL)
			log.Printf("  GraphQL playground (ngrok): %s/playground", ngrokURL)
			log.Printf("  WebSocket (ngrok): %s/ws?session=<session_id>", ngrokURL)
			log.Printf("  Game UI (ngrok): %s/", ngrokURL)

			// Serve HTTP through ngrok tunnel
			if err := http.Serve(tun, mainRouter); err != nil && err != http.ErrServerClosed {
				log.Printf("Ngrok server error: %v", err)
			}
			log.Println("Ngrok tunnel closed")
		}()
	}

	// Wait for shutdown signal
	sig := <-stop
	log.Printf("Received signal: %v. Shutting down...", sig)
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("Server stopped")
}

// initializeServices wires session/config managers and the game service.
// It also starts a background cleanup routine to prune stale sessions.
func initializeServices() (service.GameService, error) {
	// Create config manager first (needed for persistence)
	configManager, err := config.NewManager(*configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	// Create session persistence
	sessionsDir := "sessions"
	persistence, err := session.NewFilePersistence(sessionsDir, configManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create session persistence: %w", err)
	}

	// Create session manager with persistence
	sessionManager := session.NewManagerWithPersistence(persistence)

	// Load persisted sessions on startup
	if err := sessionManager.LoadPersistedSessions(); err != nil {
		log.Printf("Warning: Failed to load persisted sessions: %v", err)
	}

	// Create game service
	gameService := service.NewGameService(sessionManager, configManager)

	// Start session cleanup routine
	go sessionCleanupRoutine(sessionManager)

	// Start filesystem sync routine
	go filesystemSyncRoutine(sessionManager, persistence)

	return gameService, nil
}

// sessionCleanupRoutine periodically removes sessions that have not been accessed
// within the provided retention window.
func sessionCleanupRoutine(manager *session.Manager) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		removed := manager.CleanupExpiredSessions(24 * time.Hour)
		if removed > 0 {
			log.Printf("Cleaned up %d expired sessions", removed)
		}
	}
}

// filesystemSyncRoutine periodically syncs in-memory sessions with filesystem state.
// It removes sessions from memory when their corresponding files are deleted.
func filesystemSyncRoutine(manager *session.Manager, persistence session.SessionPersistence) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Skip if no persistence configured
		if persistence == nil {
			continue
		}

		// Get all sessions from memory
		memorySessions := manager.List()

		// Check each memory session against filesystem
		pruned := 0
		for _, session := range memorySessions {
			if !persistence.Exists(session.ID) {
				// File deleted, remove from memory
				if err := manager.DeleteFromMemory(session.ID); err == nil {
					pruned++
					log.Printf("Pruned session %s from memory (file deleted)", session.ID)
				}
			}
		}

		if pruned > 0 {
			log.Printf("Filesystem sync: pruned %d orphaned sessions from memory", pruned)
		}
	}
}
