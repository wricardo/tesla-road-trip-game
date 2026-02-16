#!/bin/bash

# Tesla Road Trip Game - Development Script
# Provides hot-reload development server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Tesla Road Trip Game - Development Mode${NC}"
echo -e "${BLUE}======================================${NC}"

# Default values
PORT=8080
WATCH=true

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        --no-watch)
            WATCH=false
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -p, --port PORT     Server port (default: 8080)"
            echo "  --no-watch          Disable file watching"
            echo "  -h, --help          Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Function to build and run the server
build_and_run() {
    echo -e "${YELLOW}Building server...${NC}"
    if go build -o statefullgame .; then
        echo -e "${GREEN}✓ Build successful${NC}"

        # Kill existing server if running
        if [ -f .dev-server.pid ]; then
            OLD_PID=$(cat .dev-server.pid)
            if kill -0 $OLD_PID 2>/dev/null; then
                echo -e "${YELLOW}Stopping existing server (PID: $OLD_PID)${NC}"
                kill $OLD_PID
                sleep 1
            fi
            rm -f .dev-server.pid
        fi

        # Start new server
        echo -e "${GREEN}Starting server on port $PORT${NC}"
        ./statefullgame -port $PORT &

        SERVER_PID=$!
        echo $SERVER_PID > .dev-server.pid
        echo -e "${GREEN}✓ Server started (PID: $SERVER_PID)${NC}"
        echo -e "${BLUE}Server available at: http://localhost:$PORT${NC}"
    else
        echo -e "${RED}✗ Build failed${NC}"
    fi
}

# Function to cleanup
cleanup() {
    echo -e "\n${YELLOW}Shutting down development server...${NC}"
    if [ -f .dev-server.pid ]; then
        PID=$(cat .dev-server.pid)
        if kill -0 $PID 2>/dev/null; then
            kill $PID
            echo -e "${GREEN}✓ Server stopped${NC}"
        fi
        rm -f .dev-server.pid
    fi
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Initial build and run
build_and_run

if [ "$WATCH" = true ]; then
    echo -e "${BLUE}Watching for file changes... (Press Ctrl+C to stop)${NC}"

    # Check if fswatch is available
    if command -v fswatch >/dev/null 2>&1; then
        # Use fswatch if available (macOS)
        fswatch -o --exclude='.*\.log$' --exclude='.*\.pid$' --exclude='statefullgame$' . | while read; do
            echo -e "${YELLOW}Files changed, reloading...${NC}"
            build_and_run
        done
    elif command -v inotifywait >/dev/null 2>&1; then
        # Use inotifywait if available (Linux)
        while inotifywait -r -e modify,create,delete --exclude='.*\.(log|pid)$' .; do
            echo -e "${YELLOW}Files changed, reloading...${NC}"
            build_and_run
        done
    else
        echo -e "${YELLOW}No file watcher available. Install fswatch (macOS) or inotify-tools (Linux) for auto-reload.${NC}"
        echo -e "${BLUE}Server running without file watching. Press Ctrl+C to stop.${NC}"
        wait
    fi
else
    echo -e "${BLUE}File watching disabled. Press Ctrl+C to stop.${NC}"
    wait
fi