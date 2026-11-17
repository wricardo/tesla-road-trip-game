"""Compare saved checkpoints on the same maps, next to the solver's optimum.

    python -m tgame.benchmark runs/bc13.pt runs/value.pt --maps easy classic

Checkpoints carry their own kind: a policy net is played greedily from its
logits, a value net is played through beam search (tgame.search). Mixing them in
one table is the point -- it is how "policy with hints" and "value + search
without hints" get compared on identical maps.
"""
from __future__ import annotations

import argparse
import os

from .env import TeslaEnv
from .evaluate import run_episode
from .maps import load
from .net import load_any
from .planner import Oracle, TooComplex
from .search import play


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("models", nargs="+")
    ap.add_argument("--maps", nargs="*", default=None)
    ap.add_argument("--beam", type=int, default=8)
    ap.add_argument("--depth", type=int, default=10)
    args = ap.parse_args()

    maps = args.maps or [
        "easy",
        "easy_trapped_park",
        "coastal_crossing",
        "classic",
        "medium_maze",
    ]
    models = [(os.path.basename(p), *load_any(p)) for p in args.models]

    header = f"{'map':20s} {'optimal':>7s} " + " ".join(f"{n:>16s}" for n, *_ in models)
    print(header)
    print("-" * len(header))
    for name in maps:
        m = load(name)
        try:
            opt = Oracle(m).optimal_moves(TeslaEnv(m))
        except TooComplex:
            opt = -1
        cells = []
        for _n, kind, net, size, hints in models:
            if kind == "value":
                win, steps, score, reason = play(
                    net, m, size, "search", args.beam, args.depth, max_steps=400
                )
            else:
                win, steps, score, reason = run_episode(net, m, size, hints=hints)
            cells.append(f"{'WIN' if win else reason} {steps:3d} ({score}/{m.nparks})")
        print(f"{name:20s} {opt:7d} " + " ".join(f"{c:>16s}" for c in cells))


if __name__ == "__main__":
    main()
