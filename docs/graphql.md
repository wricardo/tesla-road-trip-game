# GraphQL API

The Tesla Road Trip Game public game API is exposed by gqlgen at `/graphql`.

- Playground: `GET /playground`
- GraphQL HTTP endpoint: `POST /graphql`
- GraphQL WebSocket endpoint for subscriptions: `ws://<host>/graphql`
- Introspection: enabled
- LLM quick guide: `GET /llms.txt`

The server also mounts a legacy UI WebSocket route at `/ws?session=<session_id>`, but new GraphQL clients should use subscriptions on `/graphql`.

## Request format

Send GraphQL documents as JSON:

```bash
curl -s http://localhost:8000/graphql \
  -H 'content-type: application/json' \
  -d '{"query":"query { maps { mapId name description gridSize maxBattery } }"}'
```

## Core workflow

1. List available maps with `maps`.
2. Create a session with `createSession(mapID: "...")`.
3. Read state with `gameState(sessionID: "...")`.
4. Move with `move` or `bulkMove`.
5. Watch live updates with `sessionUpdated(sessionID: "...")`.

## Queries

| Query | Description |
| --- | --- |
| `session(id: ID!): Session!` | Load one saved session, including its state and map. |
| `sessions(sort: SessionSort = ACTION, order: SortOrder = DESC, limit: Int): SessionList!` | List saved sessions. Sort by `ACTION` (last action time, default) or `CREATED`; order by `ASC` or `DESC`. A positive `limit` truncates the returned list while `total` remains the untruncated count. |
| `unifiedSessions(mapName: String): UnifiedSessions!` | List sessions with each session's state and map bundled together. Optional `mapName` filters results. |
| `gameState(sessionID: ID!): GameState!` | Get the current state for a session. |
| `history(sessionID: ID!, page: Int = 1, limit: Int = 50, order: SortOrder = DESC): HistoryResponse!` | Page through move history. |
| `maps: [MapInfo!]!` | List available maps. |
| `map(name: String!, password: String): GameMap!` | Load a full map definition by name/id. `layout(password:)` needs the map's grid password when one is set. |

## Mutations

| Mutation | Description |
| --- | --- |
| `createSession(mapID: String, mapName: String, fogEnabled: Boolean = false, fogRadius: Int = 1, gridPassword: String, moveDelayMs: Int): Session!` | Create a new game session. `mapID` and `mapName` are aliases; pass one of them (prefer `mapID`). If neither is passed, the service default map is used. With `fogEnabled: true`, `grid` requires `gridPassword`; if you omit it, the server generates one and returns it once in `generatedGridPassword`. |
| `deleteSession(id: ID!): DeleteSessionResult!` | Delete a saved session. |
| `updateSession(id: ID!, displayName: String!): Session!` | Set a session's display name. |
| `move(sessionID: ID!, direction: Direction!, reset: Boolean = false): MoveResult!` | Execute one move. Optional `reset: true` resets the session before moving. Broadcasts a session update. |
| `bulkMove(sessionID: ID!, moves: [Direction!]!, reset: Boolean = false): BulkMoveResult!` | Execute a sequence of moves. Stops early on game-ending or invalid conditions reported in `stoppedReason` / `stopReasonCode`. Broadcasts a session update. |
| `reset(sessionID: ID!): GameState!` | Reset a session to the map's starting state. Broadcasts a session update. |
| `createMap(name: String!, map: GameMapInput!): GameMap!` | Save a new map definition (requires admin API key). If `map.name` is empty internally, the resolver uses the `name` argument. |
| `updateMap(name: String!, patch: GameMapPatchInput!): GameMap!` | Partially update a map (requires admin API key); omitted fields keep existing values. |
| `validateMap(map: GameMapInput!): MapValidationResult!` | Check a map definition for validity and winnability without saving it. |

## Subscriptions

Subscriptions use the GraphQL WebSocket transport on `/graphql`.

| Subscription | Description |
| --- | --- |
| `sessionUpdated(sessionID: ID!): GameState!` | Emits the new state after `move`, `bulkMove`, or `reset` for the selected session. |
| `lobbyUpdated: GameState!` | Emits lobby-wide state updates from the WebSocket hub. |

