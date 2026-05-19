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
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
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
)

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
	mainRouter.Handle("/graphql", handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver})))
	mainRouter.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))

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
