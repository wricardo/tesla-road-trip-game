"""Train V(s) without the hand-computed distance planes, then play with search.

    python -m tgame.train_value --states 2000 --epochs 12

Two evaluations are printed:
  * `one-step`  -- argmax V over legal successors (no lookahead)
  * `search`    -- beam search over V, depth 10, beam 8

If search beats one-step by a wide margin, the network learned an evaluation
and the search is doing the planning. That is the "learn the search" claim made
measurable instead of asserted.
"""
from __future__ import annotations

import argparse
import time

import numpy as np
import torch
import torch.nn.functional as F

from .env import TeslaEnv
from .maps import load
from .net import ValueNet, save_value
from .planner import Oracle, TooComplex
from .rollouts import TEST_MAPS
from .search import play
from .values import collect_values

EVAL_MAPS = TEST_MAPS + ["easy", "classic", "medium_maze"]


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--states", type=int, default=2000, help="states per map")
    ap.add_argument("--epochs", type=int, default=12)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--lr", type=float, default=1e-3)
    ap.add_argument("--width", type=int, default=32)
    ap.add_argument("--target", choices=("linear", "exp"), default="linear")
    ap.add_argument("--beam", type=int, default=8)
    ap.add_argument("--depth", type=int, default=10)
    ap.add_argument("--train-maps", nargs="*", default=None)
    ap.add_argument("--eval-maps", nargs="*", default=EVAL_MAPS)
    ap.add_argument("--out", default="ml/runs/value.pt")
    ap.add_argument("--seed", type=int, default=0)
    args = ap.parse_args()

    from .rollouts import TRAIN_MAPS

    train_maps = args.train_maps or TRAIN_MAPS
    torch.manual_seed(args.seed)
    t0 = time.time()

    feats, targets, size, used = collect_values(
        train_maps, args.states, seed=args.seed, target=args.target
    )
    print(
        f"dataset: {len(feats)} states from {len(used)} maps, padded to {size}x{size}, "
        f"mean target {targets.mean():.3f}, {time.time()-t0:.1f}s"
    )

    net = ValueNet(width=args.width)
    opt = torch.optim.Adam(net.parameters(), lr=args.lr)
    n = len(feats)
    idx = np.arange(n)
    for epoch in range(args.epochs):
        net.train()
        np.random.shuffle(idx)
        loss_sum = 0.0
        for i in range(0, n, args.batch):
            b = idx[i : i + args.batch]
            x = torch.from_numpy(feats[b]).float()
            y = torch.from_numpy(targets[b])
            pred = torch.sigmoid(net(x))
            loss = F.mse_loss(pred, y) if args.target == "linear" else F.binary_cross_entropy(pred, y)
            opt.zero_grad()
            loss.backward()
            opt.step()
            loss_sum += float(loss.detach()) * len(b)
        print(f"epoch {epoch+1:2d}/{args.epochs}  loss={loss_sum/n:.4f}  ({time.time()-t0:.0f}s)")

    print(f"\n{'map':20s} {'optimal':>7s} {'one-step':>22s} {'search':>22s}")
    print("-" * 74)
    for name in args.eval_maps:
        m = load(name)
        try:
            opt_moves = Oracle(m).optimal_moves(TeslaEnv(m))
        except TooComplex:
            opt_moves = -1
        w1, s1, sc1, r1 = play(net, m, size, "one-step", max_steps=400)
        w2, s2, sc2, r2 = play(net, m, size, "search", args.beam, args.depth, max_steps=400)
        f1 = f"{'WIN' if w1 else r1} {s1} ({sc1}/{m.nparks})"
        f2 = f"{'WIN' if w2 else r2} {s2} ({sc2}/{m.nparks})"
        print(f"{name:20s} {opt_moves:7d} {f1:>22s} {f2:>22s}")

    save_value(args.out, net, size, {"train_maps": used, "hints": False})
    print(f"\nsaved {args.out} in {time.time()-t0:.0f}s")


if __name__ == "__main__":
    main()