Example:

```graphql
subscription SessionUpdated($sessionID: ID!) {
  sessionUpdated(sessionID: $sessionID) {
    playerPos { x y }
    battery
    score
    victory
    gameOver
    message
  }
}
```

## Enums

```graphql
enum Direction { UP DOWN LEFT RIGHT }
enum SortOrder { ASC DESC }
enum SessionSort { CREATED ACTION }
```

## Common examples

### List maps

```graphql
query Maps {
  maps {
    mapId
    filename
    name
    description
    gridSize
    maxBattery
  }
}
```

### Create a session

```graphql
mutation CreateSession($mapID: String) {
  createSession(mapID: $mapID) {
    id
    mapName
    createdAt
    lastActionAt
    gameState {
      playerPos { x y }
      battery
      maxBattery
      score
      message
      fogRadius
      nearbyGrid { x y type visited id allowedDirections }
    }
  }
}
```

Variables:

```json
{ "mapID": "easy" }
```

### Get current state

This query is fog-safe: it works in every session because it never selects `grid`.

```graphql
query State($sessionID: ID!) {
  gameState(sessionID: $sessionID) {
    mapName
    playerPos { x y }
    battery
    maxBattery
    batteryRisk
    score
    totalParks
    visitedParks { id visited }
    victory
    gameOver
    message
    fogEnabled
    fogRadius
    nearbyGrid { x y type visited id allowedDirections }
  }
}
```

`nearbyGrid` is a `(2r+1) x (2r+1)` window around the player, where `r = fogRadius` (without fog, `r = 1`, a 3x3 window). Every cell carries its map coordinates in `x` / `y`; select them instead of computing positions from indexes (`nearbyGrid[j][i]` is the cell at `(playerPos.x - r + i, playerPos.y - r + j)`). Off-map cells read `building` and keep their off-map coordinates (e.g. `x: -1`).

`batteryRisk` is one of: `SAFE`; `LOW` (battery <= 1/3 of max); `CAUTION` (battery <= distance to nearest charger + 2); `DANGER` (battery <= that distance); `CRITICAL` (battery 0); `WARNING` (no charger on the map); `UNKNOWN`. The distance is Manhattan distance ignoring walls and one-way rules, so treat it as a heuristic, not reachability.

### Get the full grid

`grid(password: String)` returns the whole map, row-major (`grid[y][x]`; `RIGHT` is x+1, `DOWN` is y+1). In a non-fog session no password is needed. In a fog session, selecting `grid` without the correct password raises an error and, because `gameState` is non-null, nulls the whole response, so only select `grid` there when you hold the password.

```graphql
query FullGrid($sessionID: ID!, $password: String) {
  gameState(sessionID: $sessionID) {
    playerPos { x y }
    grid(password: $password) { type visited id allowedDirections }
  }
}
```

Always select `allowedDirections` together with `type`. One-way rule: when the lists are non-empty, a move is allowed only if its direction (`north` = `UP`, `south` = `DOWN`, `east` = `RIGHT`, `west` = `LEFT`) is listed on both the cell you leave and the cell you enter. Wrong-way moves are rejected, cost no battery, and do not end the game.

### Move once

```graphql
mutation Move($sessionID: ID!, $direction: Direction!) {
  move(sessionID: $sessionID, direction: $direction) {
    success
    message
    attemptedTo { x y tileChar tileType passable }
    step {
      idx
      dir
      from { x y }
      to { x y }
      batteryBefore
      batteryAfter
      charged
      park
      victory
    }
    gameState { playerPos { x y } battery score victory gameOver }
    events { type message timestamp position { x y } }
  }
}
```

### Bulk move

```graphql
mutation Bulk($sessionID: ID!) {
  bulkMove(sessionID: $sessionID, moves: [RIGHT, DOWN, LEFT]) {
    success
    movesExecuted
    requestedMoves
    stoppedReason
    stopReasonCode
    startPos { x y }
    endPos { x y }
    startBattery
    endBattery
    scoreDelta
    gameOver
    gameOverCode
    message
    possibleMoves
    truncated
    limit
    batteryRisk
    steps {
      idx
      dir
      from { x y }
      to { x y }
      tileChar
      tileType
      batteryBefore
      batteryAfter
      success
      charged
      park
      victory
    }
    gameState { playerPos { x y } battery score victory gameOver }
  }
}
```

