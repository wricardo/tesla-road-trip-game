# Tesla Road Trip Game

A grid-based game where you control a Tesla car to collect parks while managing battery. Built with Go, featuring configurable game layouts and persistent save states.

## Features

- **Multiple Difficulty Levels**: Choose from Classic, Easy, or Challenge modes
- **Persistent Saves**: Game state survives server restarts
- **Real-time Updates**: WebSocket support for live game state updates
- **RESTful API**: Full control via HTTP endpoints
- **Configurable Layouts**: JSON-based game configuration system

## Quick Start

### Build the Game
```bash
go build -o statefullgame
```

### Start the Server
```bash
# Classic mode (default)
./statefullgame

# Easy mode - more superchargers, higher battery
./statefullgame -config configs/easy.json

# Challenge mode - limited charging, low battery
./statefullgame -config configs/challenge.json

# Custom port
./statefullgame -port 9090
```

## Game Rules

1. **Movement**: Each move consumes 1 battery unit
2. **Objective**: Collect all parks (🌳) to win
3. **Charging**: Recharge at home (🏠) or superchargers (⚡)
4. **Obstacles**: Cannot move through water (💧) or buildings (🏢)
5. **Game Over**: Battery runs out with no way to recharge
6. **Victory**: All parks collected

## API Documentation

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Get Game State
```bash
GET /api
```
Returns current game state including grid, player position, battery, and score.

```bash
curl http://localhost:8080/api
```

#### Perform Action
```bash
POST /api
Content-Type: application/json
```

Available actions:
- **Movement**: `up`, `down`, `left`, `right`
- **Game Management**: `reset`, `save`, `load`

Examples:
```bash
# Move player
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"right"}'

# Save game
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"save"}'

# Load saved game
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"load","save_id":"save_1757156572"}'

# Reset game
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -d '{"action":"reset"}'
```

#### List Available Configurations
```bash
GET /api/configs
```
Returns all available game configurations.

```bash
curl http://localhost:8080/api/configs
```

#### List Saved Games
```bash
GET /api/saves
```
Returns all saved game sessions.

```bash
curl http://localhost:8080/api/saves
```

### WebSocket
```
ws://localhost:8080/ws
```
Connect to receive real-time game state updates.

## Game Configurations

### Classic Mode (`configs/classic.json`)
- **Grid Size**: 15x15
- **Battery**: 10 units
- **Parks**: 10
- **Description**: The original Tesla Road Trip experience

### Easy Mode (`configs/easy.json`)
- **Grid Size**: 10x10
- **Battery**: 15 units
- **Parks**: 4
- **Description**: Beginner-friendly with more charging stations

### Challenge Mode (`configs/challenge.json`)
- **Grid Size**: 20x20
- **Battery**: 8 units (starts at 5)
- **Parks**: 12
- **Description**: Limited charging options, requires careful planning

## Directory Structure

```
statefullgame/
├── configs/              # Game configuration files
│   ├── classic.json     # Default configuration
│   ├── easy.json        # Easy mode
│   └── challenge.json   # Challenge mode
├── saves/               # Saved game sessions
│   └── save_*.json      # Individual save files
├── templates/           # HTML templates
│   └── game.html        # Web interface
├── main.go              # Main application
└── statefullgame        # Compiled binary
```

## Configuration Format

Game configurations are JSON files with the following structure:

```json
{
  "name": "Configuration Name",
  "description": "Configuration description",
  "grid_size": 15,
  "max_battery": 10,
  "starting_battery": 10,
  "layout": [
    "BBBWBBBPBBBWBBB",
    "BRRRRRRRRRRRRRB",
    ...
  ],
  "legend": {
    "R": "road",
    "H": "home",
    "P": "park",
    "S": "supercharger",
    "W": "water",
    "B": "building"
  },
  "messages": {
    "welcome": "Welcome message",
    "home_charge": "Charging at home message",
    ...
  }
}
```

### Layout Legend
- `R` - Road (driveable)
- `H` - Home (starting position, charging station)
- `P` - Park (collectible objective)
- `S` - Supercharger (charging station)
- `W` - Water (obstacle)
- `B` - Building (obstacle)

## Save System

- Saves are stored in `saves/` directory
- File format: `save_<timestamp>.json`
- Contains complete game state including:
  - Grid layout and visited parks
  - Player position and battery
  - Score and game status
  - Original configuration name
- Saves can be loaded even when running different configurations

## Development

### Requirements
- Go 1.16 or higher
- Gorilla WebSocket package

### Install Dependencies
```bash
go mod download
```

### Run Tests
```bash
./test-api.sh  # Basic API test script
```

## Game Response Format

### Game State Response
```json
{
  "grid": [...],
  "player_pos": {"x": 9, "y": 7},
  "battery": 10,
  "max_battery": 10,
  "score": 0,
  "visited_parks": {},
  "message": "Welcome!",
  "game_over": false,
  "victory": false,
  "config_name": "Classic Layout",
  "save_id": "save_1757156572",
  "last_saved": "2025-09-06T07:02:52Z"
}
```

### Save List Response
```json
[
  {
    "save_id": "save_1757156572",
    "timestamp": "2025-09-06T07:02:52Z"
  }
]
```

### Config List Response
```json
[
  {
    "filename": "classic.json",
    "name": "Classic Layout",
    "description": "The original Tesla Road Trip game"
  }
]
```

## Tips & Strategies

1. **Plan Your Route**: Look for parks near charging stations
2. **Battery Management**: Always know your nearest charging point
3. **Efficient Paths**: Minimize moves between objectives
4. **Save Often**: Use the save feature before risky moves
5. **Challenge Mode**: Start by securing a path to the nearest supercharger

## License

MIT License - Feel free to modify and distribute

## Contributing

To add new game configurations:
1. Create a new JSON file in `configs/`
2. Design your layout using the legend characters
3. Adjust battery and grid size for difficulty
4. Test thoroughly before committing

## Troubleshooting

### Server won't start
- Check if port 8080 is already in use
- Use `-port` flag to specify different port

### Can't load saves
- Ensure `saves/` directory exists
- Check file permissions
- Verify save file is valid JSON

### Game state not updating
- Check WebSocket connection
- Ensure server is running
- Try refreshing the browser