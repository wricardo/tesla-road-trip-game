"""Dataset generation with exact-solver labels.

One optimal path only visits states on that path. Rollouts here are
`explore`-greedy: mostly the oracle's move, sometimes a random legal one. Every
visited state is labelled with the oracle's best action *from that state*, so
the dataset imitates the planner (not one trajectory) and covers the states an
agent actually wanders into.
"""
from __future__ import annotations

import random

import numpy as np

from .env import TeslaEnv
from .features import encode

TRAIN_MAPS = [
    "easy",
    "easy_suburban",
    "easy_gardens",
    "easy_circuit",
    "easy_highway",
    "classic",
    "medium_maze",
    "medium_island",
    "medium_downtown",
    "harbor_hop",
    "peninsula_loop",
    "switchback_ridge",
    "metro_ring",
    "everglades_fork",
    "keys_run",
    "hub_maze",
    "island_chain",
    "bayou_braids",
    "strategic",
    "one_way_city",
]

TEST_MAPS = [
    "easy_trapped_park",
    "coastal_crossing",
    "sarah_map",
    "stella_map",
]


def collect(
    map_names: list[str],
    states_per_map: int = 1500,
    seed: int = 0,
    explore: float = 0.2,
    max_extra: int = 60,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, int, list[str]]:
    """Returns (features, masks, labels, size, maps_used).

    features: (N, N_CHANNELS, size, size) float16, masks: (N,4) bool, labels: (N,) int64.
    """
    from .maps import load
    from .planner import Oracle

    rng = random.Random(seed)
    feats, masks, labels, used = [], [], [], []
    size = 0

    for name in map_names:
        m = load(name)
        oracle = Oracle(m)                     # raises TooComplex for huge maps
        size = max(size, m.h, m.w)
        got = 0
        rounds = 0
        while got < states_per_map and rounds < states_per_map * 8:
            rounds += 1
            env = TeslaEnv(m)
            horizon = max(200, 4 * (oracle.optimal_moves(env) + 1))
            for _ in range(horizon):
                if env.done:
                    break
                legal = env.legal_mask()
                best = oracle.best_action(env)
                if best is None:
                    break
                feats.append(encode(env))
                masks.append(legal)
                labels.append(best)
                got += 1
                if got >= states_per_map:
                    break
                if rng.random() < explore:
                    choices = [a for a in range(4) if legal[a]]
                    env.step(rng.choice(choices))
                else:
                    env.step(best)
        if got:
            used.append(name)

    if not feats:
        raise RuntimeError("no labelled states collected")

    f = np.zeros((len(feats), feats[0].shape[0], size, size), dtype=np.float16)
    for i, x in enumerate(feats):
        f[i, :, : x.shape[1], : x.shape[2]] = x
    return f, np.array(masks), np.array(labels, dtype=np.int64), size, used


def items_to_arrays(items, size: int):
    """(features, masks, labels) from a list of (feature, mask, label) tuples."""
    f = np.zeros((len(items), items[0][0].shape[0], size, size), dtype=np.float16)
    for i, (x, _m, _l) in enumerate(items):
        f[i, :, : x.shape[1], : x.shape[2]] = x
    return f, np.array([m for _f, m, _l in items]), np.array([l for _f, _m, l in items], dtype=np.int64)


def collect_from_policy(net, m, size: int, oracle, episodes: int = 3, max_steps: int = 600):
    """DAgger step: states the *current policy* visits, labelled by the solver."""
    from .net import greedy_action

    items = []
    for _ in range(episodes):
        env = TeslaEnv(m)
        for _step in range(max_steps):
            if env.done:
                break
            best = oracle.best_action(env)
            if best is None:
                break
            items.append((encode(env), env.legal_mask(), best))
            env.step(greedy_action(net, env, size))
    return items