For long planned routes, GraphQL aliases allow multiple bulk moves in one request. Each alias runs after the previous field and resumes from the latest session state:

```graphql
mutation Route($sessionID: ID!) {
  reset(sessionID: $sessionID) { battery score }

  c1: bulkMove(sessionID: $sessionID, moves: [UP, UP, RIGHT, RIGHT, DOWN]) {
    movesExecuted success stoppedReason gameState { playerPos { x y } battery victory gameOver }
  }

  c2: bulkMove(sessionID: $sessionID, moves: [LEFT, LEFT, UP, UP, RIGHT]) {
    movesExecuted success stoppedReason gameState { playerPos { x y } battery victory gameOver }
  }
}
```

### Sessions and history

```graphql
query SessionsAndHistory($sessionID: ID!) {
  sessions(sort: ACTION, order: DESC, limit: 10) {
    count
    total
    sort
    order
    sessions { id mapName lastActionAt gameState { score victory gameOver } }
  }

  history(sessionID: $sessionID, page: 1, limit: 20, order: DESC) {
    totalMoves
    page
    pageSize
    totalPages
    hasNext
    hasPrevious
    moves {
      moveNumber
      action
      fromPosition { x y }
      toPosition { x y }
      battery
      success
      timestamp
    }
  }
}
```

### Create a map

```graphql
mutation CreateMap($name: String!, $map: GameMapInput!) {
  createMap(name: $name, map: $map) {
    name
    description
    gridSize
    maxBattery
    startingBattery
    layout
    cellConfigs { key type allowedDirections }
  }
}
```

A `GameMapInput` requires `name`, `description`, `gridSize`, `maxBattery`, `startingBattery`, `layout`, and `legend`; `cellConfigs` is optional (defaults to `[]`) and sets per-cell `allowedDirections` for one-way roads. Use `validateMap` first to check winnability.

## Type reference

### Session types

```graphql
type Session {
  id: ID!
  displayName: String
  mapName: String!
  createdAt: String!
  lastActionAt: String!
  gameState: GameState!
  gameMap: GameMap!
  generatedGridPassword: String  # only set in createSession when fog is on and no gridPassword was given
}

type SessionList {
  count: Int!
  total: Int!
  sessions: [Session!]!
  sort: String!
  order: String!
}

type UnifiedSessions {
  mapName: String!
  count: Int!
  sessions: [UnifiedSession!]!
}

type UnifiedSession {
  sessionId: ID!
  createdAt: String!
  lastActionAt: String!
  gameState: GameState!
  gameMap: GameMap!
}
```

### Game state and movement types

```graphql
type GameState {
  grid(password: String): [[Cell!]!]!
  playerPos: Position!
  battery: Int!
  maxBattery: Int!
  score: Int!
  totalParks: Int!
  visitedParks: [VisitedPark!]!
  message: String!
  gameOver: Boolean!
  victory: Boolean!
  mapName: String!
  moveHistory: [MoveHistoryEntry!]!
  totalMoves: Int!
  resetCount: Int!
  nearbyGrid: [[Cell!]!]!
  currentMoves: [MoveHistoryEntry!]!
  currentMovesCount: Int!
  batteryRisk: String!
  fogEnabled: Boolean!
  fogRadius: Int!
  moveDelayMs: Int!
}

type Cell { x: Int!, y: Int!, type: String!, visited: Boolean!, id: String!, allowedDirections: [String!]! }
type Position { x: Int!, y: Int! }
type VisitedPark { id: String!, visited: Boolean! }

type MoveHistoryEntry {
  action: String!
  fromPosition: Position!
  toPosition: Position!
  battery: Int!
  timestamp: Int!
  success: Boolean!
  moveNumber: Int!
}
```

### Result types

