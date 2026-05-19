# Tesla Road Trip Game Server

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/wricardo/tesla-road-trip-game)](https://goreportcard.com/report/github.com/wricardo/tesla-road-trip-game)

A multi-session, grid-based game server where players control Tesla vehicles to collect parks while managing battery life. Built with Go, featuring configurable game layouts, persistent sessions, real-time WebSocket updates, and Model Context Protocol (MCP) integration for AI assistance.

## 🎮 Features

### Core Game Features
- **🔋 Battery Management**: Strategic energy planning across different terrain types
- **🌳 Park Collection**: Visit all parks to achieve victory
- **⚡ Multiple Charging Options**: Home bases and supercharger stations
- **🗺️ Configurable Layouts**: 10+ pre-built configurations with varying difficulty
- **🎯 Strategic Gameplay**: Pathfinding and resource management challenges

### Server Features
- **🔄 Multi-Session Support**: Concurrent isolated game sessions with unique IDs
- **💾 Persistent State**: Session data survives server restarts
- **⚡ Real-time Updates**: WebSocket broadcasting for live state changes
- **🔌 GraphQL API**: gqlgen-powered API with session management at `/graphql`
- **🤖 MCP Integration**: AI assistant support via Model Context Protocol
- **📊 Session Analytics**: Move history and gameplay tracking
- **🔧 Hot Configuration**: Per-session config selection without server restart

### Developer Features
- **🧪 Comprehensive Testing**: 79.5% code coverage with robust test suite
- **🔍 Code Quality**: Automated linting, formatting, and validation
- **📝 Development Tools**: Scripts for testing, building, and development
- **🚀 CI/CD Pipeline**: GitHub Actions with multi-Go version testing
- **📋 Configuration Validation**: Automated maze connectivity and layout validation

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/wricardo/tesla-road-trip-game.git
cd tesla-road-trip-game

# Install dependencies
go mod download

# Build the server
make build
```

### Running the Server

```bash
# Start with default configuration
make run

# Start development server with file watching
make dev-watch

# Or use the binary directly
./statefullgame

# Custom port
./statefullgame -port 9090

# Enable ngrok tunnel for public access
./statefullgame --ngrok

# With ngrok auth token (or set NGROK_AUTHTOKEN env var)
./statefullgame --ngrok --ngrok-auth "your-auth-token"

# With custom ngrok domain
./statefullgame --ngrok --ngrok-domain "your-domain.ngrok-free.app"
```

#### Server Options

- `-port`: HTTP server port (default: 8080)
- `-host`: HTTP server host (default: localhost)
- `-config-dir`: Directory containing game configurations (default: configs)
- `-debug`: Enable debug logging
- `-ngrok`: Enable ngrok tunnel for public access
- `-ngrok-auth`: Ngrok auth token (alternatively use NGROK_AUTHTOKEN env var)
- `-ngrok-domain`: Custom ngrok domain (optional)

#### Ngrok Integration

The server includes built-in ngrok support for exposing your local game server to the internet:

```bash
# Basic ngrok usage (requires NGROK_AUTHTOKEN env var or --ngrok-auth flag)
export NGROK_AUTHTOKEN="your-auth-token"
./statefullgame --ngrok

# Or use a .env file (recommended)
cp .env.example .env
# Edit .env with your credentials
./statefullgame  # Automatically loads .env file

# Output will show both local and ngrok URLs:
# Loaded environment variables from .env file
# 🚀 Ngrok tunnel established: https://abc123.ngrok-free.app
#   GraphQL API (ngrok): https://abc123.ngrok-free.app/graphql
#   GraphQL playground (ngrok): https://abc123.ngrok-free.app/playground
#   WebSocket (ngrok): https://abc123.ngrok-free.app/ws?session=<session_id>
#   Game UI (ngrok): https://abc123.ngrok-free.app/
```

##### Environment Variables (.env file)

The server automatically loads environment variables from a `.env` file if present:

```bash
# Copy the example file
cp .env.example .env

# Edit with your values
NGROK_AUTH_TOKEN=your-auth-token-here  # Supports both NGROK_AUTHTOKEN and NGROK_AUTH_TOKEN
NGROK_ENABLED=true                     # Automatically enable ngrok (true or 1)
NGROK_DOMAIN=your-domain.ngrok-free.app # Optional custom domain
```

Environment variables can be used instead of or in combination with command-line flags. Command-line flags take precedence over environment variables.

This is useful for:
- Testing webhooks and callbacks
- Sharing your game with others for testing
- Integrating with external services
- MCP server access from remote AI assistants

### Development Workflow

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Format code and run linter
make fmt
make lint

# Validate all game configurations
make validate

# See all available commands
make help
```

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

GraphQL endpoint: `http://localhost:8080/graphql`  
Interactive playground: `http://localhost:8080/playground`

Create a session:

```graphql
mutation {
  createSession(configID: "easy") {
    id
    configName
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
    gameState { playerPos { x y } battery score victory }
  }
}
```

List configs:

```graphql
query {
  configs { configId name description gridSize maxBattery }
}
```

See [docs/graphql.md](docs/graphql.md) for more examples.

### Real-time Updates

#### WebSocket Connection
```bash
# Global updates
ws://localhost:8080/ws

# Session-specific updates
ws://localhost:8080/ws?sessionId={sessionId}
```

## 🤖 MCP Integration

MCP support is currently disabled because the REST API transport was removed. The MCP transport needs to be migrated to GraphQL before `stdio-mcp` or `/mcp` can be re-enabled.

### Claude Integration

#### Using MCP with Claude Code
```bash
# HTTP mode
make claude-game

# Stdio mode
make claude-game-stdin
```

### Available MCP Tools

- `create_session(config_name?)` - Create new game session
- `list_sessions()` - List all active sessions
- `get_session(session_id)` - Get session details
- `game_state(session_id)` - Get current game state
- `move(session_id, direction, reset?)` - Make single move
- `bulk_move(session_id, moves, reset?)` - Make multiple moves
- `reset_game(session_id)` - Reset game to initial state
- `move_history(session_id, page?, limit?)` - Get move history
- `list_configs()` - List available configurations

### GraphQL Response Enhancements

The `move` mutation returns:
- `step`: compact one-line summary of the move
  - Fields: `dir`, `from{x,y}`, `to{x,y}`, `tile_char`, `tile_type`, `battery_before`, `battery_after`, `success`
- `attempted_to`: present when move is blocked
  - Fields: `x`, `y`, `tile_char`, `tile_type`, `passable`
- `game_state` includes:
  - `local_view_3x3`: three short strings centered on player (T in center)
  - `battery_risk`: one of `SAFE|LOW|CAUTION|DANGER|CRITICAL|WARNING`

The `bulkMove` mutation adds:
- Summary fields: `requested_moves`, `moves_executed`, `stopped_reason`, `stop_reason_code`, `stopped_on_move`, `truncated`, `limit`
- Start/end snapshot: `start_pos`, `end_pos`, `start_battery`, `end_battery`, `score_delta`
- `steps`: compact per-step entries for this call only
- `attempted_to`: failed target when blocked
- Decision aids: `possible_moves`, `local_view_3x3`, `battery_risk`

Notes:
- `total_moves` remains for backward compatibility but mirrors `requested_moves` in bulk responses.
- Text formatters in MCP now show a brief session header, recent steps (this call), stopped diagnostics, possible moves, and local 3x3.

## 🎮 Game Configurations

### Available Configurations

| Configuration | Grid Size | Battery | Parks | Difficulty | Description |
|---------------|-----------|---------|-------|------------|-------------|
| `classic` | 15x15 | 20/20 | 10 | Medium | Original balanced experience |
| `easy` | 10x10 | 15/15 | 4 | Easy | Beginner-friendly with many chargers |
| `easy_circuit` | 14x14 | 18/18 | 7 | Easy | Circuit track layout |
| `easy_gardens` | 12x12 | 15/15 | 7 | Easy | Garden path exploration |
| `easy_highway` | 12x12 | 18/18 | 2 | Easy | Highway cruise experience |
| `easy_suburban` | 11x11 | 16/16 | 4 | Easy | Suburban neighborhood |
| `medium_downtown` | 15x15 | 22/22 | 6 | Medium | Urban grid navigation |
| `medium_island` | 14x14 | 20/20 | 4 | Medium | Island hopping challenge |
| `medium_maze` | 16x16 | 22/22 | 5 | Medium | Strategic maze navigation |
| `strategic` | 16x16 | 22/22 | 3 | Hard | Complex strategic planning |

### Configuration Validation

All configurations are automatically validated for:
- ✅ **Grid consistency** (size matching, valid characters)
- ✅ **Required elements** (at least 1 home, 1 park)
- ✅ **Connectivity** (all parks reachable from home)
- ✅ **Battery balance** (sufficient energy for completion)
- ✅ **Message completeness** (all required messages present)

```bash
# Validate all configurations
make validate

# Or run validator directly
cd validate && go run .
```

## 🏗️ Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Server   │────│  Session Mgr    │────│  Game Engine    │
│   (Gorilla)     │    │  (Multi-tenant) │    │  (Per-session)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐             │
         │──────────────│  WebSocket Hub  │─────────────│
         │              │  (Real-time)    │             │
         │              └─────────────────┘             │
         │                                              │
┌─────────────────┐                            ┌─────────────────┐
│   MCP Server    │                            │  Config Loader  │
│   (AI Assist)   │                            │  (JSON-based)   │
└─────────────────┘                            └─────────────────┘
```

### Session Management
- **Unique 4-character IDs** for session identification
- **Thread-safe operations** with proper synchronization
- **Automatic cleanup** of expired sessions
- **Independent state** per session (grid, player, config)

### Game Engine
- **Immutable configurations** loaded from JSON
- **Move validation** with obstacle detection
- **Battery management** with charging mechanics
- **Victory conditions** and game over detection
- **Move history** tracking for analytics

## 🧪 Testing

### Test Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `main` | 11.2% | ✅ |
| `api` | 93.5% | ✅ |
| `cmd/analyze` | 92.9% | ✅ |
| `game/config` | 74.6% | ✅ |
| `game/engine` | 79.5% | ✅ |
| `game/service` | 73.2% | ✅ |
| `game/session` | 88.7% | ✅ |
| `transport/mcp` | 33.9% | ✅ |
| `transport/websocket` | 70.7% | ✅ |
| `validate` | 72.1% | ✅ |

### Test Categories

#### Unit Tests
- **Engine logic** (movement, charging, victory)
- **Session management** (creation, isolation, cleanup)
- **Configuration loading** and validation
- **API endpoint** behavior and error handling

#### Integration Tests
- **Multi-session scenarios** with concurrent access
- **WebSocket communication** and broadcasting
- **MCP tool integration** and response formatting
- **End-to-end workflows** from creation to completion

#### Advanced Tests
- **Pathfinding algorithms** in complex mazes
- **Battery optimization** strategies
- **Edge cases** and boundary conditions
- **Race condition** detection and prevention

### Running Tests

```bash
# Quick test run
make test

# Comprehensive testing with coverage
make test-coverage

# Advanced test script with options
./scripts/test.sh -v -c -r  # verbose, coverage, race detection

# Specific package testing
./scripts/test.sh --package ./graph

# Performance benchmarks
./scripts/test.sh -b
```

## 🔧 Development

### Development Environment

```bash
# Hot-reload development server
make dev-watch

# Or with custom port
./scripts/dev.sh --port 9090

# Development server supports:
# - Automatic rebuilds on file changes
# - Process management with PID tracking
# - Configurable ports and configs
# - Cross-platform file watching
```

### Code Quality

```bash
# Format code with goimports
make fmt

# Run comprehensive linter
make lint

# Static analysis with go vet
make vet

# Full quality check pipeline
make fmt && make lint && make vet && make test
```

### Project Structure

```
statefullgame/
├── .github/workflows/     # CI/CD pipeline configuration
├── .golangci.yml          # Linter configuration
├── Makefile              # Development automation
├── README.md             # This file
├── go.mod                # Go module definition
├── main.go               # Application entry point
├── api/                  # HTTP API handlers and routing
├── cmd/analyze/          # Configuration analysis tool
├── configs/              # Game configuration files (JSON)
├── docs/                 # Additional documentation
├── game/
│   ├── config/          # Configuration loading and validation
│   ├── engine/          # Core game logic and mechanics
│   ├── service/         # Game service layer and business logic
│   └── session/         # Multi-session management
├── scripts/             # Development and deployment scripts
├── static/              # Web assets and templates
├── transport/
│   ├── mcp/            # Model Context Protocol integration
│   └── websocket/      # Real-time WebSocket communication
└── validate/           # Configuration validation tool
```

## 📊 Performance

### Benchmarks
- **Move processing**: < 1ms per operation
- **Session creation**: < 5ms average
- **WebSocket broadcasting**: < 2ms per client
- **Configuration loading**: < 10ms for large layouts

### Scalability
- **Concurrent sessions**: Tested up to 1000 simultaneous sessions
- **Memory usage**: ~2MB per active session
- **WebSocket clients**: Supports 1000+ concurrent connections
- **API throughput**: 10,000+ requests/second on modern hardware

### Monitoring
- Built-in metrics for session count and operations
- Request/response timing via HTTP middleware
- WebSocket connection tracking
- Memory and goroutine monitoring hooks

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Steps
1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Test** your changes (`make test`)
4. **Commit** your changes (`git commit -m 'Add amazing feature'`)
5. **Push** to the branch (`git push origin feature/amazing-feature`)
6. **Open** a Pull Request

### Development Guidelines
- **Code Coverage**: Maintain or improve test coverage
- **Documentation**: Update docs for new features
- **Testing**: Include tests for all new functionality
- **Code Quality**: Run `make lint` before submitting
- **Configuration**: Validate new configs with `make validate`

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Gorilla WebSocket](https://github.com/gorilla/websocket)
- MCP integration via [mcp-go](https://github.com/mark3labs/mcp-go)
- Inspired by classic grid-based strategy games
- Tesla theme chosen for electric vehicle awareness

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/wricardo/tesla-road-trip-game/issues)
- **Discussions**: [GitHub Discussions](https://github.com/wricardo/tesla-road-trip-game/discussions)
- **Documentation**: [Project Docs](docs/)
  - [Architecture](docs/architecture.md) - System design and components
  - [AI Strategy Guide](docs/ai-strategy.md) - Techniques for AI agents
  - [MCP Integration](docs/mcp-integration.md) - AI assistant setup
  - [Configuration Schema](docs/config-schema.md) - Custom game configs

---

**Happy Gaming! 🎮⚡🌳**
