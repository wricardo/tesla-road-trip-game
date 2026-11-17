"""Search with a learned value function instead of hand-computed distances.

`greedy_action` in net.py trusts planes 6-7, which are BFS distances computed by
the encoder. Here the encoder gets `hints=False` (those planes zeroed) and the
decision comes from a value network plus explicit lookahead:

    beam search: expand every legal child, score with V, keep the best `beam`,
                 repeat `depth` times, play the first move of the best line.

This is the AlphaZero shape minus self-play: a learned evaluation wrapped in a
search. With a perfect V, depth 1 is already optimal; the search exists to cover
what the network gets wrong.
"""
from __future__ import annotations

import numpy as np

from .features import encode
from .net import pad, value_batch


def _score(net, states, size: int, hints: bool) -> np.ndarray:
    feats = [pad(encode(s, hints=hints), size) for s in states]
    return value_batch(net, feats)


def one_step_action(net, env, size: int, hints: bool = False) -> int | None:
    """Best immediate successor by V. No lookahead beyond one move."""
    if env.done:
        return None
    children, actions = [], []
    for a in range(4):
        if not env.legal_mask()[a]:
            continue
        n = env.copy()
        n.step(a)
        if n.done and not n.win:
            continue
        if n.win:
            return a
        children.append(n)
        actions.append(a)
    if not children:
        return None
    vals = _score(net, children, size, hints)
    return actions[int(np.argmax(vals))]


def search_action(
    net,
    env,
    size: int,
    hints: bool = False,
    beam: int = 8,
    depth: int = 10,
) -> int | None:
    if env.done:
        return None
    # Exploring deeper than the remaining battery only walks into certain
    # death: every leaf would be a stranded terminal.
    depth = max(1, min(depth, env.battery))
    frontier: list[tuple[object, tuple[int, ...]]] = [(env, ())]
    best_path: tuple[int, ...] | None = None
    best_val = -1.0
    for _ in range(depth):
        seen: dict[tuple, int] = {}
        cand: list[tuple[object, tuple[int, ...]]] = []
        for state, path in frontier:
            for a in range(4):
                if not state.legal_mask()[a]:
                    continue
                n = state.copy()
                n.step(a)
                if n.done and not n.win:
                    continue
                npath = path + (a,)
                if n.win:
                    return npath[0]
                key = n.state_key()
                if key in seen:
                    continue
                seen[key] = len(cand)
                cand.append((n, npath))
        if not cand:
            break
        vals = _score(net, [c[0] for c in cand], size, hints)
        top = int(np.argmax(vals))
        if vals[top] > best_val:
            best_val = float(vals[top])
            best_path = cand[top][1]
        keep = np.argsort(-vals)[:beam]
        frontier = [cand[int(i)] for i in keep]
    if best_path is not None:
        return best_path[0]
    return None


def play(
    net,
    m,
    size: int,
    mode: str = "search",
    beam: int = 8,
    depth: int = 10,
    max_steps: int = 400,
    hints: bool = False,
) -> tuple[bool, int, int, str]:
    from .env import TeslaEnv

    env = TeslaEnv(m)
    steps = 0
    reason = "timeout"
    while not env.done and steps < max_steps:
        if mode == "search":
            a = search_action(net, env, size, hints, beam, depth)
        else:
            a = one_step_action(net, env, size, hints)
        if a is None:
            reason = "stuck"
            break
        _r, _done, info = env.step(a)
        steps += 1
        if env.win:
            reason = "win"
        elif info.crash:
            reason = "crash"
        elif info.stranded:
            reason = "stranded"
    return env.win, steps, env.score, reason
