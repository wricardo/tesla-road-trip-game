var e=`
  mutation UpdateSession($id: ID!, $displayName: String!) {
    updateSession(id: $id, displayName: $displayName) {
      id
      displayName
      mapName
    }
  }
`,t=`
  query Sessions {
    sessions(sort: ACTION, order: DESC) {
      sessions {
        id
        displayName
        mapName
        lastActionAt
        gameState {
          battery
          maxBattery
          score
          victory
          gameOver
          totalMoves
          resetCount
          fogEnabled
          fogRadius
          playerPos { x y }
          nearbyGrid { type visited id allowedDirections }
        }
      }
    }
  }
`,n=`
  query Maps {
    maps {
      mapId
      name
      description
      gridSize
      maxBattery
    }
  }
`,r=`
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
`,i=`
  mutation CreateSession($mapID: String, $fogEnabled: Boolean, $fogRadius: Int, $gridPassword: String, $moveDelayMs: Int) {
    createSession(mapID: $mapID, fogEnabled: $fogEnabled, fogRadius: $fogRadius, gridPassword: $gridPassword, moveDelayMs: $moveDelayMs) {
      id
      mapName
      generatedGridPassword
    }
  }
`,a=`
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
        resetCount
        totalParks
        message
        mapName
        fogEnabled
        fogRadius
        playerPos { x y }
        nearbyGrid { type visited id allowedDirections }
        currentMoves { fromPosition { x y } toPosition { x y } success }
      }
    }
  }
`,o=`
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) {
      battery
      maxBattery
      score
      victory
      gameOver
      totalMoves
      resetCount
      totalParks
      message
      mapName
      fogEnabled
      fogRadius
      playerPos { x y }
      nearbyGrid { type visited id allowedDirections }
      currentMoves { fromPosition { x y } toPosition { x y } success }
    }
  }
`;export{o as a,a as i,n,t as o,r,e as s,i as t};