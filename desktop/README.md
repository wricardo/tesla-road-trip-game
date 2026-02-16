# Tesla Road Trip - Multi-Car Desktop Client

Desktop client displaying **multiple Tesla cars (sessions) racing on the same grid simultaneously**.

## Features

- **Multi-car single-grid view** - Up to 9 cars on one map
- **Real-time WebSocket updates** - Instant sync for all cars
- **Color-coded cars** - Each car has unique color (Red, Blue, Green, Yellow, etc.)
- **Session switching** - Switch control between cars with number keys (1-9)
- **Dynamic car creation** - Press N to add new cars on the fly
- **Per-car stats** - Battery, moves, score for each car in header
- **Same map guarantee** - All cars share same config/map
- **Active car highlight** - >>> marker shows which car you're controlling

## Build

```bash
go build -o tesla-desktop
```

## Usage

### Start with 1 Car (Auto-Create)
```bash
./tesla-desktop
```

### Start with Multiple Cars (Auto-Create)
```bash
# Create 3 cars on startup
./tesla-desktop "" "" ""

# Create 5 cars
./tesla-desktop "" "" "" "" ""
```

### Connect to Existing Sessions
```bash
# Single car
./tesla-desktop abc1

# Multiple existing cars on same map
./tesla-desktop abc1 xyz2 def3
```

**Important**: All sessions must use the same config/map. New cars automatically use the same config as the first car.

## Controls

### Car Selection
- **1-9** - Switch active car (number keys)
- **N** - Add new car with same map (max 9 total)

### Active Car Control
- **Arrow Keys / WASD** - Move the active car
- **R** - Reset active car

## Screen Layout

```
┌────────────────────────────────────────────────────┐
│ HEADER - All Car Stats                             │
│ >>> [1] abc1 [WS] BAT:18/20 MV:45 SC:3            │ ← Active (Red)
│     [2] xyz2 [WS] BAT:12/20 MV:32 SC:1            │ ← Blue
│     [3] def3 [WS] BAT:20/20 MV:15 SC:0            │ ← Green
├────────────────────────────────────────────────────┤
│                                                    │
│         SHARED GAME GRID                          │
│                                                    │
│    All cars visible with:                         │
│    - Unique colors                                │
│    - Session numbers on cars                      │
│    - Same map/obstacles                           │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Car Colors

1. Red
2. Blue
3. Green
4. Yellow
5. Magenta
6. Cyan
7. Orange
8. Purple
9. Pink

## Header Info

Each car shows:
- **>>>** - Active car indicator
- **[#]** - Car number
- **SessionID** - Unique session identifier
- **[WS/POLL]** - Connection status
- **BAT** - Battery level
- **MV** - Total moves
- **SC** - Score
- **VICTORY!** or **GAME OVER** - End state

## Grid Legend

- **Gray** - Roads
- **Green** - Home/Garage - charges battery
- **Red supercharger** - Charges battery
- **Blue** - Water (blocked)
- **Brown** - Buildings (blocked)
- **Orange** - Parks (uncollected)
- **Orange with numbers** - Parks showing which cars collected them (e.g., "13" = cars 1 and 3)
- **Colored squares with numbers** - Cars (1-9)

## Requirements

- Tesla game server running on `http://localhost:8080`
- Go 1.21+ with Ebiten v2
- All sessions must use same config

## Example: Racing 3 Cars

```bash
# Start game with 3 cars
./tesla-desktop "" "" ""

# Game shows:
# - 3 colored cars on same grid
# - Header with stats for all 3
# - Control car #1 (red) by default
# - Press 2 to switch to car #2 (blue)
# - Press 3 to switch to car #3 (green)
# - All cars update in real-time via WebSocket
```

## Park Collection Display

When a car collects a park, that park shows the car number:
- **Park with "1"** - Car #1 collected this park
- **Park with "23"** - Cars #2 and #3 both collected this park
- **Park with no number** - Uncollected by all cars

## Reset Behavior

When you press **R** to reset:
- **Only the active car resets** - Position, battery, and collected parks
- **Other cars continue** - Their progress is NOT affected
- **Park display updates** - Numbers removed for reset car only

Example: If car #1 resets, parks showing "13" become "3" (only car #3 now)

## Tips

- **Race yourself**: Create multiple cars and see which strategy wins
- **Battery management**: Watch all car batteries - low battery = dead car
- **Strategic switching**: Switch cars to optimize parallel exploration
- **Same map only**: All cars must be on same config - enforced automatically
- **Park tracking**: Numbers on parks show which cars collected them
