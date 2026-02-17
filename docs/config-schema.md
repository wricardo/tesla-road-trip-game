# Tesla Road Trip Game - Configuration Schema Documentation

## Overview

Game configurations are JSON files that define the game's layout, difficulty, and behavior. All config files must conform to the schema defined in `config-schema.json`.

## GameConfig Go Structure

```go
type GameConfig struct {
    Name              string            `json:"name"`
    Description       string            `json:"description"`
    GridSize          int               `json:"grid_size"`
    MaxBattery        int               `json:"max_battery"`
    StartingBattery   int               `json:"starting_battery"`
    Layout            []string          `json:"layout"`
    Legend            map[string]string `json:"legend"`
    WallCrashEndsGame bool              `json:"wall_crash_ends_game"`
    Messages          struct {
        Welcome            string `json:"welcome"`
        HomeCharge         string `json:"home_charge"`
        SuperchargerCharge string `json:"supercharger_charge"`
        ParkVisited        string `json:"park_visited"`
        ParkAlreadyVisited string `json:"park_already_visited"`
        Victory            string `json:"victory"`
        OutOfBattery       string `json:"out_of_battery"`
        Stranded           string `json:"stranded"`
        CantMove           string `json:"cant_move"`
        BatteryStatus      string `json:"battery_status"`
        HitWall            string `json:"hit_wall"`
    } `json:"messages"`
}
```

## Field Specifications

### Core Fields (Required)

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `name` | string | 1-100 chars | Configuration name |
| `description` | string | 1-500 chars | Mode description |
| `grid_size` | integer | 5-50 | Square grid dimension |
| `max_battery` | integer | 1-100 | Maximum battery capacity |
| `starting_battery` | integer | 1-max_battery | Initial battery level |
| `layout` | string[] | Must match grid_size | Grid layout rows |
| `legend` | object | Fixed mapping | Character to cell type mapping |
| `messages` | object | All required | Game event messages |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `wall_crash_ends_game` | boolean | false | Whether hitting walls ends game |

## Layout Characters

Each character in the layout array represents a cell type:

- `R` - Road (traversable)
- `H` - Home (charging station, starting position)
- `P` - Park (collectible objective)
- `S` - Supercharger (charging station)
- `W` - Water (obstacle)
- `B` - Building (obstacle)

## Validation Rules

### Structure Validation

1. **Grid Consistency**: Layout array length must equal `grid_size`
2. **Row Consistency**: Each layout string length must equal `grid_size`
3. **Battery Logic**: `starting_battery` ≤ `max_battery`
4. **Character Validity**: Only R, H, P, S, W, B allowed in layout
5. **Essential Cells**: At least one H (home) and one P (park) required

### Message Format Validation

Messages with format specifiers must include:
- `park_visited`: Must contain `%d` for score
- `victory`: Must contain `%d` for park count
- `battery_status`: Must contain `%d/%d` for current/max battery

### Conditional Validation

- If `wall_crash_ends_game` is `true`, `messages.hit_wall` is required

## Example Configuration

```json
{
  "name": "Challenge Mode",
  "description": "Difficult layout with limited battery",
  "grid_size": 10,
  "max_battery": 8,
  "starting_battery": 5,
  "layout": [
    "BBBBBBBBBB",
    "BRRRSRRRRB",
    "BRPRRRRPRB",
    "BRRRHHRRR",
    "BRRHHHHHRB",
    "BRRHHHRRB",
    "BRRRRRRRB",
    "BRPRRRRPRB",
    "BRRRSRRRRB",
    "BBBBBBBBBB"
  ],
  "legend": {
    "R": "road",
    "H": "home",
    "P": "park",
    "S": "supercharger",
    "W": "water",
    "B": "building"
  },
  "wall_crash_ends_game": true,
  "messages": {
    "welcome": "Welcome to Challenge Mode!",
    "home_charge": "Battery recharged at home!",
    "supercharger_charge": "Supercharged!",
    "park_visited": "Park visited! Score: %d",
    "park_already_visited": "Already visited",
    "victory": "Victory! All %d parks visited!",
    "out_of_battery": "Battery depleted!",
    "stranded": "Stranded!",
    "cant_move": "Can't move there!",
    "battery_status": "Battery: %d/%d",
    "hit_wall": "Crashed into wall! Game Over!"
  }
}
```

## Validation in Go

The game server validates configurations on load using `validateGameConfig()`:

```go
config, err := loadGameConfig("configs/myconfig.json")
if err != nil {
    // Validation failed - err contains specific issue
    log.Fatal(err)
}
```

Common validation errors:
- `config validation: grid_size must be between 5 and 50, got 60`
- `config validation: layout must contain at least one home (H) cell`
- `config validation: messages.hit_wall is required when wall_crash_ends_game is true`

## JSON Schema Validation

For external validation, use the JSON schema with any JSON Schema validator:

```bash
# Using ajv-cli (npm install -g ajv-cli)
ajv validate -s config-schema.json -d configs/myconfig.json

# Using Python jsonschema
python -m jsonschema -i configs/myconfig.json config-schema.json
```

## Creating New Configurations

1. Copy an existing config as template
2. Modify grid layout maintaining size consistency
3. Ensure at least one H and one P cell
4. Set appropriate battery limits for difficulty
5. Customize messages (remember format specifiers)
6. Set `wall_crash_ends_game` based on difficulty
7. Test with the game server to verify validation passes