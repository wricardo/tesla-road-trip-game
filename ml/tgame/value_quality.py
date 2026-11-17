"""How good is a value network, independent of the search that uses it?

    python -m tgame.value_quality --model runs/value.pt --maps easy classic

Two numbers per map, measured on states a solver-guided walk visits:

* `rank-acc` -- fraction of states where argmax V over the legal successors is
  the solver's optimal move. 0.25 is chance (4 actions); a high correlation with
  low rank-acc means V is smooth but locally flat.
* `corr` -- Pearson correlation between V and exp(-d/20), the target shape.

Separating these from win rate matters: search can rescue a weak V (it did on
stella_map) and can also bounce around inside a flat V (it did on classic).
"""
from __future__ import annotations

import argparse
import math
import random

import numpy as np

from .env import TeslaEnv
from .features import encode
from .maps import load
from .net import load_value, pad, value_batch
from .planner import Oracle


def diagnose(net, size: int, name: str, samples: int = 300, seed: int = 0) -> tuple[float, float]:
    m = load(name)
    oracle = Oracle(m)
    rng = random.Random(seed)
    env = TeslaEnv(m)
    match = total = 0
    preds, targets = [], []
    for _ in range(samples):
        if env.done:
            env = TeslaEnv(m)
        best = oracle.best_action(env)
        kids = []
        for a in range(4):
            if not env.legal_mask()[a]:
                continue
            n = env.copy()
            n.step(a)
            if n.done and not n.win:
                continue
            kids.append((a, n))
        if kids and best is not None:
            vals = value_batch(net, [pad(encode(n, hints=False), size) for _a, n in kids])
            arg = [a for a, _n in kids][int(np.argmax(vals))]
            match += int(arg == best)
            total += 1
            for (_a, n), v in zip(kids, vals):
                d = oracle.distance(n.x, n.y, n.battery, n.visited)
                if d >= 0:
                    preds.append(float(v))
                    targets.append(math.exp(-d / 20.0))
        move = best if best is not None else rng.randrange(4)
        if env.legal_mask()[move]:
            env.step(move)
    corr = float(np.corrcoef(preds, targets)[0, 1]) if len(preds) > 10 else float("nan")
    return match / max(total, 1), corr


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="ml/runs/value.pt")
    ap.add_argument("--maps", nargs="*",
                    default=["easy", "coastal_crossing", "stella_map", "classic",
                             "medium_maze", "sarah_map"])
    ap.add_argument("--samples", type=int, default=300)
    args = ap.parse_args()

    net, size = load_value(args.model)
    print(f"{'map':18s} {'rank-acc':>8s} {'corr':>6s}")
    for name in args.maps:
        acc, corr = diagnose(net, size, name, args.samples)
        print(f"{name:18s} {acc:8.2f} {corr:6.2f}")


if __name__ == "__main__":
    main()
