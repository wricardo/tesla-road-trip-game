export const SESSIONS_QUERY = `
  query Sessions {
    sessions {
      sessions {
        id
        configName
        gameState {
          battery
          maxBattery
          score
          victory
          gameOver
          totalMoves
          playerPos { x y }
          grid { type visited id }
        }
      }
    }
  }
`;

export const LOBBY_SUBSCRIPTION = `
  subscription LobbyUpdated {
    lobbyUpdated {
      configName
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
      configName
      playerPos { x y }
      grid { type visited id }
    }
  }
`;

export const CONFIGS_QUERY = `
  query Configs {
    configs {
      configId
      name
      description
      gridSize
      maxBattery
    }
  }
`;

export const CREATE_SESSION_MUTATION = `
  mutation CreateSession($configName: String) {
    createSession(configName: $configName) {
      id
      configName
    }
  }
`;