```graphql
type MoveResult {
  success: Boolean!
  gameState: GameState!
  message: String!
  events: [GameEvent!]!
  step: StepInfo
  attemptedTo: AttemptInfo
}

type BulkMoveResult {
  movesExecuted: Int!
  totalMoves: Int!
  requestedMoves: Int!
  success: Boolean!
  gameState: GameState!
  events: [GameEvent!]!
  stoppedReason: String!
  stopReasonCode: String!
  stoppedOnMove: Int!
  truncated: Boolean!
  limit: Int!
  startPos: Position!
  endPos: Position!
  startBattery: Int!
  endBattery: Int!
  scoreDelta: Int!
  steps: [StepInfo!]!
  attemptedTo: AttemptInfo
  gameOver: Boolean!
  gameOverCode: String!
  message: String!
  possibleMoves: [String!]!
  batteryRisk: String!
}

type GameEvent { type: String!, message: String!, timestamp: String!, position: Position! }

type StepInfo {
  idx: Int!
  dir: String!
  from: Position!
  to: Position!
  tileChar: String!
  tileType: String!
  batteryBefore: Int!
  batteryAfter: Int!
  success: Boolean!
  charged: Boolean!
  park: Boolean!
  victory: Boolean!
}

type AttemptInfo {
  x: Int!
  y: Int!
  tileChar: String!
  tileType: String!
  passable: Boolean!
}

type MapValidationResult { valid: Boolean!, winnable: Boolean!, message: String!, error: String }
```

### Map types

```graphql
type MapInfo {
  filename: String!
  mapId: String!
  name: String!
  description: String!
  gridSize: Int!
  maxBattery: Int!
}

type GameMap {
  name: String!
  description: String!
  gridSize: Int!
  maxBattery: Int!
  startingBattery: Int!
  layout(password: String): [String!]!
  legend: [LegendEntry!]!
  cellConfigs: [CellConfigEntry!]!
}

type LegendEntry { key: String!, value: String! }

type CellConfigEntry { key: String!, type: String!, allowedDirections: [String!]! }
```

Input types mirror the map output types:

```graphql
input GameMapInput {
  name: String!
  description: String!
  gridSize: Int!
  maxBattery: Int!
  startingBattery: Int!
  layout: [String!]!
  legend: [LegendEntryInput!]!
  cellConfigs: [CellConfigEntryInput!] = []
}

input LegendEntryInput { key: String!, value: String! }

input CellConfigEntryInput { key: String!, type: String!, allowedDirections: [String!]! }

# Partial update input for updateMap; omitted fields keep existing values.
input GameMapPatchInput {
  name: String
  description: String
  gridSize: Int
  maxBattery: Int
  startingBattery: Int
  layout: [String!]
  legend: [LegendEntryInput!]
  cellConfigs: [CellConfigEntryInput!]
}
```

## Grid and gameplay notes

Common map characters are:

| Character | Typical type | Passable | Effect |
| --- | --- | --- | --- |
| `R` | road | yes | Normal movement. |
| `H` | home | yes | Recharges to max battery. |
| `S` | supercharger | yes | Recharges to max battery. |
| `P` | park | yes | Marks park visited; visiting all parks wins. |
| `B` | building | no | Obstacle. |
| `W` | water | no | Obstacle. |

Gameplay rules exposed through the API:

- Each successful move costs 1 battery. `bulkMove` accepts at most 50 moves; extra moves are dropped and `truncated` is `true`.
- Recharging cells restore battery to `maxBattery`. Reaching 0 battery away from a charger ends the game.
- `victory` becomes `true` once all parks are visited.
- Moving into a building, water, or off the map ends the game immediately (the crash costs no battery). Never probe by bumping into cells; read `nearbyGrid`/`grid` instead.
- Moves against a one-way cell's `allowedDirections` are rejected without cost and do not end the game.
- After `bulkMove`, inspect `stoppedReason`, `stopReasonCode`, `attemptedTo`, and `possibleMoves` before replanning. `possibleMoves` lists the directions you can legally move from the end position (walls, map edge, one-way roads, battery considered); it is empty after game over.
