// Command tesla-road-trip starts the Tesla Road Trip Game server.
//
// It supports two modes:
//  1. "server" (default) – runs the HTTP server exposing GraphQL, WebSocket, and an /mcp HTTP endpoint
//  2. "stdio-mcp" – runs an MCP stdio server and spins up an internal HTTP API if none is available
//
// Flags control host/port, config directory, debug logging, version output,
// and public URL rendering for /llms.txt.
package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/99designs/gqlgen/graphql"
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
)

// Version information
const (
	Version = "2.0.0"
	AppName = "Tesla Road Trip Game Server"
)

// Configuration flags control how the server starts and which services are enabled.
var (
	port        = flag.Int("port", 8000, "HTTP server port")
	host        = flag.String("host", "localhost", "HTTP server host")
	configDir   = flag.String("config-dir", getConfigDirDefault(), "Directory containing game configurations")
	sessionsDir = flag.String("sessions-dir", getSessionsDirDefault(), "Directory for persisted game sessions")
	debug       = flag.Bool("debug", false, "Enable debug logging")
	version     = flag.Bool("version", false, "Show version information")
	publicURL   = flag.String("public-url", "", "Public base URL served in llms.txt (e.g. https://myserver.com). Defaults to the request's scheme and Host")
)

//go:embed llms.txt.tmpl
var llmsTxtSource string

// llmsTxtTemplate renders /llms.txt; edit llms.txt.tmpl, not this file.
var llmsTxtTemplate = template.Must(template.New("llms").Parse(llmsTxtSource))

// getConfigDirDefault returns the default configuration directory.
// It first honors the CONFIG_DIR environment variable, then falls back to "maps".
func getConfigDirDefault() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}
	return "maps"
}

// getSessionsDirDefault returns the default session persistence directory.
// It first honors the SESSIONS_DIR environment variable, then falls back to "sessions".
func getSessionsDirDefault() string {
	if sessionsDir := os.Getenv("SESSIONS_DIR"); sessionsDir != "" {
		return sessionsDir
	}
	return "sessions"
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

// enableGraphQLOperationLogging logs every GraphQL operation on completion.
func enableGraphQLOperationLogging(gqlSrv *handler.Server) {
	gqlSrv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)

		opName := "<anonymous>"
		if opCtx.OperationName != "" {
			opName = opCtx.OperationName
		}

		opType := "unknown"
		if opCtx.Operation != nil {
			opType = string(opCtx.Operation.Operation)
		}

		start := time.Now()

		responseHandler := next(ctx)
		return func(ctx context.Context) *graphql.Response {
			resp := responseHandler(ctx)
			errCount := 0
			if resp != nil {
				errCount = len(resp.Errors)
			}
			log.Printf(
				"[graphql] op.end name=%q type=%s duration=%s errors=%d",
				opName,
				opType,
				time.Since(start),
				errCount,
			)
			return resp
		}
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
		fmt.Fprintf(os.Stderr, "  %s                    # Run HTTP server on default port 8000\n", os.Args[0])
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
	enableGraphQLOperationLogging(gqlSrv)
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
		// graphql-transport-ws clients must answer pings; connections that miss
		// pongs for 2x this interval are closed so dead peers do not hold
		// subscriptions open until the kernel TCP timeout.
		PingPongInterval: 10 * time.Second,
		Upgrader:         websocket.DefaultUpgrader(),
	})
	mainRouter.Handle("/graphql", withHTTPRequest(gqlSrv))
	if playgroundEnabled {
		mainRouter.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))
		log.Println("GraphQL playground: enabled (set GRAPHQL_PLAYGROUND=false to disable)")
	} else {
		log.Println("GraphQL playground: disabled")
	}

	// /llms.txt — rendered from template. Base URL: -public-url if set, otherwise
	// the host the client actually used, so docs never point at a different server.
	mainRouter.HandleFunc("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		baseURL := strings.TrimRight(*publicURL, "/")
		if baseURL == "" {
			scheme := "http"
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}
			h := r.Host
			if h == "" {
				h = addr
			}
			baseURL = scheme + "://" + h
		}
		wsURL := "ws" + strings.TrimPrefix(baseURL, "http")
		var buf bytes.Buffer
		if err := llmsTxtTemplate.Execute(&buf, struct{ BaseURL, WSURL string }{BaseURL: baseURL, WSURL: wsURL}); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(buf.Bytes())
	})

	// MCP HTTP endpoint (Streamable HTTP transport).
	if mcpEnabled {
		mcpSrv := mcptransport.NewServer(gameService, hub)
		mainRouter.Handle("/mcp", mcpSrv.Handler())
		log.Println("MCP endpoint: enabled at /mcp (set MCP_ENABLED=false to disable)")
	} else {
		log.Println("MCP endpoint: disabled")
	}

	// Mount static UI and WebSocket routes at root.
	mainRouter.Handle("/", apiServer)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mainRouter,
		// Keep read timeout bounded, but allow long-running GraphQL mutations (e.g. bulkMove
		// with per-step delay) to complete without the server closing the socket mid-response.
		ReadTimeout: 15 * time.Second,
		// No write timeout to avoid EOF on valid long responses.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

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

	// Wait for shutdown signal
	sig := <-stop
	log.Printf("Received signal: %v. Shutting down...", sig)

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
	persistence, err := session.NewFilePersistence(*sessionsDir, configManager)
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
