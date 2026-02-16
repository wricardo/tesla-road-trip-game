# Tesla Road Trip Game - MCP Server

This document explains how to use the Tesla Road Trip Game MCP server, which acts as a proxy to the HTTP API server, allowing LLMs like Claude to play the game through structured MCP tool calls.

## Overview

The MCP server acts as a **proxy layer** that translates MCP tool calls into HTTP API requests. This architecture provides:

- **Single source of truth**: HTTP server maintains the game state
- **Clean separation**: MCP server handles protocol translation only  
- **Easy comparison**: Same underlying game logic for both interfaces
- **Shared state**: Both HTTP and MCP clients can interact with the same game instance

## Architecture

```
┌─────────────┐    HTTP Requests     ┌─────────────┐
│ MCP Server  │ ───────────────────► │ HTTP Server │
│ (Protocol   │                      │ (Game       │
│  Proxy)     │ ◄─────────────────── │  Engine)    │
└─────────────┘    HTTP Responses    └─────────────┘
       ▲                                     ▲
       │ MCP Tools                           │ REST API
       │                                     │ WebSocket
┌─────────────┐                      ┌─────────────┐
│   Claude    │                      │ Web Browser │
│    LLM      │                      │  curl, etc  │
└─────────────┘                      └─────────────┘
```

## Starting the Server

The game now has two simple modes:

### Option 1: HTTP Server Mode (with MCP endpoint)
```bash
# Start HTTP server with REST API, WebSocket, and MCP endpoint
./statefullgame
# or explicitly:
./statefullgame server

# Server provides:
# - REST API: http://localhost:8080/api/*
# - WebSocket: ws://localhost:8080/ws
# - MCP endpoint: http://localhost:8080/mcp
```

### Option 2: MCP Stdio Mode (self-contained)
```bash
# Run MCP server via stdio with internal HTTP server
./statefullgame stdio-mcp
# or:
./statefullgame mcp-stdio
./statefullgame mcp

# This mode:
# - Runs MCP server on stdio for Claude integration
# - Automatically starts internal HTTP server on random port
# - No separate server needed - completely self-contained
```

## Configuration

- **Port**: Use `-port` flag (default: 8080) for HTTP server mode
- **Config Directory**: Use `-config-dir` flag or `CONFIG_DIR` environment variable
- **Default Config**: Use `-config` flag to specify default configuration file

## Available MCP Tools

### Core Game Tools

1. **`game_state`** - Get current game state
   - No parameters required
   - Returns: Complete game state including grid, player position, battery, score

2. **`move`** - Move the player
   - Parameters:
     - `direction` (required): "up", "down", "left", or "right"
   - Returns: Updated game state after the move

3. **`reset_game`** - Reset game to initial state
   - No parameters required
   - Returns: Fresh game state

### Save/Load Tools

4. **`save_game`** - Save current game state
   - No parameters required
   - Returns: Save ID and confirmation message

5. **`load_game`** - Load a saved game state
   - Parameters:
     - `save_id` (required): ID of save to load (e.g., "save_1234567890")
   - Returns: Loaded game state

6. **`list_saves`** - List all available saves
   - No parameters required
   - Returns: Array of save files with timestamps

### Configuration Tools

7. **`list_configs`** - List available game configurations
   - No parameters required
   - Returns: Array of available config files with descriptions

8. **`load_config`** - Load a specific configuration
   - Parameters:
     - `config_path` (required): Path to config file (e.g., "configs/easy.json")
   - Returns: New game state with loaded configuration

### Information Tools

9. **`game_info`** - Get detailed game statistics
   - No parameters required
   - Returns: Comprehensive game information including progress, map details, and configuration

## Game Mechanics

The game mechanics are identical to the HTTP API version:

- **Objective**: Collect all parks (P) while managing Tesla battery
- **Movement**: Each move consumes 1 battery unit
- **Charging**: Home (H) and Supercharger (S) stations restore full battery
- **Victory**: Win by visiting all parks on the map
- **Grid**: Navigate around water (W) and buildings (B) which block movement

## Example MCP Tool Usage

### Basic Game Flow

1. **Check current state**: Call `game_state` to see the map and player position
2. **Plan route**: Identify parks and charging stations
3. **Move**: Use `move` tool with direction parameter
4. **Monitor progress**: Use `game_info` for statistics
5. **Save progress**: Use `save_game` for checkpoints

### Configuration Management

```json
// List available configurations
{"tool": "list_configs"}

// Load a specific configuration  
{"tool": "load_config", "parameters": {"config_path": "configs/challenge.json"}}

// Reset to fresh state
{"tool": "reset_game"}
```

### Save Management

```json
// Save current progress
{"tool": "save_game"}

// List all saves
{"tool": "list_saves"}

// Load specific save
{"tool": "load_game", "parameters": {"save_id": "save_1234567890"}}
```

## Comparison: MCP vs HTTP API

| Feature | MCP Tool Call | HTTP Request | Same Data? |
|---------|---------------|--------------|------------|
| Get state | `game_state` → HTTP proxy | `GET /api` | ✅ Identical |
| Move player | `move` → `POST /api {"action":"right"}` | `POST /api {"action":"right"}` | ✅ Identical |
| Save game | `save_game` → `POST /api {"action":"save"}` | `POST /api {"action":"save"}` | ✅ Identical |
| Load game | `load_game` → `POST /api {"action":"load",...}` | `POST /api {"action":"load",...}` | ✅ Identical |
| List saves | `list_saves` → `GET /api/saves` | `GET /api/saves` | ✅ Identical |
| Real-time updates | ❌ Not supported | ✅ WebSocket `/ws` | Different |

**Key Insight**: Since MCP tools proxy to the same HTTP endpoints, the data and game logic are **identical**. The only difference is the protocol layer (MCP tools vs direct HTTP calls).

## Integration with LLM Applications

The MCP server is designed to work with MCP-compatible LLM applications. The server:

- Provides structured schemas for all tool parameters
- Returns consistent JSON responses
- Maintains game state between tool calls
- Supports the full game feature set

## Development and Testing

For development, you can test individual tools using MCP client libraries or tools that support the MCP protocol. The server logs all operations to stderr while maintaining clean JSON communication on stdout.

## Configuration Files

The server supports the same configuration files as the HTTP server:

- `configs/classic.json` - Default balanced gameplay
- `configs/easy.json` - Larger battery, easier navigation
- `configs/challenge.json` - Harder gameplay with smaller battery

Each configuration defines the grid layout, battery limits, and game messages.

## Troubleshooting

- **Server won't start**: Check that configuration file exists and is valid JSON
- **Tool errors**: Verify parameter names and types match the schema
- **Save/load issues**: Ensure `saves/` directory exists and has write permissions
- **State inconsistencies**: Use `reset_game` to return to a known good state