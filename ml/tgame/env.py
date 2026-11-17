"""Exact re-implementation of the Tesla Road Trip rules as a small MDP.

Mirrors engine.GameState.MovePlayer, including the details that make the game
harder than it looks:

  * moving into water/building/out-of-bounds ends the game immediately
  * direction-restricted road cells gate both exit and entry
  * home/supercharger refill the battery on arrival
  * battery hits 0 away from a charger -> stranded, game over
  * victory is checked before the stranded check
"""
from __future__ import annotations

from dataclasses import dataclass, field

import numpy as np

from .maps import ACTIONS, HOME, PARK, PASSABLE, SUPERCHARGER, WATER, BUILDING, GameMap

N_ACTIONS = 4


@dataclass
class StepInfo:
    park_visited: bool = False
    crash: bool = False
    stranded: bool = False
    victory: bool = False
    blocked: bool = False   # direction restriction refused the move; no cost


@dataclass
class TeslaEnv:
    m: GameMap
    reward_park: float = 1.0
    reward_win: float = 10.0
    reward_death: float = -5.0
    x: int = field(init=False)
    y: int = field(init=False)
    battery: int = field(init=False)
    visited: int = field(init=False)      # bitmask of visited parks
    done: bool = field(init=False)
    win: bool = field(init=False)
    steps: int = field(init=False)

    def __post_init__(self):
        self.reset()

    # ------------------------------------------------------------------ state
    def reset(self) -> "TeslaEnv":
        self.x, self.y = self.m.start
        self.battery = self.m.start_battery
        self.visited = 0
        self.done = False
        self.win = False
        self.steps = 0
        return self

    @property
    def full_mask(self) -> int:
        return (1 << self.m.nparks) - 1

    @property
    def score(self) -> int:
        return bin(self.visited).count("1")

    def cell(self, x: int, y: int) -> int:
        return int(self.m.cell[y, x])

    def legal_mask(self) -> np.ndarray:
        """Boolean mask of actions that are both possible and safe.

        Wall moves are excluded: they are *legal* in the engine's sense (the
        engine accepts the call and then kills you). For a learning agent the
        only sane action space is "moves that do not lose immediately", so the
        mask encodes that convention explicitly.
        """
        mask = np.zeros(N_ACTIONS, dtype=bool)
        if self.done:
            return mask
        for _, _, a in self.m.successors(self.x, self.y):
            mask[a] = True
        return mask

    # ------------------------------------------------------------------- step
    def step(self, action: int) -> tuple[float, bool, StepInfo]:
        if self.done:
            raise RuntimeError("step() on a terminal state; call reset()")
        info = StepInfo()
        dx, dy, bit, _name = ACTIONS[action]
        nx, ny = self.x + dx, self.y + dy

        # Exit restriction. The engine refuses without cost.
        if not self.m.allow[self.y, self.x] & bit:
            info.blocked = True
            return 0.0, False, info

        # Anything impassable is an instant loss.
        if not (0 <= nx < self.m.w and 0 <= ny < self.m.h) or self.m.cell[ny, nx] in (WATER, BUILDING):
            self.done = True
            info.crash = True
            return self.reward_death, True, info

        # Entry restriction: refused without cost.
        if not self.m.allow[ny, nx] & bit:
            info.blocked = True
            return 0.0, False, info

        # engine checks battery <= 0 after the wall check; it can only be 0 here
        # if a previous step left us alive on a charger cell, which refills.
        self.x, self.y = nx, ny
        self.battery -= 1
        self.steps += 1
        reward = 0.0

        ctype = self.cell(nx, ny)
        if ctype in (HOME, SUPERCHARGER):
            self.battery = self.m.max_battery
        elif ctype == PARK:
            pid = int(self.m.park_id[ny, nx])
            bit_p = 1 << pid
            if not self.visited & bit_p:
                self.visited |= bit_p
                reward += self.reward_park
                info.park_visited = True
                if self.visited == self.full_mask:
                    self.done = True
                    self.win = True
                    info.victory = True
                    reward += self.reward_win
                    return reward, True, info

        if self.battery == 0 and ctype not in (HOME, SUPERCHARGER):
            self.done = True
            info.stranded = True
            reward += self.reward_death

        return reward, self.done, info

    # --------------------------------------------------------------- utilities
    def copy(self) -> "TeslaEnv":
        e = object.__new__(TeslaEnv)
        e.m = self.m
        e.reward_park = self.reward_park
        e.reward_win = self.reward_win
        e.reward_death = self.reward_death
        e.x, e.y = self.x, self.y
        e.battery = self.battery
        e.visited = self.visited
        e.done = self.done
        e.win = self.win
        e.steps = self.steps
        return e

    def encode(self) -> np.ndarray:
        from .features import encode
        return encode(self)

    def render(self) -> str:
        visited_xy = set()
        for y in range(self.m.h):
            for x in range(self.m.w):
                pid = int(self.m.park_id[y, x])
                if pid >= 0 and self.visited >> pid & 1:
                    visited_xy.add((x, y))
        return self.m.render((self.x, self.y), visited_xy)

    def state_key(self) -> tuple:
        return (self.x, self.y, self.battery, self.visited, self.done, self.win)


def from_live_state(grid_types, park_ids, visited_ids, pos, battery, max_battery) -> TeslaEnv:
    """Build an env from a GraphQL game state so a trained policy can drive the
    real server. Park ids are 'park_<n>' as produced by the Go engine."""
    h, w = len(grid_types), len(grid_types[0])
    cell = np.array([[int(t) for t in row] for row in grid_types], dtype=np.int8)
    park_id = np.full((h, w), -1, dtype=np.int16)
    for y in range(h):
        for x in range(w):
            pid = park_ids[y][x]
            if pid:
                park_id[y, x] = int(pid.split("_")[1])
    m = GameMap(
        name="live",
        h=h,
        w=w,
        cell=cell,
        park_id=park_id,
        allow=np.full((h, w), 15, dtype=np.uint8),
        max_battery=max_battery,
        start_battery=battery,
        start=pos,
    )
    env = TeslaEnv(m)
    env.x, env.y = pos
    env.battery = battery
    mask = 0
    for pid in visited_ids:
        mask |= 1 << int(pid.split("_")[1])
    env.visited = mask
    return env
