"""Play a real session on the game server with the trained policy.

    python -m tgame.play_live --map easy --url http://localhost:9191

Creates a session, then loops: read the server's game state, rebuild the
feature maps from it, ask the network for a move, POST it. The server is the
ground truth -- if the env model were wrong, the moves would diverge from the
real rules and the run would visibly stall or crash.
"""
from __future__ import annotations

import argparse
import json
import urllib.request

from .env import from_live_state
from .maps import CELL_NAMES
from .net import load_policy, greedy_action

TYPE_INDEX = {name: i for i, name in enumerate(CELL_NAMES)}
DIR_NAMES = ("UP", "DOWN", "LEFT", "RIGHT")

STATE_FIELDS = """
  grid { type visited id }
  playerPos { x y }
  battery maxBattery score gameOver victory message mapName
  visitedParks { id visited }
"""


def gql(url: str, query: str, variables: dict) -> dict:
    body = json.dumps({"query": query, "variables": variables}).encode()
    req = urllib.request.Request(
        url.rstrip("/") + "/graphql", body, {"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=20) as resp:
        payload = json.loads(resp.read())
    if payload.get("errors"):
        raise RuntimeError(payload["errors"][0].get("message"))
    return payload["data"]


def create_session(url: str, map_id: str) -> tuple[str, dict]:
    data = gql(
        url,
        "mutation($mapID: String) { createSession(mapID: $mapID) "
        "{ id gameState {" + STATE_FIELDS + "} } }",
        {"mapID": map_id},
    )
    sess = data["createSession"]
    if not sess["id"]:
        raise RuntimeError("server returned an empty session id; check the map id")
    return sess["id"], sess["gameState"]


def get_state(url: str, sid: str) -> dict:
    data = gql(
        url,
        "query($id: ID!) { session(id: $id) { gameState {" + STATE_FIELDS + "} } }",
        {"id": sid},
    )
    return data["session"]["gameState"]


def move(url: str, sid: str, direction: int) -> dict:
    data = gql(
        url,
        "mutation($id: ID!, $d: Direction!) { move(sessionID: $id, direction: $d) "
        "{ success message gameState {" + STATE_FIELDS + "} } }",
        {"id": sid, "d": DIR_NAMES[direction]},
    )
    return data["move"]


def env_from_state(state: dict):
    grid = [[TYPE_INDEX[c["type"]] for c in row] for row in state["grid"]]
    ids = [
        [c["id"] if c["type"] == "park" else "" for c in row] for row in state["grid"]
    ]
    visited = [p["id"] for p in state["visitedParks"] if p["visited"]]
    pos = (state["playerPos"]["x"], state["playerPos"]["y"])
    return from_live_state(
        grid, ids, visited, pos, state["battery"], state["maxBattery"]
    )


def render(state: dict) -> str:
    pos = (state["playerPos"]["x"], state["playerPos"]["y"])
    grid = state["grid"]
    seen = {
        (x, y)
        for y, row in enumerate(grid)
        for x, c in enumerate(row)
        if c["type"] == "park" and c["visited"]
    }
    glyph = {"road": ".", "home": "H", "park": "P", "supercharger": "S",
             "water": "~", "building": "B"}
    rows = []
    for y, row in enumerate(grid):
        cells = []
        for x, c in enumerate(row):
            ch = glyph.get(c["type"], "?")
            if (x, y) in seen:
                ch = "*"
            if (x, y) == pos:
                ch = "@"
            cells.append(ch)
        rows.append("".join(cells))
    return "\n".join(rows)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--map", default="easy")
    ap.add_argument("--url", default="http://localhost:9191")
    ap.add_argument("--model", default="ml/runs/bc.pt")
    ap.add_argument("--max-steps", type=int, default=500)
    ap.add_argument("--quiet", action="store_true")
    args = ap.parse_args()

    net, size = load_policy(args.model)
    sid, state = create_session(args.url, args.map)
    print(f"session {sid} on {state['mapName']} ({len(state['grid'])}x{len(state['grid'][0])})")

    steps = 0
    while not state["gameOver"] and steps < args.max_steps:
        env = env_from_state(state)
        action = greedy_action(net, env, size)
        result = move(args.url, sid, action)
        state = result["gameState"]
        steps += 1
        if not args.quiet:
            print(
                f"  {steps:3d} {DIR_NAMES[action]:5s} -> ({state['playerPos']['x']},"
                f"{state['playerPos']['y']}) battery={state['battery']}/{state['maxBattery']} "
                f"score={state['score']} {state['message']}"
            )
        if not result["success"] and not state["gameOver"]:
            print("  server refused the move; stopping to avoid a loop")
            break

    print()
    print(render(state))
    print(
        f"\n{steps} moves, score={state['score']}, victory={state['victory']}, "
        f"gameOver={state['gameOver']} -- {state['message']}"
    )


if __name__ == "__main__":
    main()
