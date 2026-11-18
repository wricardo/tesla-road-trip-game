# Tesla Road Trip Game — development tasks.
# Run `just --list` (or plain `just`) for the full list.

set shell := ["bash", "-c"]

# List available recipes.
default:
    @just --list

# Install Go modules and frontend npm packages (run once after clone).
[group('setup')]
setup:
    go mod download
    npm --prefix frontend ci

# Install developer tools (goimports, golangci-lint).
[group('setup')]
tools:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Installing developer tools (goimports, golangci-lint)..."
    command -v goimports >/dev/null 2>&1 || GO111MODULE=on go install golang.org/x/tools/cmd/goimports@latest
    command -v golangci-lint >/dev/null 2>&1 || GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Download and tidy Go dependencies.
[group('setup')]
deps:
    echo "Downloading and tidying dependencies..."
    go mod download
    go mod tidy

# Build the game server binary.
[group('build')]
build:
    echo "Building Tesla Road Trip Game server..."
    go build -o tesla-road-trip .

# Build the frontend static bundle into frontend/build.
[group('build')]
build-frontend:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building frontend static bundle..."
    if [ ! -d frontend/node_modules ]; then
        (cd frontend && npm ci)
    fi
    cd frontend && npm run build

# Build the terminal UI client binary.
[group('build')]
build-tui:
    echo "Building TUI client..."
    go build -o tesla-road-trip-tui ./cmd/tui

# Build and run the TUI client against http://localhost:8000.
[group('build')]
tui: build-tui
    echo "Starting TUI (connect to http://localhost:8000)..."
    ./tesla-road-trip-tui

# Build frontend + server and run the server on :8000.
[group('run')]
run: build-frontend build
    #!/usr/bin/env bash
    set -euo pipefail
    just stop-port 8000
    echo "Starting Tesla Road Trip Game server..."
    exec ./tesla-road-trip

# Run the backend on :8000 (no frontend rebuild).
[group('run')]
dev: build
    echo "Starting development server (Ctrl+C to stop)..."
    ./tesla-road-trip -port 8000

# Kill the local :8000 listener, rebuild, and restart in the background.
[group('run')]
restart: build-frontend build
    #!/usr/bin/env bash
    set -euo pipefail
    just stop-port 8000
    echo "Starting Tesla Road Trip Game server..."
    nohup ./tesla-road-trip -port 8000 > /tmp/tesla-road-trip.log 2>&1 &
    echo $! > .server.pid
    echo "Started tesla-road-trip PID $(cat .server.pid)"
    echo "Log: /tmp/tesla-road-trip.log"
    sleep 1
    lsof -i :8000 -sTCP:LISTEN -n -P || true

# Run the backend on :9090 for frontend live dev.
[group('run')]
dev-backend:
    echo "Starting backend on http://localhost:9090 (Ctrl+C to stop)..."
    go run . -port 9090

# Run the Svelte frontend with live reload on :5173.
[group('run')]
frontend-dev:
    echo "Starting frontend on http://localhost:5173 (backend: http://localhost:9090)..."
    cd frontend && npm run dev:local

# Run backend :9090 + live frontend :5173 together.
[group('run')]
dev-live:
    #!/usr/bin/env bash
    set -euo pipefail
    for port in 9090 5173; do
        just stop-port "$port"
    done
    echo "Starting backend on :9090 and frontend on :5173..."
    trap 'kill 0' INT TERM EXIT
    go run . -port 9090 &
    cd frontend && npm run dev:local

# Run all tests.
[group('test')]
test:
    echo "Running all tests..."
    go test ./...

# Run tests with verbose output.
[group('test')]
test-verbose:
    echo "Running tests with verbose output..."
    go test -v ./...

# Run tests and write coverage.html.
[group('test')]
test-coverage:
    echo "Running tests with coverage..."
    go test -cover ./...
    echo ""
    echo "Detailed coverage report:"
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report saved to coverage.html"

# Validate all game configurations.
[group('test')]
validate: build
    echo "Validating game configurations..."
    cd validate && go run .

