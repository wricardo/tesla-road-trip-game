"""Map loading that mirrors game/engine.InitGameStateFromConfig.

The Go engine is the source of truth. Everything here reproduces it:
  * char -> cell type via the fixed legend (R/H/P/S/W/B)
  * extra chars from `cell_configs` are roads with direction restrictions
  * the player starts on the LAST 'H' in scan order (the loop overwrites homePos)
  * parks get ids "park_<n>" in scan order
"""
from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path

import numpy as np

# Cell types, kept in the same order as the engine's constants.
ROAD, HOME, PARK, SUPERCHARGER, WATER, BUILDING = range(6)
CELL_NAMES = ("road", "home", "park", "supercharger", "water", "building")
PASSABLE = (ROAD, HOME, PARK, SUPERCHARGER)

TYPE_OF_CHAR = {
    "R": ROAD,
    "H": HOME,
    "P": PARK,
    "S": SUPERCHARGER,
    "W": WATER,
    "B": BUILDING,
}

# Direction bits. Cardinal names match engine.AllowedDirections.
N, S, W, E = 1, 2, 4, 8
CARDINAL = {"north": N, "south": S, "west": W, "east": E}
ALL_DIRS = N | S | W | E

# Action index -> (dx, dy, direction bit, engine direction string).
# Order matches engine.GetPossibleMoves: up, down, left, right.
ACTIONS = ((0, -1, N, "up"), (0, 1, S, "down"), (-1, 0, W, "left"), (1, 0, E, "right"))
OPPOSITE_BIT = {N: S, S: N, W: E, E: W}


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def maps_dir() -> Path:
    return Path(os.environ.get("TESLA_MAPS_DIR", repo_root() / "maps"))


@dataclass
class GameMap:
    name: str
    h: int
    w: int
    cell: np.ndarray          # int8 (h, w) cell type
    park_id: np.ndarray       # int16 (h, w) park index or -1
    allow: np.ndarray         # uint8 (h, w) bitmask of allowed directions
    max_battery: int
    start_battery: int
    start: tuple[int, int]    # (x, y)

    @property
    def nparks(self) -> int:
        return int((self.cell == PARK).sum())

    def successors(self, x: int, y: int) -> list[tuple[int, int, int]]:
        """Legal (nx, ny, action) ignoring battery. Walls are never legal: a
        wall move is an instant loss, so the agent must never consider it."""
        out = []
        for a, (dx, dy, bit, _) in enumerate(ACTIONS):
            nx, ny = x + dx, y + dy
            if not (0 <= nx < self.w and 0 <= ny < self.h):
                continue
            if self.cell[ny, nx] in (WATER, BUILDING):
                continue
            if not self.allow[y, x] & bit:          # exit restriction
                continue
            if not self.allow[ny, nx] & bit:        # entry restriction
                continue
            out.append((nx, ny, a))
        return out

    def render(self, pos=None, visited=()) -> str:
        glyph = ".HPSWB"
        rows = []
        for y in range(self.h):
            row = []
            for x in range(self.w):
                ch = glyph[self.cell[y, x]]
                if self.cell[y, x] == ROAD:
                    ch = "R"
                if (x, y) in visited:
                    ch = "*"
                if pos == (x, y):
                    ch = "@"
                row.append(ch)
            rows.append("".join(row))
        return "\n".join(rows)


def load(name: str, size: int | None = None) -> GameMap:
    """Load maps/<name>.json. `size` overrides the JSON grid_size if given
    (useful for the 50x50 map, which we crop for training)."""
    path = maps_dir() / f"{name}.json"
    cfg = json.loads(path.read_text())
    gs = int(cfg.get("grid_size", size or len(cfg["layout"])))
    w = gs

    cell = np.full((gs, w), BUILDING, dtype=np.int8)
    allow = np.full((gs, w), ALL_DIRS, dtype=np.uint8)
    park_id = np.full((gs, w), -1, dtype=np.int16)
    cell_configs = cfg.get("cell_configs") or {}

    park_count = 0
    home = (0, 0)
    for y, row in enumerate(cfg["layout"]):
        if y >= gs:
            break
        for x, ch in enumerate(row):
            if x >= w:
                break
            if ch in TYPE_OF_CHAR:
                cell[y, x] = TYPE_OF_CHAR[ch]
                if ch == "H":
                    home = (x, y)   # last one wins, same as the Go loop
                elif ch == "P":
                    park_id[y, x] = park_count
                    park_count += 1
            elif ch in cell_configs:
                cc = cell_configs[ch]
                cell[y, x] = ROAD   # engine validation only allows road here
                dirs = cc.get("allowed_directions") or []
                if dirs:
                    mask = 0
                    for d in dirs:
                        mask |= CARDINAL[d]
                    allow[y, x] = mask
            else:
                raise ValueError(f"{name}: unknown layout char {ch!r} at ({x},{y})")

    return GameMap(
        name=cfg.get("name", name),
        h=gs,
        w=w,
        cell=cell,
        park_id=park_id,
        allow=allow,
        max_battery=int(cfg["max_battery"]),
        start_battery=int(cfg["starting_battery"]),
        start=home,
    )


def available(names: list[str] | None = None) -> list[str]:
    if names:
        return names
    return sorted(p.stem for p in maps_dir().glob("*.json") if p.stem != "custom")
