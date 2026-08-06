export const UPDATE_SESSION_MUTATION = `
  mutation UpdateSession($id: ID!, $displayName: String!) {
    updateSession(id: $id, displayName: $displayName) {
      id
      displayName
      mapName
    }
  }
`;

export const SESSIONS_QUERY = `
  query Sessions {
    sessions {
      sessions {
        id
        displayName
        mapName
        gameState {
          battery
          maxBattery
          score
          victory
          gameOver
          totalMoves
          fogEnabled
          fogRadius
          playerPos { x y }
          nearbyGrid { type visited id allowedDirections }
        }
      }
    }
  }
`;

export const LOBBY_SUBSCRIPTION = `
  subscription LobbyUpdated {
    lobbyUpdated {
      mapName
      battery
      maxBattery
      score
      victory
      gameOver
      totalMoves
      playerPos { x y }
    }
  }
`;

export const SESSION_SUBSCRIPTION = `
  subscription SessionUpdated($sessionID: ID!) {
    sessionUpdated(sessionID: $sessionID) {
      battery
      maxBattery
      score
      victory
      gameOver
      totalMoves
      message
      mapName
      playerPos { x y }
      nearbyGrid { type visited id allowedDirections }
    }
  }
`;

export const MAPS_QUERY = `
  query Maps {
    maps {
      mapId
      name
      description
      gridSize
      maxBattery
    }
  }
`;

export const MAP_QUERY = `
  query Map($name: String!, $password: String) {
    map(name: $name, password: $password) {
      name
      description
      gridSize
      maxBattery
      startingBattery
      layout
      legend { key value }
      cellConfigs { key type allowedDirections }
    }
  }
`;

export const CREATE_SESSION_MUTATION = `
  mutation CreateSession($mapID: String, $fogEnabled: Boolean, $fogRadius: Int, $gridPassword: String, $moveDelayMs: Int) {
    createSession(mapID: $mapID, fogEnabled: $fogEnabled, fogRadius: $fogRadius, gridPassword: $gridPassword, moveDelayMs: $moveDelayMs) {
      id
      mapName
    }
  }
`;

export const MOVE_MUTATION = `
  mutation Move($sessionID: ID!, $direction: Direction!) {
    move(sessionID: $sessionID, direction: $direction) {
      success
      message
      attemptedTo { x y tileChar tileType passable }
      gameState {
        battery
        maxBattery
        score
        victory
        gameOver
        totalMoves
        mapName
        fogEnabled
        fogRadius
        playerPos { x y }
        nearbyGrid { type visited id allowedDirections }
        currentMoves { fromPosition { x y } toPosition { x y } success }
      }
    }
  }
`;

export const RESET_MUTATION = `
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) {
      battery
      maxBattery
      score
      victory
      gameOver
      totalMoves
      mapName
      fogEnabled
      fogRadius
      playerPos { x y }
      nearbyGrid { type visited id allowedDirections }
      currentMoves { fromPosition { x y } toPosition { x y } success }
    }
  }
`;
