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
curl -s http://localhost:8080/graphql \
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
| `sessions(sort: SessionSort = ACCESSED, order: SortOrder = DESC, limit: Int): SessionList!` | List saved sessions. Sort by `CREATED` or `ACCESSED`; order by `ASC` or `DESC`. A positive `limit` truncates the returned list while `total` remains the untruncated count. |
| `unifiedSessions(mapName: String): UnifiedSessions!` | List sessions with each session's state and map bundled together. Optional `mapName` filters results. |
| `gameState(sessionID: ID!): GameState!` | Get the current state for a session. |
| `history(sessionID: ID!, page: Int = 1, limit: Int = 50, order: SortOrder = DESC): HistoryResponse!` | Page through move history. |
| `maps: [MapInfo!]!` | List available maps. |
| `map(name: String!): GameMap!` | Load a full map definition by name/id. |

## Mutations

| Mutation | Description |
| --- | --- |
| `createSession(mapID: String, mapName: String): Session!` | Create a new game session. `mapID` and `mapName` are aliases in the resolver; pass one of them. If neither is passed, the service default map is used. |
| `deleteSession(id: ID!): DeleteSessionResult!` | Delete a saved session. |
| `move(sessionID: ID!, direction: Direction!, reset: Boolean = false): MoveResult!` | Execute one move. Optional `reset: true` resets the session before moving. Broadcasts a session update. |
| `bulkMove(sessionID: ID!, moves: [Direction!]!, reset: Boolean = false): BulkMoveResult!` | Execute a sequence of moves. Stops early on game-ending or invalid conditions reported in `stoppedReason` / `stopReasonCode`. Broadcasts a session update. |
| `reset(sessionID: ID!): GameState!` | Reset a session to the map's starting state. Broadcasts a session update. |
| `createMap(name: String!, map: GameMapInput!): GameMap!` | Save a new map definition. If `map.name` is empty internally, the resolver uses the `name` argument. |

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
enum SessionSort { CREATED ACCESSED }
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
    lastAccessedAt
    gameState {
      playerPos { x y }
      battery
      maxBattery
      score
      message
      localView3x3
    }
  }
}
```

Variables:

```json
{ "mapID": "easy" }
```

### Get current state

```graphql
query State($sessionID: ID!) {
  gameState(sessionID: $sessionID) {
    mapName
    playerPos { x y }
    battery
    maxBattery
    batteryRisk
    score
    visitedParks { id visited }
    victory
    gameOver
    message
    localView3x3
    grid { type visited id }
  }
}
```

`grid` is row-major (`grid[y][x]`). `localView3x3` is a compact three-row view centered on the player. `batteryRisk` is a human-readable risk label such as `safe`, `moderate`, `high`, or `critical`.

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
    localView3x3
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
  sessions(sort: ACCESSED, order: DESC, limit: 10) {
    count
    total
    sort
    order
    sessions { id mapName lastAccessedAt gameState { score victory gameOver } }
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
  }
}
```

A `GameMapInput` requires all fields from `GameMap`: `name`, `description`, `gridSize`, `maxBattery`, `startingBattery`, `layout`, `legend`, `wallCrashEndsGame`, and `messages`.

## Type reference

### Session types

```graphql
type Session {
  id: ID!
  mapName: String!
  createdAt: String!
  lastAccessedAt: String!
  gameState: GameState!
  gameMap: GameMap!
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
  lastAccessedAt: String!
  gameState: GameState!
  gameMap: GameMap!
}
```

### Game state and movement types

```graphql
type GameState {
  grid: [[Cell!]!]!
  playerPos: Position!
  battery: Int!
  maxBattery: Int!
  score: Int!
  visitedParks: [VisitedPark!]!
  message: String!
  gameOver: Boolean!
  victory: Boolean!
  mapName: String!
  moveHistory: [MoveHistoryEntry!]!
  totalMoves: Int!
  localView: [SurroundingCell!]!
  currentMoves: [MoveHistoryEntry!]!
  currentMovesCount: Int!
  localView3x3: [String!]!
  batteryRisk: String!
}

type Cell { type: String!, visited: Boolean!, id: String! }
type Position { x: Int!, y: Int! }
type VisitedPark { id: String!, visited: Boolean! }
type SurroundingCell { x: Int!, y: Int!, type: String! }

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
  localView3x3: [String!]!
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
  layout: [String!]!
  legend: [LegendEntry!]!
  wallCrashEndsGame: Boolean!
  messages: MapMessages!
}

type LegendEntry { key: String!, value: String! }

type MapMessages {
  welcome: String!
  homeCharge: String!
  superchargerCharge: String!
  parkVisited: String!
  parkAlreadyVisited: String!
  victory: String!
  outOfBattery: String!
  stranded: String!
  cantMove: String!
  batteryStatus: String!
  hitWall: String!
}
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
  wallCrashEndsGame: Boolean!
  messages: MapMessagesInput!
}

input LegendEntryInput { key: String!, value: String! }

input MapMessagesInput {
  welcome: String!
  homeCharge: String!
  superchargerCharge: String!
  parkVisited: String!
  parkAlreadyVisited: String!
  victory: String!
  outOfBattery: String!
  stranded: String!
  cantMove: String!
  batteryStatus: String!
  hitWall: String!
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

- Each successful move costs 1 battery.
- Recharging cells restore battery to `maxBattery`.
- `victory` becomes `true` once all parks are visited.
- `gameOver` becomes `true` on battery depletion and on wall collisions for maps with `wallCrashEndsGame: true`.
- After `bulkMove`, inspect `stoppedReason`, `stopReasonCode`, `attemptedTo`, and `possibleMoves` before replanning.
