// Command statefullgame starts the Tesla Road Trip Game server.
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

GraphQL endpoint: POST {{.BaseURL}}/graphql
Playground:       GET  {{.BaseURL}}/playground

---

## Quick Start

### 1. List available configs

` + "```" + `graphql
query {
  configs {
    configId
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
  createSession(configID: "easy") {
    id
    configName
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
  gameState(sessionID: "abcd") {
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

` + "`" + `localView3x3` + "`" + ` returns a 3-row ASCII snapshot centered on the player — useful for quick spatial awareness without reading the full grid.

` + "`" + `batteryRisk` + "`" + ` is one of: ` + "`" + `"safe"` + "`" + `, ` + "`" + `"moderate"` + "`" + `, ` + "`" + `"high"` + "`" + `, ` + "`" + `"critical"` + "`" + `.

### 4. Move

` + "```" + `graphql
mutation {
  move(sessionID: "abcd", direction: RIGHT) {
    success
    message
    gameState {
      playerPos { x y }
      battery
      victory
      gameOver
    }
    attemptedTo { x y tileChar tileType passable }
  }
}
` + "```" + `

Directions: ` + "`" + `UP` + "`" + ` ` + "`" + `DOWN` + "`" + ` ` + "`" + `LEFT` + "`" + ` ` + "`" + `RIGHT` + "`" + `

Optional ` + "`" + `reset: true` + "`" + ` resets the game before executing the move.

### 5. Bulk move (sequence)

` + "```" + `graphql
mutation {
  bulkMove(sessionID: "abcd", moves: [UP, UP, RIGHT, RIGHT, DOWN]) {
    movesExecuted
    success
    stoppedReason
    stopReasonCode
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

Bulk moves stop early on: wall collision (if ` + "`" + `wallCrashEndsGame=true` + "`" + `), battery depletion, or victory.
Check ` + "`" + `stoppedReason` + "`" + ` / ` + "`" + `stopReasonCode` + "`" + ` to understand why execution halted.

### 5a. Execute a long route — reset + chained bulkMoves in one request

GraphQL allows multiple named operations (aliases) in a single mutation. Use this to reset and execute a full route in one round trip:

` + "```" + `graphql
mutation {
  reset(sessionID: "abcd") { battery score }

  c1: bulkMove(sessionID: "abcd", moves: [LEFT,LEFT,UP,UP,RIGHT,RIGHT,RIGHT,UP,UP]) {
    movesExecuted success stoppedReason
    gameState { playerPos { x y } battery victory gameOver }
  }

  c2: bulkMove(sessionID: "abcd", moves: [RIGHT,RIGHT,DOWN,DOWN,LEFT,LEFT,LEFT,DOWN]) {
    movesExecuted success stoppedReason
    gameState { playerPos { x y } battery victory gameOver }
  }
}
` + "```" + `

Each ` + "`" + `bulkMove` + "`" + ` alias (c1, c2, c3 …) resumes from where the previous left off. Pack ~50 moves per chunk. Check ` + "`" + `stoppedReason` + "`" + ` on each — if ` + "`" + `"wall"` + "`" + ` or ` + "`" + `"battery"` + "`" + `, replan from that chunk's ` + "`" + `gameState` + "`" + `.

### 6. Reset session

` + "```" + `graphql
mutation {
  reset(sessionID: "abcd") {
    playerPos { x y }
    battery
    score
    gameOver
    victory
  }
}
` + "```" + `

---

## Grid Cell Types

| char | type         | passable | effect                    |
|------|--------------|----------|---------------------------|
| R    | road         | yes      | none                      |
| H    | home         | yes      | charges battery to max    |
| S    | supercharger | yes      | charges battery to max    |
| P    | park         | yes      | collect to score / win    |
| B    | building     | no       | impassable obstacle       |
| W    | water        | no       | impassable obstacle       |

Grid is returned as ` + "`" + `grid: [[Cell]]` + "`" + ` — row-major, ` + "`" + `grid[y][x]` + "`" + `.
` + "`" + `Cell.type` + "`" + ` uses the names above. ` + "`" + `Cell.id` + "`" + ` is a coordinate string.

**Warning:** R and B look similar in monospace. Parse character-by-character before assuming a row is blocked.

---

## Winning

Collect every park (P). ` + "`" + `victory: true` + "`" + ` appears in ` + "`" + `gameState` + "`" + ` once all parks are visited.
Track progress via ` + "`" + `visitedParks` + "`" + ` — compare visited count to total P cells in the grid.

---

## Battery

- Each move costs 1 battery.
- Stepping on H or S restores battery to ` + "`" + `maxBattery` + "`" + `.
- Reaching ` + "`" + `battery: 0` + "`" + ` ends the game (` + "`" + `gameOver: true` + "`" + `, ` + "`" + `gameOverCode: "battery"` + "`" + `).
- Plan routes through charging cells. Check ` + "`" + `batteryRisk` + "`" + ` before long moves.

---

## Session Management

` + "```" + `graphql
# List all sessions
query {
  sessions {
    count
    sessions { id configName lastAccessedAt gameState { victory gameOver score } }
  }
}

# Delete a session
mutation {
  deleteSession(id: "abcd") { message }
}
` + "```" + `

---

## Move History

` + "```" + `graphql
query {
  history(sessionID: "abcd", page: 1, limit: 20, order: DESC) {
    totalMoves
    totalPages
    hasNext
    moves {
      moveNumber action
      fromPosition { x y }
      toPosition { x y }
      battery success timestamp
    }
  }
}
` + "```" + `

---

## Strategy Tips for LLMs

1. Call ` + "`" + `gameState` + "`" + ` first — read ` + "`" + `localView3x3` + "`" + ` for immediate surroundings, full ` + "`" + `grid` + "`" + ` for planning.
2. Identify H/S cells near your route before venturing far from the start.
3. Use ` + "`" + `bulkMove` + "`" + ` for known-safe corridors; use single ` + "`" + `move` + "`" + ` when navigating around obstacles.
4. After ` + "`" + `bulkMove` + "`" + `, check ` + "`" + `stopReasonCode` + "`" + ` — if ` + "`" + `"wall"` + "`" + ` or ` + "`" + `"battery"` + "`" + `, update your map and replan.
5. Never assume a row is fully blocked — verify each cell character individually (R≠B≠W).
6. Recharge proactively. Keep at least 3 battery buffer relative to distance to nearest H/S.
`))

// getConfigDirDefault returns the default configuration directory.
// It first honors the CONFIG_DIR environment variable, then falls back to "configs".
func getConfigDirDefault() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}
	return "configs"
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

	// GraphQL is the public game API.
	graphqlResolver := graph.NewResolver(gameService, hub)
	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver}))
	gqlSrv.Use(extension.Introspection{})
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(&transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.DefaultUpgrader(),
	})
	mainRouter.Handle("/graphql", gqlSrv)
	mainRouter.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))

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
