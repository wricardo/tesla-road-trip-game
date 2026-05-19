# GraphQL API

The public game API is exposed at `/graphql` using gqlgen. The interactive GraphQL playground is available at `/playground`.

## Examples

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

Bulk move:

```graphql
mutation Bulk($sessionID: ID!) {
  bulkMove(sessionID: $sessionID, moves: [RIGHT, DOWN, LEFT]) {
    success
    movesExecuted
    stoppedReason
    gameState { playerPos { x y } battery score }
  }
}
```

List configs:

```graphql
query {
  configs { configId name description gridSize maxBattery }
}
```

Get state and history:

```graphql
query State($sessionID: ID!) {
  gameState(sessionID: $sessionID) {
    grid { type visited id }
    playerPos { x y }
    battery
    score
    victory
    localView3x3
  }
  history(sessionID: $sessionID, limit: 10) {
    totalMoves
    moves { moveNumber action success battery }
  }
}
```

REST game routes have been removed; use `/graphql` for game operations. WebSocket remains available at `/ws` for realtime updates.
