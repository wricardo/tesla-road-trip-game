"""Compare saved checkpoints on the same maps, next to the solver's optimum.

    python -m tgame.benchmark ml/runs/bc.pt ml/runs/dqn.pt --maps easy coastal_crossing
"""
from __future__ import annotations

import argparse
import os

from .env import TeslaEnv
from .evaluate import run_episode
from .maps import load
from .net import load_policy
from .planner import Oracle, TooComplex


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("models", nargs="+")
    ap.add_argument("--maps", nargs="*", default=None)
    args = ap.parse_args()

    maps = args.maps or [
        "easy",
        "easy_trapped_park",
        "coastal_crossing",
        "classic",
        "medium_maze",
    ]
    models = [(os.path.basename(p), *load_policy(p)) for p in args.models]

    header = f"{'map':20s} {'optimal':>7s} " + " ".join(f"{n:>16s}" for n, _a, _b in models)
    print(header)
    print("-" * len(header))
    for name in maps:
        m = load(name)
        try:
            opt = Oracle(m).optimal_moves(TeslaEnv(m))
        except TooComplex:
            opt = -1
        cells = []
        for _n, net, size in models:
            win, steps, score, reason = run_episode(net, m, size)
            cells.append(f"{'WIN' if win else reason} {steps:3d} ({score}/{m.nparks})")
        print(f"{name:20s} {opt:7d} " + " ".join(f"{c:>16s}" for c in cells))


if __name__ == "__main__":
    main()