# Real-browser UI checks in Chrome against a throwaway server.
[group('test')]
verify-ui headed='0': build-frontend build
    #!/usr/bin/env bash
    set -euo pipefail
    port="${VERIFY_UI_PORT:-8765}"
    dir="$(mktemp -d)"
    cleanup() {
        kill "${srv:-}" 2>/dev/null || true
        rm -rf "$dir"
    }
    trap cleanup EXIT INT TERM
    SESSIONS_DIR="$dir/sessions" ./tesla-road-trip -port "$port" -host 127.0.0.1 >"$dir/server.log" 2>&1 &
    srv=$!
    ready=""
    for _ in $(seq 1 80); do
        if curl -fsS -m 2 -o /dev/null -X POST "http://127.0.0.1:$port/graphql" \
            -H 'content-type: application/json' --data '{"query":"{maps{mapId}}"}' 2>/dev/null; then
            ready=1
            break
        fi
        sleep 0.25
    done
    if [ -z "$ready" ]; then
        echo "server did not start; see $dir/server.log"
        cat "$dir/server.log"
        exit 1
    fi
    cd frontend && BASE="http://127.0.0.1:$port" HEADED="{{ headed }}" node e2e/ui.mjs

# Format Go code (gofmt + goimports).
[group('quality')]
fmt:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Formatting Go code..."
    go fmt ./...
    if command -v goimports >/dev/null 2>&1; then
        goimports -w .
    else
        echo "goimports not found. Install with: just tools"
    fi

# Show files that need formatting.
[group('quality')]
fmt-check:
    #!/usr/bin/env bash
    set -uo pipefail
    echo "Checking formatting (gofmt)..."
    files="$(gofmt -l .)"
    if [ -n "$files" ]; then
        echo "These files need gofmt:"
        echo "$files"
        exit 1
    fi
    echo "All files are properly formatted"

# Run golangci-lint (requires golangci-lint; see `just tools`).
[group('quality')]
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running linter..."
    if command -v golangci-lint >/dev/null 2>&1; then
        golangci-lint run
    else
        echo "golangci-lint not found. Install with: just tools"
        exit 1
    fi

# Run go vet on all packages.
[group('quality')]
vet:
    echo "Running go vet..."
    go vet ./...

# Vet core packages (skips the flaky test package).
[group('quality')]
vet-safe:
    echo "Running go vet (safe subset)..."
    go vet ./api ./game/engine ./game/session ./transport/mcp ./transport/websocket ./validate ./cmd/analyze .

# Alias for vet.
[group('quality')]
vet-all: vet

# fmt-check + vet-safe + lint (fast CI).
[group('quality')]
verify: fmt-check vet-safe lint

# Start Claude with the HTTP MCP configuration (./mcp.json).
[group('clients')]
claude-game:
    echo "Starting Claude with HTTP MCP configuration..."
    claude --strict-mcp-config --mcp-config ./mcp.json

# Check local server status on :8000.
[group('utilities')]
status:
    #!/usr/bin/env bash
    set -uo pipefail
    echo "Tesla Road Trip Game - Server Status"
    echo "===================================="
    echo ""
    echo "Checking local server (port 8000):"
    if curl -s http://localhost:8000/api >/dev/null 2>&1; then
        echo "✅ Server is running on port 8000"
        echo "   Game API: http://localhost:8000/api"
        echo "   Web UI: http://localhost:8000"
        echo "   MCP endpoint: http://localhost:8000/mcp"
    else
        echo "❌ Server is not running on port 8000"
        if lsof -i :8000 >/dev/null 2>&1; then
            echo "   Port 8000 is occupied by another process:"
            lsof -i :8000
        else
            echo "   Port 8000 is available"
        fi
    fi
    echo ""
    echo "Process information:"
    if pgrep -f tesla-road-trip >/dev/null 2>&1; then
        echo "✅ tesla-road-trip processes:"
        ps aux | grep tesla-road-trip | grep -v grep | awk '{print "   PID " $2 ": " $11 " " $12 " " $13}'
    else
        echo "❌ No tesla-road-trip processes found"
    fi

# Remove build artifacts and local state files.
[group('utilities')]
clean:
    echo "Cleaning build artifacts..."
    rm -f tesla-road-trip
    rm -f coverage.out
    rm -f coverage.html
    rm -f .mcp-server.pid
    rm -f .server.pid

# Stop whatever listens on the given TCP port (used by run/restart/dev-live).
[private]
stop-port port:
    #!/usr/bin/env bash
    set -uo pipefail
    pids="$(lsof -tiTCP:"{{ port }}" -sTCP:LISTEN 2>/dev/null || true)"
    if [ -n "$pids" ]; then
        echo "Stopping port {{ port }}: $pids"
        kill $pids 2>/dev/null || true
        sleep 1
        pids="$(lsof -tiTCP:"{{ port }}" -sTCP:LISTEN 2>/dev/null || true)"
        if [ -n "$pids" ]; then
            echo "Force stopping port {{ port }}: $pids"
            kill -9 $pids 2>/dev/null || true
        fi
    fi
