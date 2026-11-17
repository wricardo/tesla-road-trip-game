"""Exact solver: distance-to-victory over (cell, battery, visited-parks).

Standard retrograde (backwards) BFS. Every state has unit-cost moves, so a BFS
seeded from *all* won states, expanding predecessors, yields the exact minimum
number of moves to win from every state. That gives us:

  * an optimal action label for any state (supervised targets for imitation)
  * the optimal move count, so we can report "agent vs optimal"

State space is h*w*(max_battery+1)*2^nparks. Keep maps with more than ~30M
states out of the oracle (the 50x50 map has 24 parks -> 1.7e13 states).
"""
from __future__ import annotations

import numpy as np

from .env import TeslaEnv
from .maps import ACTIONS, HOME, PARK, PASSABLE, SUPERCHARGER, WATER, BUILDING, GameMap

MAX_STATES = 30_000_000


class TooComplex(RuntimeError):
    pass


class Oracle:
    def __init__(self, m: GameMap):
        self.m = m
        self.nb = m.max_battery + 1
        self.masks = 1 << m.nparks
        self.full = self.masks - 1
        self.stride_pos = self.nb * self.masks
        self.total = m.h * m.w * self.stride_pos
        if self.total > MAX_STATES:
            raise TooComplex(
                f"{m.name}: {self.total/1e6:.1f}M states (limit {MAX_STATES/1e6:.0f}M); "
                "the oracle is exponential in the number of parks"
            )
        self.dist = np.full(self.total, -1, dtype=np.int32)
        self._build()

    # Only some (cell, battery) pairs can exist: leaving a non-charger always
    # costs a unit, so max battery happens only on a charger (or at the start
    # cell, whose battery is starting_battery). Without this filter the reverse
    # search happily invents states like "road with a full battery" and its
    # distances leak into reachable states.
    def _state_ok(self, x: int, y: int, b: int) -> bool:
        if b <= 0 or b > self.m.max_battery:
            return False
        charger = self.m.cell[y, x] in (HOME, SUPERCHARGER)
        if (x, y) == self.m.start:
            return b == self.m.start_battery or (charger and b == self.m.max_battery)
        return b == self.m.max_battery if charger else b < self.m.max_battery

    # --------------------------------------------------------------- indexing
    def idx(self, x: int, y: int, battery: int, mask: int) -> int:
        return (y * self.m.w + x) * self.stride_pos + battery * self.masks + mask

    def _build(self) -> None:
        m, dist = self.m, self.dist
        q = np.empty(self.total, dtype=np.int64)
        head = tail = 0

        # Seed: every won state (all parks visited) is distance 0.
        for y in range(m.h):
            for x in range(m.w):
                if m.cell[y, x] not in PASSABLE:
                    continue
                base = (y * m.w + x) * self.stride_pos
                for b in range(self.nb):
                    if not self._state_ok(x, y, b):
                        continue
                    i = base + b * self.masks + self.full
                    if dist[i] < 0:
                        dist[i] = 0
                        q[tail] = i
                        tail += 1

        while head < tail:
            s = int(q[head])
            head += 1
            d = int(dist[s]) + 1
            mask = s % self.masks
            rest = s // self.masks
            b = rest % self.nb
            pos = rest // self.nb
            x, y = pos % m.w, pos // m.w
            dest_charger = m.cell[y, x] in (HOME, SUPERCHARGER)
            park_here = int(m.park_id[y, x])

            # A state with 0 battery exists only on a charger (refilled) or as a
            # victory arrived on the final park (the win check precedes the
            # stranded check in the engine).
            if b == 0 and not dest_charger and not (mask == self.full and park_here >= 0):
                continue

            masks = (mask, mask & ~(1 << park_here)) if park_here >= 0 else (mask,)

            for dx, dy, bit, _ in ACTIONS:
                px, py = x - dx, y - dy
                if not (0 <= px < m.w and 0 <= py < m.h):
                    continue
                if m.cell[py, px] not in PASSABLE:
                    continue
                if not m.allow[py, px] & bit:      # exit from predecessor
                    continue
                if not m.allow[y, x] & bit:        # entry into this cell
                    continue
                p_charger = m.cell[py, px] in (HOME, SUPERCHARGER)
                if p_charger:
                    # a state on a charger always holds max battery
                    pred_batteries = (m.max_battery,) if not dest_charger else (m.max_battery,)
                elif dest_charger:
                    # arriving at a charger refills, so the arrival battery carries
                    # no information: any non-zero predecessor battery works
                    pred_batteries = range(1, m.max_battery + 1)
                else:
                    if b + 1 > m.max_battery:
                        continue
                    pred_batteries = (b + 1,)
                base = (py * m.w + px) * self.stride_pos
                for pm in masks:
                    if pm == self.full:            # would already have won
                        continue
                    for pb in pred_batteries:
                        if not self._state_ok(px, py, pb):
                            continue
                        i = base + pb * self.masks + pm
                        if dist[i] < 0:
                            dist[i] = d
                            q[tail] = i
                            tail += 1

    # ---------------------------------------------------------------- queries
    def distance(self, x: int, y: int, battery: int, mask: int) -> int:
        if not (0 <= x < self.m.w and 0 <= y < self.m.h):
            return -1
        if not (0 <= battery <= self.m.max_battery):
            return -1
        return int(self.dist[self.idx(x, y, battery, mask)])

    def best_action(self, env: TeslaEnv) -> int | None:
        """Optimal first action, or None if the state is winning-impossible."""
        if env.done:
            return None
        best, best_d = None, None
        legal = env.legal_mask()
        for a in range(4):
            if not legal[a]:
                continue
            nxt = env.copy()
            nxt.step(a)
            if nxt.done and not nxt.win:
                continue
            d = 0 if nxt.win else self.distance(nxt.x, nxt.y, nxt.battery, nxt.visited)
            if d < 0:
                continue
            if best_d is None or d < best_d:
                best, best_d = a, d
        return best

    def optimal_moves(self, env: TeslaEnv) -> int:
        return self.distance(env.x, env.y, env.battery, env.visited)


def solve(m: GameMap) -> Oracle:
    return Oracle(m)
