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
- **🤖 AI-Friendly GraphQL**: Introspection-enabled API, Playground, and copyable LLM prompts
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
./tesla-road-trip

# Custom port
./tesla-road-trip -port 9090

# Enable ngrok tunnel for public access
./tesla-road-trip --ngrok

# With ngrok auth token (or set NGROK_AUTHTOKEN env var)
./tesla-road-trip --ngrok --ngrok-auth "your-auth-token"

# With custom ngrok domain
./tesla-road-trip --ngrok --ngrok-domain "your-domain.ngrok-free.app"
```

#### Server Options

- `-port`: HTTP server port (default: 8000)
- `-host`: HTTP server host (default: localhost)
- `-config-dir`: Directory containing game configurations (default: maps)
- `-debug`: Enable debug logging
- `-ngrok`: Enable ngrok tunnel for public access
- `-ngrok-auth`: Ngrok auth token (alternatively use NGROK_AUTHTOKEN env var)
- `-ngrok-domain`: Custom ngrok domain (optional)

#### Ngrok Integration

The server includes built-in ngrok support for exposing your local game server to the internet:

```bash
# Basic ngrok usage (requires NGROK_AUTHTOKEN env var or --ngrok-auth flag)
export NGROK_AUTHTOKEN="your-auth-token"
./tesla-road-trip --ngrok

# Or use a .env file (recommended)
cp .env.example .env
# Edit .env with your credentials
./tesla-road-trip  # Automatically loads .env file

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
- GraphQL access from remote AI assistants

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

## 🕹️ Clients

- **Web UI**: served by the Go server from the SvelteKit build at `/`.
- **TUI**: run the terminal client from `cmd/tui`.
- **Godot desktop**: open `godot/project.godot` in Godot 4.2+ to run a native desktop client. It is a thin GraphQL HTTP client; all authoritative game rules stay on the server. See `godot/README.md`.

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
Subscription WebSocket endpoint: `ws://localhost:8080/graphql`  
LLM quick guide: `http://localhost:8080/llms.txt`

List maps:

```graphql
query {
  maps { mapId name description gridSize maxBattery }
}
```

Create a session:

```graphql
mutation {
  createSession(mapID: "easy") {
    id
    mapName
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
    gameState { playerPos { x y } battery score victory gameOver }
  }
}
```

Subscribe to session updates:

```graphql
subscription Watch($sessionID: ID!) {
  sessionUpdated(sessionID: $sessionID) {
    playerPos { x y }
    battery
    score
    victory
    gameOver
  }
}
```

See [docs/graphql.md](docs/graphql.md) for the full API reference, examples, result types, and map input schema.

### Real-time Updates

Use GraphQL subscriptions on `ws://localhost:8080/graphql` for new clients. The server also keeps a legacy UI WebSocket route at `/ws?session=<session_id>`.

## 🤖 AI / Agent Integration

Use the GraphQL API for AI and agent integrations. GraphQL introspection is enabled, and `/playground` provides an interactive schema explorer. For terminal-based querying, [gqlcli](https://github.com/wricardo/gqlcli#-quick-start--using-the-cli) is an open-source GraphQL CLI that works well with this API.

> Note: Legacy MCP/stdio transports are disabled because the REST API transport was removed. Use `/graphql` for game operations.

### GraphQL Response Enhancements

The `move` mutation returns:
- `step`: compact summary of the move
  - Fields: `dir`, `from { x y }`, `to { x y }`, `tileChar`, `tileType`, `batteryBefore`, `batteryAfter`, `success`
- `attemptedTo`: present when move metadata is available for the attempted target
  - Fields: `x`, `y`, `tileChar`, `tileType`, `passable`
- `gameState` includes:
  - `localView3x3`: three short strings centered on player (T in center)
  - `batteryRisk`: human-readable battery risk label

The `bulkMove` mutation adds:
- Summary fields: `requestedMoves`, `movesExecuted`, `stoppedReason`, `stopReasonCode`, `stoppedOnMove`, `truncated`, `limit`
- Start/end snapshot: `startPos`, `endPos`, `startBattery`, `endBattery`, `scoreDelta`
- `steps`: compact per-step entries for this call only
- `attemptedTo`: failed/attempted target metadata when available
- Decision aids: `possibleMoves`, `localView3x3`, `batteryRisk`

Notes:
- `totalMoves` is the GraphQL field name for cumulative session moves.
- Bulk responses expose both `requestedMoves` and `movesExecuted` so agents can detect truncation or blocked routes.

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
- **GraphQL operation coverage** and response formatting
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
# - Configurable ports and maps
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
tesla-road-trip/
├── .github/workflows/     # CI/CD pipeline configuration
├── .golangci.yml          # Linter configuration
├── Makefile              # Development automation
├── README.md             # This file
├── go.mod                # Go module definition
├── main.go               # Application entry point
├── api/                  # HTTP API handlers and routing
├── cmd/analyze/          # Configuration analysis tool
├── cmd/tui/              # Terminal UI client
├── godot/                # Godot 4 desktop client
├── maps/                 # Game configuration files (JSON)
├── docs/                 # Additional documentation
├── game/
│   ├── config/          # Configuration loading and validation
│   ├── engine/          # Core game logic and mechanics
│   ├── service/         # Game service layer and business logic
│   └── session/         # Multi-session management
├── scripts/             # Development and deployment scripts
├── static/              # Web assets and templates
├── transport/
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
- GraphQL API powered by [gqlgen](https://gqlgen.com/)
- [gqlcli](https://github.com/wricardo/gqlcli) — open-source GraphQL CLI for querying this API from the terminal
- Inspired by classic grid-based strategy games
- Tesla theme chosen for electric vehicle awareness

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/wricardo/tesla-road-trip-game/issues)
- **Discussions**: [GitHub Discussions](https://github.com/wricardo/tesla-road-trip-game/discussions)
- **Documentation**: [Project Docs](docs/)
  - [Architecture](docs/architecture.md) - System design and components
  - [AI Strategy Guide](docs/ai-strategy.md) - Techniques for AI agents
  - [GraphQL API](docs/graphql.md) - Queries, mutations, and Playground
  - [Configuration Schema](docs/config-schema.md) - Custom game configs

---

**Happy Gaming! 🎮⚡🌳**
