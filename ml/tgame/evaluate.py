"""Roll the trained policy through whole games and report wins vs optimal."""
from __future__ import annotations

import numpy as np
import torch

from .env import TeslaEnv
from .maps import load
from .net import PolicyNet, greedy_action, masked_logits, pad
from .features import encode
from .planner import Oracle, TooComplex


def run_episode(
    net: PolicyNet, m, size: int, max_steps: int = 400, hints: bool = False
) -> tuple[bool, int, int, str]:
    """Returns (win, steps, score, how it ended). How it ended matters: a crash
    and a stranding are different mistakes and need different fixes."""
    env = TeslaEnv(m)
    steps = 0
    reason = "timeout"
    while not env.done and steps < max_steps:
        a = greedy_action(net, env, size, hints=hints)
        _r, _done, info = env.step(a)
        steps += 1
        if env.win:
            reason = "win"
        elif info.crash:
            reason = "crash"
        elif info.stranded:
            reason = "stranded"
    return env.win, steps, env.score, reason


def evaluate(
    net: PolicyNet, maps: list[str], size: int, verbose: bool = True,
    hints: bool = False,
) -> dict:
    rows = []
    for name in maps:
        m = load(name)
        try:
            opt = Oracle(m).optimal_moves(TeslaEnv(m))
        except TooComplex:
            opt = -1
        win, steps, score, reason = run_episode(net, m, size, hints=hints)
        rows.append((name, win, steps, opt, score, m.nparks))
        if verbose:
            tag = "WIN " if win else "LOSS"
            print(
                f"  {tag} {name:20s} steps={steps:4d} optimal={opt:4d} "
                f"parks={score}/{m.nparks} ended={reason}"
            )
    wins = sum(r[1] for r in rows)
    return {
        "wins": wins,
        "games": len(rows),
        "win_rate": wins / max(len(rows), 1),
        "rows": rows,
    }


def batch_accuracy(net: PolicyNet, feats, masks, labels, size: int, batch: int = 256) -> float:
    """Teacher-forced accuracy: how often the policy picks the oracle's move."""
    net.eval()
    correct = 0
    with torch.no_grad():
        for i in range(0, len(feats), batch):
            x = torch.from_numpy(feats[i : i + batch]).float()
            y = torch.from_numpy(labels[i : i + batch])
            msk = torch.from_numpy(masks[i : i + batch])
            pred = masked_logits(net(x), msk).argmax(dim=1)
            correct += int((pred == y).sum())
    return correct / len(feats)
