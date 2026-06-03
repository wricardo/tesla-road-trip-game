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
          playerPos { x y }
          grid { type visited id allowedDirections }
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
      grid { type visited id allowedDirections }
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
  query Map($name: String!) {
    map(name: $name) {
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
  mutation CreateSession($mapID: String) {
    createSession(mapID: $mapID) {
      id
      mapName
    }
  }
`;
