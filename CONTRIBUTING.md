# Contributing to Tesla Road Trip Game

Thank you for your interest in contributing to the Tesla Road Trip Game! This document provides guidelines and information for contributors.

## 🚀 Quick Start for Contributors

### Prerequisites

- **Go 1.21 or higher**
- **Git** for version control
- **Make** for build automation (optional but recommended)

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/mcp-training.git
   cd mcp-training/statefullgame
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Verify setup**
   ```bash
   make test
   make build
   ```

4. **Start development server**
   ```bash
   make dev-watch
   ```

## 🔄 Development Workflow

### Before Making Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Ensure tests pass**
   ```bash
   make test
   ```

### Making Changes

1. **Code Quality Checks**
   ```bash
   # Format code
   make fmt

   # Run linter
   make lint

   # Static analysis
   make vet
   ```

2. **Test Your Changes**
   ```bash
   # Run all tests
   make test

   # Run tests with coverage
   make test-coverage

   # Validate configurations (if config changes)
   make validate
   ```

3. **Advanced Testing**
   ```bash
   # Run with race detection
   ./scripts/test.sh -r

   # Run benchmarks
   ./scripts/test.sh -b

   # Test specific package
   ./scripts/test.sh --package ./api
   ```

### Submitting Changes

1. **Commit with descriptive messages**
   ```bash
   git add .
   git commit -m "feat: add new game configuration validation

   - Add connectivity validation for game layouts
   - Ensure all parks are reachable from home position
   - Include comprehensive error messages for debugging"
   ```

2. **Push and create Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```

## 📝 Contribution Types

### 🐛 Bug Fixes

- **Search existing issues** before creating new ones
- **Include reproduction steps** in your bug report
- **Add tests** that demonstrate the bug and verify the fix
- **Update documentation** if the bug was related to incorrect docs

### ✨ New Features

- **Discuss major features** in an issue before implementing
- **Follow existing code patterns** and architecture
- **Include comprehensive tests** for new functionality
- **Update documentation** including README and code comments

### 🎮 Game Configurations

- **Create meaningful layouts** that are fun and challenging
- **Test thoroughly** to ensure all parks are reachable
- **Follow naming conventions**: `difficulty_theme.json`
- **Include descriptive name and description** in the JSON
- **Validate using `make validate`** before submitting

### 📚 Documentation

- **Keep examples up-to-date** with current API
- **Include code examples** for new features
- **Update architectural diagrams** if needed
- **Improve clarity** and fix typos

### 🧪 Testing

- **Increase test coverage** for untested code paths
- **Add integration tests** for complex workflows
- **Include edge case testing** for boundary conditions
- **Test performance** for scalability improvements

## 🏗️ Code Guidelines

### Go Code Style

- **Follow standard Go conventions**: Use `gofmt`, `goimports`
- **Write clear function names**: Prefer `CreateGameSession` over `CreateGS`
- **Include package documentation**: Every package should have a doc.go
- **Use meaningful variable names**: Avoid abbreviations unless common
- **Handle errors explicitly**: Always check and handle error returns

### Project Structure

- **Place files in appropriate packages**:
  - `api/` - HTTP handlers and routing
  - `game/engine/` - Core game logic
  - `game/service/` - Business logic layer
  - `game/session/` - Session management
  - `transport/` - Communication protocols

- **Follow naming conventions**:
  - Files: `snake_case.go` or `package_name.go`
  - Types: `PascalCase`
  - Functions: `PascalCase` (exported) or `camelCase` (private)
  - Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`

### Testing Guidelines

- **Test file naming**: `*_test.go` alongside source files
- **Table-driven tests** for multiple test cases
- **Descriptive test names**: `TestCreateSession_WithInvalidConfig`
- **Mock external dependencies** for unit tests
- **Include setup and teardown** in test helpers

