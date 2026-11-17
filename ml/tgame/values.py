"""Value targets: v(s) = exp(-optimal_moves_to_win / TAU).

The target still comes from the exact solver, but the *model* only sees local
information -- planes 6-7 (BFS distances) are zeroed. So the network has to
compress the solver's global knowledge into its weights, and the search in
search.py decompresses it at play time.

Dead ends matter: `d < 0` means the solver proved no win from there, and it gets
target 0 rather than being skipped. A value function that has never seen a dead
end cannot teach a search to avoid one.
"""
from __future__ import annotations

import math
import random

import numpy as np

from .env import TeslaEnv
from .features import N_CHANNELS, encode

TAU = 20.0
# Linear targets keep 1-move differences visible everywhere: exp(-d/20) maps
# distances 60 and 61 to 0.0498 and 0.0474, which a sigmoid simply cannot
# resolve, and most of a 124-move map lives out there. D_MAX is the horizon the
# value is allowed to distinguish.
D_MAX = 150.0


def target_of(distance: int, mode: str = "linear") -> float:
    if distance < 0:
        return 0.0
    if mode == "exp":
        return math.exp(-distance / TAU)
    return max(0.0, 1.0 - distance / D_MAX)


def collect_values(
    map_names: list[str],
    states_per_map: int = 2000,
    seed: int = 0,
    explore: float = 0.3,
    target: str = "linear",
) -> tuple[np.ndarray, np.ndarray, int, list[str]]:
    """Returns (features with hints=False, targets, padded size, maps used).

    The array is preallocated: a 40k-state list of float32 planes costs ~1 GB
    before it is converted, which is enough to get the process OOM-killed.
    """
    from .maps import load
    from .planner import Oracle

    rng = random.Random(seed)
    maps = [load(n) for n in map_names]
    size = max(max(m.h, m.w) for m in maps)
    cap = states_per_map * len(maps)
    feats = np.zeros((cap, N_CHANNELS, size, size), dtype=np.float16)
    targets = np.zeros(cap, dtype=np.float32)
    used: list[str] = []
    k = 0

    for m in maps:
        oracle = Oracle(m)
        got = 0
        rounds = 0
        while got < states_per_map and rounds < states_per_map and k < cap:
            rounds += 1
            env = TeslaEnv(m)
            horizon = max(200, 4 * (oracle.optimal_moves(env) + 1))
            for _ in range(horizon):
                if env.done or k >= cap:
                    break
                d = oracle.distance(env.x, env.y, env.battery, env.visited)
                feats[k, :, : m.h, : m.w] = encode(env, hints=False)
                targets[k] = target_of(d, target)
                k += 1
                got += 1
                if got >= states_per_map or d < 0:
                    break
                legal = [a for a in range(4) if env.legal_mask()[a]]
                best = oracle.best_action(env)
                if not legal or best is None:
                    break
                env.step(rng.choice(legal) if rng.random() < explore else best)
        if got:
            used.append(m.name)

    if k == 0:
        raise RuntimeError("no value samples collected")
    return feats[:k], targets[:k], size, used
