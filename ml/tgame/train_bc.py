"""Behavior cloning, optionally followed by DAgger rounds.

    python -m tgame.train_bc --states 1500 --epochs 15 --dagger 3

Plain imitation learns from states the *solver* visits. Once the policy is
deployed it drifts into states the solver never produced and has no label for
them -- that is why 99% teacher-forced accuracy still loses games. DAgger fixes
it by relabelling the states the policy itself visits with the solver and
retraining, so the training distribution follows the policy.
"""
from __future__ import annotations

import argparse
import time

import numpy as np
import torch
import torch.nn.functional as F

from .evaluate import batch_accuracy, evaluate
from .maps import load
from .net import PolicyNet, masked_logits, save_policy
from .planner import Oracle
from .rollouts import TEST_MAPS, TRAIN_MAPS, collect, collect_from_policy, items_to_arrays


def train(net, opt, feats, masks, labels, size, epochs, batch, log_prefix="") -> None:
    n = len(feats)
    indices = np.arange(n)
    for epoch in range(epochs):
        net.train()
        np.random.shuffle(indices)
        total, correct, loss_sum = 0, 0, 0.0
        for i in range(0, n, batch):
            idx = indices[i : i + batch]
            x = torch.from_numpy(feats[idx]).float()
            msk = torch.from_numpy(masks[idx])
            y = torch.from_numpy(labels[idx])
            logits = masked_logits(net(x), msk)
            loss = F.cross_entropy(logits, y)
            opt.zero_grad()
            loss.backward()
            opt.step()
            loss_sum += float(loss.detach()) * len(idx)
            correct += int((logits.argmax(dim=1) == y).sum())
            total += len(idx)
        print(
            f"{log_prefix}epoch {epoch+1:2d}/{epochs}  loss={loss_sum/total:.3f}  "
            f"acc={correct/total:.3f}"
        )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--states", type=int, default=1500, help="labelled states per map")
    ap.add_argument("--epochs", type=int, default=15)
    ap.add_argument("--dagger", type=int, default=0, help="DAgger rounds after initial training")
    ap.add_argument("--dagger-epochs", type=int, default=5)
    ap.add_argument("--dagger-episodes", type=int, default=2, help="policy rollouts per map per round")
    ap.add_argument("--batch", type=int, default=64)
    ap.add_argument("--lr", type=float, default=1e-3)
    ap.add_argument("--width", type=int, default=32)
    ap.add_argument(
        "--dilations",
        default="1,2,4,8,1,1",
        help="comma-separated conv dilations; default receptive field ~35 cells",
    )
    ap.add_argument(
        "--hints",
        action="store_true",
        help="feed BFS distance planes (distillation baseline; NOT learned planning)",
    )
    ap.add_argument("--explore", type=float, default=0.2)
    ap.add_argument("--train-maps", nargs="*", default=TRAIN_MAPS)
    ap.add_argument("--test-maps", nargs="*", default=TEST_MAPS)
    ap.add_argument("--out", default="ml/runs/bc.pt")
    ap.add_argument("--seed", type=int, default=0)
    args = ap.parse_args()
    dilations = tuple(int(d) for d in args.dilations.split(","))

    torch.manual_seed(args.seed)
    t0 = time.time()
    feats, masks, labels, size, used = collect(
        args.train_maps, args.states, seed=args.seed, explore=args.explore,
        hints=args.hints,
    )
    print(
        f"dataset: {len(feats)} states from {len(used)} maps, "
        f"padded to {size}x{size}, hints={args.hints}, {time.time()-t0:.1f}s"
    )
    print("label distribution (up/down/left/right):", np.bincount(labels, minlength=4).tolist())

    net = PolicyNet(width=args.width, dilations=dilations)
    opt = torch.optim.Adam(net.parameters(), lr=args.lr)
    train(net, opt, feats, masks, labels, size, args.epochs, args.batch)
    print(f"teacher-forced accuracy: {batch_accuracy(net, feats, masks, labels, size):.3f}")
    print("held-out (before DAgger):", evaluate(net, args.test_maps, size, verbose=False, hints=args.hints)["win_rate"])

    oracles = {name: Oracle(load(name)) for name in args.train_maps}
    for rnd in range(1, args.dagger + 1):
        added = []
        for name in args.train_maps:
            m = load(name)
            added += collect_from_policy(
                net, m, size, oracles[name], episodes=args.dagger_episodes,
                hints=args.hints,
            )
        if not added:
            break
        af, am, al = items_to_arrays(added, size)
        feats = np.concatenate([feats, af])
        masks = np.concatenate([masks, am])
        labels = np.concatenate([labels, al])
        print(f"\nDAgger round {rnd}: +{len(added)} policy-visited states ({len(feats)} total)")
        train(net, opt, feats, masks, labels, size, args.dagger_epochs, args.batch, log_prefix="  ")
        print(
            f"  held-out after round {rnd}: "
            f"{evaluate(net, args.test_maps, size, verbose=False, hints=args.hints)['win_rate']:.2f}"
        )

    print(f"\nheld-out maps ({len(args.test_maps)}):")
    test = evaluate(net, args.test_maps, size, hints=args.hints)
    print(f"held-out win rate: {test['wins']}/{test['games']}")
    train_stats = evaluate(net, args.train_maps, size, verbose=False, hints=args.hints)
    print(f"training-map win rate: {train_stats['wins']}/{train_stats['games']}")

    save_policy(
        args.out,
        net,
        size,
        {
            "train_maps": used,
            "test_maps": args.test_maps,
            "test_win_rate": test["win_rate"],
            "train_win_rate": train_stats["win_rate"],
            "dagger_rounds": args.dagger,
            "hints": args.hints,
            "dilations": list(dilations),
        },
    )
    print(f"\nsaved {args.out} in {time.time()-t0:.0f}s")


if __name__ == "__main__":
    main()
