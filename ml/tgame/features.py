"""Feature maps for the policy network.

A 3x3 CNN on raw cell types cannot see 15 cells down a corridor, and battery
management needs exactly that kind of long-range information. So the encoder
hands the network a few *derived* planes that any agent could compute itself:
BFS distances to the nearest reachable unvisited park and to the nearest
charger. Nothing here leaks the answer: no target park, no planned path.
"""
from __future__ import annotations

import numpy as np

from .maps import ACTIONS, HOME, PARK, PASSABLE, SUPERCHARGER, GameMap

N_CHANNELS = 13
BIG = 10_000


def _predecessors(m: GameMap) -> list[list[int]]:
    """pred[v] = list of cells u with an allowed move u -> v (index y*w+x)."""
    pred: list[list[int]] = [[] for _ in range(m.h * m.w)]
    for y in range(m.h):
        for x in range(m.w):
            if m.cell[y, x] not in PASSABLE:
                continue
            for nx, ny, _a in m.successors(x, y):
                pred[ny * m.w + nx].append(y * m.w + x)
    return pred


def _bfs_from(pred: list[list[int]], sources: list[int], n: int) -> np.ndarray:
    dist = np.full(n, BIG, dtype=np.int32)
    frontier = list(sources)
    for s in frontier:
        dist[s] = 0
    head = 0
    while head < len(frontier):
        v = frontier[head]
        head += 1
        dv = dist[v] + 1
        for u in pred[v]:
            if dist[u] == BIG:
                dist[u] = dv
                frontier.append(u)
    return dist


class MapFeatures:
    """Per-map cached distance planes."""

    def __init__(self, m: GameMap):
        self.m = m
        n = m.h * m.w
        pred = _predecessors(m)
        self.park_dist = np.full((max(m.nparks, 1), m.h, m.w), BIG, dtype=np.int32)
        for y in range(m.h):
            for x in range(m.w):
                pid = int(m.park_id[y, x])
                if pid >= 0:
                    self.park_dist[pid] = _bfs_from(pred, [y * m.w + x], n).reshape(m.h, m.w)
        chargers = [
            y * m.w + x
            for y in range(m.h)
            for x in range(m.w)
            if m.cell[y, x] in (HOME, SUPERCHARGER)
        ]
        self.charger_dist = _bfs_from(pred, chargers, n).reshape(m.h, m.w)
        self.passable = np.isin(m.cell, PASSABLE).astype(np.float32)
        # One-way roads are part of the state: they gate both entry and exit, so
        # the policy cannot plan around them unless it can see them. One plane
        # per direction, 1.0 where that move is forbidden.
        self.blocked = np.stack(
            [((m.allow & bit) == 0).astype(np.float32) for _dx, _dy, bit, _n in ACTIONS]
        )
        self.is_home = (m.cell == HOME).astype(np.float32)
        self.is_super = (m.cell == SUPERCHARGER).astype(np.float32)
        self.norm = float(m.h + m.w)

    def encode(
        self,
        x: int,
        y: int,
        battery: int,
        visited: int,
        max_battery: int,
    ) -> np.ndarray:
        m = self.m
        f = np.empty((N_CHANNELS, m.h, m.w), dtype=np.float32)
        f[0] = self.passable
        f[1] = self.is_home
        f[2] = self.is_super

        unvisited = np.zeros(max(m.nparks, 1), dtype=bool)
        for pid in range(m.nparks):
            unvisited[pid] = not (visited >> pid & 1)
        park_mask = m.park_id >= 0
        per_cell = unvisited[np.maximum(m.park_id, 0)]
        f[3] = (park_mask & per_cell).astype(np.float32)
        f[4] = (park_mask & ~per_cell).astype(np.float32)
        f[5] = 0.0
        f[5, y, x] = 1.0

        if m.nparks and unvisited.any():
            nearest = np.where(unvisited[:, None, None], self.park_dist, BIG).min(axis=0)
        else:
            nearest = np.zeros((m.h, m.w), dtype=np.int32)
        f[6] = np.clip(nearest / self.norm, 0.0, 1.0)
        f[7] = np.clip(self.charger_dist / self.norm, 0.0, 1.0)
        f[8] = float(battery) / max(max_battery, 1)
        f[9:13] = self.blocked
        return f


_CACHE: dict[int, MapFeatures] = {}


def map_features(m: GameMap) -> MapFeatures:
    key = id(m)
    mf = _CACHE.get(key)
    if mf is None:
        mf = _CACHE[key] = MapFeatures(m)
    return mf


def encode(env) -> np.ndarray:
    mf = map_features(env.m)
    return mf.encode(env.x, env.y, env.battery, env.visited, env.m.max_battery)
