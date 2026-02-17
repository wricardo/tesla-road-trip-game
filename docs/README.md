# Tesla Road Trip Game - Documentation

Comprehensive documentation for the Tesla Road Trip multi-session game server.

## 📚 Documentation Index

### Core Documentation

- **[Architecture](architecture.md)** - System design, components, and data flow
- **[MCP Integration](mcp-integration.md)** - Model Context Protocol server setup and usage
- **[Configuration Schema](config-schema.md)** - Game configuration format and validation

### AI Strategy & Development

- **[AI Strategy Guide](ai-strategy.md)** - Strategies for AI agents playing the game
  - Character recognition (critical for success)
  - Navigation algorithms
  - Battery management
  - Proven success patterns

### Planning & Design Documents

- **[Epic: Architectural Refactoring](epic-architectural-refactoring.md)** - Major refactoring epic
- **[Architecture Refactoring Diagram](architecture-refactoring-diagram.md)** - Visual architecture diagrams

### User Stories

- **[Story 1: GameService Extraction](stories/story-1-gameservice-extraction.md)**

### Historical Documents

See [archive/](archive/) for historical documentation:
- Migration guides
- Refactoring completion notes
- Integration points documentation
- Old architecture documentation

## 🚀 Quick Links

### Getting Started
- [Main README](../README.md) - Quick start, installation, and usage
- [Contributing Guide](../CONTRIBUTING.md) - How to contribute to the project
- [Claude Development Guide](../CLAUDE.md) - Development instructions for Claude Code

### API & Integration
- [MCP Integration](mcp-integration.md) - AI assistant integration
- [Configuration Schema](config-schema.md) - Creating custom game configs

### Architecture & Design
- [Architecture Overview](architecture.md) - Technical architecture
- [Refactoring Epic](epic-architectural-refactoring.md) - Major design changes

### AI Development
- [AI Strategy Guide](ai-strategy.md) - Techniques for AI agents

## 📖 Documentation Structure

```
docs/
├── README.md                               # This file
├── architecture.md                         # System architecture
├── architecture-refactoring-diagram.md     # Architecture diagrams
├── ai-strategy.md                          # AI agent strategies
├── config-schema.md                        # Configuration format
├── mcp-integration.md                      # MCP server guide
├── epic-architectural-refactoring.md       # Refactoring epic
├── stories/                                # Development stories
│   └── story-1-gameservice-extraction.md
└── archive/                                # Historical documents
    ├── ARCHITECTURE_FINAL.md
    ├── INTEGRATION_POINTS.md
    ├── REFACTORING_COMPLETE.md
    ├── migration_guide.md
    └── medium_maze_solution.md
```

## 🔧 Development Resources

### Testing
```bash
make test              # Run all tests
make test-coverage     # Run tests with coverage
make validate          # Validate game configurations
```

### Building
```bash
make build             # Build binary
make run               # Run server
make dev-watch         # Development mode with hot reload
```

### Code Quality
```bash
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet
```

## 📝 Contributing to Documentation

When adding new documentation:

1. **Place correctly**:
   - Core technical docs → `docs/`
   - Historical/archived docs → `docs/archive/`
   - User-facing docs → Root README

2. **Update this index** when adding new docs

3. **Use clear headings** and table of contents for long docs

4. **Include code examples** where applicable

5. **Link between docs** to create a web of knowledge

## 🤝 Need Help?

- Check the [Main README](../README.md) for quick start
- See [Contributing Guide](../CONTRIBUTING.md) for development setup
- Review [Architecture](architecture.md) for system design
- Read [AI Strategy](ai-strategy.md) for gameplay techniques
