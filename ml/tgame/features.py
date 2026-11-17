"""Feature maps for the network: local planes, plus optional BFS hints.

The default encoder is local. It exposes the map, the objective set, the car,
the battery and the one-way restrictions -- everything a player can read off the
screen -- but no search result. The network has to learn to navigate.

Planes 6-7 (BFS distances to the nearest unvisited park and the nearest
charger) are an opt-in `hints=True` mode. They exist only as a labelled
distillation baseline: with them the policy is copying the encoder's search, so
results from that setting must never be described as learned planning.
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
    """Per-map cached planes. BFS distance planes are built on first use only,
    so the default (hint-free) encoder never pays for the search."""

    def __init__(self, m: GameMap):
        self.m = m
        self._n = m.h * m.w
        self._pred = _predecessors(m)
        self._park_dist: np.ndarray | None = None
        self._charger_dist: np.ndarray | None = None
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

    @property
    def park_dist(self) -> np.ndarray:
        if self._park_dist is None:
            m = self.m
            self._park_dist = np.full(
                (max(m.nparks, 1), m.h, m.w), BIG, dtype=np.int32
            )
            for y in range(m.h):
                for x in range(m.w):
                    pid = int(m.park_id[y, x])
                    if pid >= 0:
                        self._park_dist[pid] = _bfs_from(
                            self._pred, [y * m.w + x], self._n
                        ).reshape(m.h, m.w)
        return self._park_dist

    @property
    def charger_dist(self) -> np.ndarray:
        if self._charger_dist is None:
            m = self.m
            chargers = [
                y * m.w + x
                for y in range(m.h)
                for x in range(m.w)
                if m.cell[y, x] in (HOME, SUPERCHARGER)
            ]
            self._charger_dist = _bfs_from(self._pred, chargers, self._n).reshape(
                m.h, m.w
            )
        return self._charger_dist

    def encode(
        self,
        x: int,
        y: int,
        battery: int,
        visited: int,
        max_battery: int,
        hints: bool = False,
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

        if hints:
            if m.nparks and unvisited.any():
                nearest = np.where(unvisited[:, None, None], self.park_dist, BIG).min(axis=0)
            else:
                nearest = np.zeros((m.h, m.w), dtype=np.int32)
            f[6] = np.clip(nearest / self.norm, 0.0, 1.0)
            f[7] = np.clip(self.charger_dist / self.norm, 0.0, 1.0)
        else:
            f[6] = 0.0
            f[7] = 0.0
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


# Planes 6 and 7 are hand-computed search results (BFS distances). They are off
# by default: the network gets local information only and must learn to
# navigate. `hints=True` turns the problem into distillation of the encoder's
# search, which is useful as a baseline but is not learned planning.
HINT_PLANES = (6, 7)


def encode(env, hints: bool = False) -> np.ndarray:
    mf = map_features(env.m)
    return mf.encode(
        env.x, env.y, env.battery, env.visited, env.m.max_battery, hints=hints
    )
