#!/bin/bash

# Tesla Road Trip Game - MCP Integration Test
# This script demonstrates the HTTP server + MCP proxy architecture

echo "🎮 Tesla Road Trip Game - MCP Integration Test"
echo "=============================================="

# Check if servers are built
if [ ! -f "./statefullgame" ]; then
    echo "❌ Building statefullgame..."
    go build
    if [ $? -ne 0 ]; then
        echo "❌ Build failed"
        exit 1
    fi
fi

echo "✅ Starting HTTP server on port 8080..."
./statefullgame serve -p 8080 &
HTTP_PID=$!

# Wait a moment for server to start
sleep 2

echo "🔍 Testing HTTP server connectivity..."
if ! curl -s http://localhost:8080/api > /dev/null; then
    echo "❌ HTTP server is not responding"
    kill $HTTP_PID 2>/dev/null
    exit 1
fi

echo "✅ HTTP server is running (PID: $HTTP_PID)"

echo "📊 Current game state via HTTP API:"
curl -s http://localhost:8080/api | jq -r '.message'

echo ""
echo "🔧 MCP server configuration:"
echo "  - Proxies to: http://localhost:8080"
echo "  - Available tools: game_state, move, reset_game, save_game, load_game, list_saves, list_configs, game_info"

echo ""
echo "📋 Usage Examples:"
echo ""
echo "1. Start MCP server (in another terminal):"
echo "   ./statefullgame mcp"
echo ""
echo "2. Test with different HTTP server:"
echo "   ./statefullgame serve -p 9090  # Terminal 1"
echo "   ./statefullgame mcp -s http://localhost:9090  # Terminal 2"
echo ""
echo "3. Compare protocols:"
echo "   # HTTP API"
echo "   curl http://localhost:8080/api"
echo "   curl -X POST http://localhost:8080/api -d '{\"action\":\"right\"}'"
echo ""
echo "   # MCP Tools (via LLM client)"
echo "   game_state  → GET /api"
echo "   move(right) → POST /api {\"action\":\"right\"}"

echo ""
echo "🎯 Architecture Benefits:"
echo "  ✅ Single source of truth (HTTP server maintains state)"
echo "  ✅ Protocol independence (same game logic for both)"
echo "  ✅ Easy comparison (MCP vs HTTP for same operations)"
echo "  ✅ Shared state (both clients see same game instance)"

echo ""
echo "🧪 Integration Test Complete!"
echo "HTTP server running on PID $HTTP_PID"
echo ""
echo "To stop HTTP server: kill $HTTP_PID"
echo "To start MCP server: ./statefullgame mcp"