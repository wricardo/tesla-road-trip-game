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
`,i=`
  mutation CreateSession($mapID: String) {
    createSession(mapID: $mapID) {
      id
      mapName
    }
  }
`;export{e as a,t as i,n,r,i as t};