```go
func TestGameEngine_Move_Success(t *testing.T) {
    tests := []struct {
        name      string
        direction string
        expected  bool
    }{
        {"move right on road", "right", true},
        {"move up on road", "up", true},
        {"move into wall", "down", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := setupTestEngine(t)
            result := engine.Move(tt.direction)
            if result != tt.expected {
                t.Errorf("Move() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### API Design Guidelines

- **RESTful principles**: Use appropriate HTTP methods and status codes
- **Consistent response format**: Standard JSON structure across endpoints
- **Session-aware endpoints**: Support `sessionId` parameter for multi-session
- **Error handling**: Return meaningful error messages with proper status codes
- **Versioning consideration**: Design APIs that can evolve

### Configuration Guidelines

- **Valid JSON format**: Use proper JSON syntax and validation
- **Reasonable difficulty**: Test configurations are completable
- **Connectivity validation**: Ensure all parks are reachable
- **Descriptive metadata**: Include meaningful name and description
- **Performance consideration**: Avoid excessively large grids

Example configuration:
```json
{
  "name": "Forest Adventure",
  "description": "Navigate through forest paths to visit scenic parks",
  "grid_size": 12,
  "max_battery": 18,
  "starting_battery": 15,
  "layout": [
    "BBBBBBBBBBBB",
    "BHRRRRRRRRRB",
    "BRWWWWWWWPRB",
    "..."
  ],
  "legend": {
    "R": "road",
    "H": "home",
    "P": "park",
    "S": "supercharger",
    "W": "water",
    "B": "building"
  },
  "wall_crash_ends_game": false,
  "messages": {
    "welcome": "Welcome to the forest adventure!",
    "victory": "You've explored all the scenic spots!",
    "out_of_battery": "Your Tesla ran out of power in the wilderness!",
    "supercharger_charge": "Charging at the forest supercharger station",
    "home_charge": "Safely charging at your forest cabin",
    "park_visited": "You discovered a beautiful forest park!",
    "battery_status": "Battery: %d/%d units remaining",
    "cant_move": "Path blocked by dense forest"
  }
}
```

## 🧪 Testing Your Contributions

### Required Tests

Before submitting, ensure:

1. **All existing tests pass**
   ```bash
   make test
   ```

2. **No linting errors**
   ```bash
   make lint
   ```

3. **Code is properly formatted**
   ```bash
   make fmt
   ```

4. **Configurations are valid** (if applicable)
   ```bash
   make validate
   ```

### Test Coverage

- **Maintain or improve** overall test coverage
- **Add tests for new code** - aim for >80% coverage
- **Include edge cases** and error conditions
- **Test concurrent scenarios** for session management

### Integration Testing

For significant changes, test:

1. **Multi-session scenarios**
   ```bash
   # Create multiple sessions and verify isolation
   curl -X POST http://localhost:8080/api/sessions
   curl -X POST http://localhost:8080/api/sessions -d '{"config_name":"easy"}'
   ```

2. **WebSocket functionality**
   ```bash
   # Connect to WebSocket and verify updates
   websocat ws://localhost:8080/ws
   ```

3. **MCP integration**
   ```bash
   # Test MCP tools if MCP-related changes
   make claude-game-stdin
   ```

## 📋 Pull Request Checklist

Before submitting your PR, ensure:

- [ ] **Tests pass**: `make test` succeeds
- [ ] **Code is formatted**: `make fmt` applied
- [ ] **No lint errors**: `make lint` passes
- [ ] **Documentation updated**: README and code comments current
- [ ] **Configurations valid**: `make validate` passes (if applicable)
- [ ] **Commit messages clear**: Descriptive commit messages
- [ ] **PR description complete**: Explains what, why, and how

### Pull Request Template

When creating a PR, include:

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (fix or feature causing existing functionality to change)
- [ ] Documentation update
- [ ] Configuration addition

## Testing
- [ ] Added tests for new functionality
- [ ] All existing tests pass
- [ ] Manual testing completed

## Configuration Changes (if applicable)
- [ ] New configurations validated
- [ ] All parks reachable from home
- [ ] Reasonable difficulty level

## Documentation Updates
- [ ] README updated
- [ ] Code comments added/updated
- [ ] API documentation current
```

## 🎯 Areas Needing Contribution

### High Priority

1. **Performance Optimization**
   - Optimize move processing for large grids
   - Improve WebSocket broadcasting efficiency
   - Add connection pooling for high concurrency

2. **Enhanced Testing**
   - Increase test coverage for edge cases
   - Add performance benchmarks
   - Integration tests for full workflows

3. **Configuration Improvements**
   - More diverse game layouts
   - Difficulty balancing
   - Theme-based configurations

### Medium Priority

1. **API Enhancements**
   - Batch operations for multiple sessions
   - Advanced filtering and sorting
   - Rate limiting and authentication

2. **Developer Experience**
   - Better error messages
   - Enhanced debugging tools
   - Development environment improvements

3. **Documentation**
   - Video tutorials for setup
   - Architecture deep dives
   - API usage examples

### Nice to Have

1. **New Features**
   - Tournament mode with multiple players
   - Replay system for moves
   - Statistics and analytics

2. **Platform Support**
   - Docker containerization
   - Kubernetes deployment configs
   - Cloud deployment guides

## 🆘 Getting Help

### Communication Channels

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Pull Request Comments**: For code-specific questions

### Documentation Resources

- **README.md**: Overview and quick start
- **CLAUDE.md**: Project-specific AI assistance guidelines
- **Code Comments**: Inline documentation
- **Test Files**: Usage examples and expected behavior

### Development Resources

- **Makefile**: Available development commands
- **Scripts**: Development and testing automation
- **Configs**: Example game configurations
- **Tests**: Comprehensive test examples

## 🙏 Recognition

Contributors will be recognized in:

- **README acknowledgments**
- **Release notes** for significant contributions
- **GitHub contributor graphs**
- **Special thanks** in documentation

Thank you for contributing to Tesla Road Trip Game! 🎮⚡🌳