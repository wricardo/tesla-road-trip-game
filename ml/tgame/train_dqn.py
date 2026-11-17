"""DQN: learn from reward alone, no solver labels.

    python -m tgame.train_dqn --steps 60000

This is the classic value-based RL loop -- replay buffer, epsilon-greedy
exploration, frozen target network -- with two modifications that the game
forces:

  * action masking, because a wall move is an instant loss and an untrained
    network would otherwise kill itself constantly;
  * a small per-step penalty, because the only intrinsic cost (battery) is
    invisible until it strands you a hundred moves later.

Compare its win rate with train_bc.py: on a deterministic planning problem,
supervised imitation of a search is far more sample-efficient than Q-learning.
"""
from __future__ import annotations

import argparse
import random
import time
from collections import deque

import numpy as np
import torch
import torch.nn.functional as F

from .env import TeslaEnv
from .evaluate import evaluate
from .features import N_CHANNELS, encode
from .maps import load
from .net import PolicyNet, masked_logits, pad, save_policy
from .rollouts import TEST_MAPS, TRAIN_MAPS


class Replay:
    def __init__(self, capacity: int, size: int):
        self.cap = capacity
        self.size = size
        self.f = np.zeros((capacity, N_CHANNELS, size, size), dtype=np.float16)
        self.m = np.zeros((capacity, 4), dtype=bool)
        self.a = np.zeros(capacity, dtype=np.int64)
        self.r = np.zeros(capacity, dtype=np.float32)
        self.f2 = np.zeros((capacity, N_CHANNELS, size, size), dtype=np.float16)
        self.m2 = np.zeros((capacity, 4), dtype=bool)
        self.done = np.zeros(capacity, dtype=bool)
        self.i = 0
        self.n = 0

    def add(self, f, m, a, r, f2, m2, done) -> None:
        i = self.i
        self.f[i], self.m[i], self.a[i], self.r[i] = f, m, a, r
        self.f2[i], self.m2[i], self.done[i] = f2, m2, done
        self.i = (i + 1) % self.cap
        self.n = min(self.n + 1, self.cap)

    def sample(self, batch: int, rng):
        idx = rng.integers(0, self.n, size=batch)
        return (
            torch.from_numpy(self.f[idx]).float(),
            torch.from_numpy(self.m[idx]),
            torch.from_numpy(self.a[idx]),
            torch.from_numpy(self.r[idx]),
            torch.from_numpy(self.f2[idx]).float(),
            torch.from_numpy(self.m2[idx]),
            torch.from_numpy(self.done[idx]),
        )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--steps", type=int, default=60000)
    ap.add_argument("--batch", type=int, default=64)
    ap.add_argument("--lr", type=float, default=2e-4)
    ap.add_argument("--gamma", type=float, default=0.99)
    ap.add_argument("--eps-start", type=float, default=1.0)
    ap.add_argument("--eps-end", type=float, default=0.05)
    ap.add_argument("--target-every", type=int, default=500)
    ap.add_argument("--step-penalty", type=float, default=0.01)
    ap.add_argument("--width", type=int, default=32)
    ap.add_argument("--train-maps", nargs="*", default=TRAIN_MAPS)
    ap.add_argument("--test-maps", nargs="*", default=TEST_MAPS)
    ap.add_argument("--out", default="ml/runs/dqn.pt")
    ap.add_argument("--seed", type=int, default=0)
    args = ap.parse_args()

    torch.manual_seed(args.seed)
    rng = np.random.default_rng(args.seed)
    py_rng = random.Random(args.seed)

    maps = [load(n) for n in args.train_maps]
    size = max(max(m.h, m.w) for m in maps)
    net = PolicyNet(width=args.width)
    target = PolicyNet(width=args.width)
    target.load_state_dict(net.state_dict())
    opt = torch.optim.Adam(net.parameters(), lr=args.lr)
    replay = Replay(min(args.steps, 50000), size)

    env = TeslaEnv(py_rng.choice(maps))
    f = pad(encode(env), size)
    t0 = time.time()
    episode_reward = 0.0
    wins = episodes = 0

    for step in range(1, args.steps + 1):
        mask = env.legal_mask()
        eps = args.eps_start + (args.eps_end - args.eps_start) * min(1.0, step / (0.6 * args.steps))
        if py_rng.random() < eps:
            choices = [a for a in range(4) if mask[a]]
            action = py_rng.choice(choices)
        else:
            with torch.no_grad():
                x = torch.from_numpy(f).float().unsqueeze(0)
                logits = masked_logits(net(x), torch.from_numpy(mask).unsqueeze(0))
                action = int(logits.argmax(dim=1).item())

        reward, done, _info = env.step(action)
        reward -= args.step_penalty
        episode_reward += reward
        f2 = pad(encode(env), size) if not done else np.zeros_like(f)
        replay.add(f, mask, action, reward, f2, env.legal_mask(), done)

        if done:
            episodes += 1
            wins += int(env.win)
            env = TeslaEnv(py_rng.choice(maps))
            f = pad(encode(env), size)
            episode_reward = 0.0
        else:
            f = f2

        if replay.n >= 2000 and step % 2 == 0:
            bf, bm, ba, br, bf2, bm2, bd = replay.sample(args.batch, rng)
            with torch.no_grad():
                q_next = masked_logits(target(bf2), bm2).max(dim=1).values
                y = br + args.gamma * q_next * (~bd)
            q = masked_logits(net(bf), bm).gather(1, ba[:, None]).squeeze(1)
            loss = F.smooth_l1_loss(q, y)
            opt.zero_grad()
            loss.backward()
            opt.step()

        if step % args.target_every == 0:
            target.load_state_dict(net.state_dict())

        if step % 5000 == 0:
            print(
                f"step {step:6d}  eps={eps:.2f}  loss={float(loss.detach()):.3f}  "
                f"episodes={episodes}  wins={wins}  ({time.time()-t0:.0f}s)"
            )

    print(f"\ntrained {args.steps} steps in {time.time()-t0:.0f}s, "
          f"training-episode wins {wins}/{episodes}")
    print("held-out maps:")
    test = evaluate(net, args.test_maps, size)
    print(f"held-out win rate: {test['wins']}/{test['games']}")
    train = evaluate(net, args.train_maps, size, verbose=False)
    print(f"training-map win rate: {train['wins']}/{train['games']}")
    save_policy(
        args.out,
        net,
        size,
        {"test_win_rate": test["win_rate"], "train_win_rate": train["win_rate"]},
    )
    print(f"saved {args.out}")


if __name__ == "__main__":
    main()
