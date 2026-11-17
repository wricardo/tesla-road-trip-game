"""Differential test: our Python env vs the real Go engine.

Random traces (including deliberate crashes and collisions) are replayed by
`go run ./cmd/envcheck`, which uses engine.MovePlayer itself. Any divergence in
position, battery, score, visited parks or terminal flag is a bug in the
Python env and is reported with the offending action prefix.
"""
from __future__ import annotations

import json
import random
import subprocess
import sys
from pathlib import Path

from .env import TeslaEnv
from .maps import ACTIONS, maps_dir, repo_root

TRACE_COUNT = 40
MAX_STEPS = 60


def _snapshot(env: TeslaEnv) -> dict:
    return {
        "x": env.x,
        "y": env.y,
        "battery": env.battery,
        "score": env.score,
        "game_over": env.done,
        "victory": env.win,
        "visited": sorted(
            f"park_{pid}" for pid in range(env.m.nparks) if env.visited >> pid & 1
        ),
    }


def build_traces(names: list[str], seed: int = 0) -> tuple[list[dict], list[list[dict]]]:
    rng = random.Random(seed)
    traces, expected = [], []
    per_map = max(1, TRACE_COUNT // len(names))
    for name in names:
        from .maps import load

        for _ in range(per_map):
            env = TeslaEnv(load(name))
            actions = []
            states = [_snapshot(env)]
            for _step in range(MAX_STEPS):
                if env.done:
                    break
                if rng.random() < 0.15:
                    action = rng.randrange(4)   # may be a wall move (instant loss)
                else:
                    legal = [a for a in range(4) if env.legal_mask()[a]]
                    if not legal:
                        break
                    action = rng.choice(legal)
                wall = _is_wall(env, action)
                actions.append(ACTIONS[action][3])
                env.step(action)
                states.append(_snapshot(env))
                if wall or env.done:
                    break
            traces.append({"map_file": str(maps_dir() / f"{name}.json"), "actions": actions})
            expected.append(states)
    return traces, expected


def _is_wall(env: TeslaEnv, action: int) -> bool:
    dx, dy, _bit, _ = ACTIONS[action]
    nx, ny = env.x + dx, env.y + dy
    if not (0 <= nx < env.m.w and 0 <= ny < env.m.h):
        return True
    return int(env.m.cell[ny, nx]) in (4, 5)  # WATER, BUILDING


def compare(names: list[str], seed: int = 0) -> int:
    traces, expected = build_traces(names, seed)
    proc = subprocess.run(
        ["go", "run", "./cmd/envcheck"],
        cwd=repo_root(),
        input=json.dumps(traces),
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        print(proc.stderr[-4000:], file=sys.stderr)
        raise SystemExit("envcheck failed")
    got = json.loads(proc.stdout)

    failures = 0
    checked = 0
    for i, (exp, res) in enumerate(zip(expected, got)):
        if res.get("error"):
            print(f"trace {i}: engine error: {res['error']}")
            failures += 1
            continue
        go_states = res["states"]
        for j, (a, b) in enumerate(zip(exp, go_states)):
            checked += 1
            if a != b:
                failures += 1
                print(f"MISMATCH trace {i} step {j} map={traces[i]['map_file']}")
                print(f"  python: {a}")
                print(f"  go    : {b}")
                print(f"  actions: {traces[i]['actions'][:j]}")
                break
        if len(exp) != len(go_states):
            failures += 1
            print(f"MISMATCH trace {i}: {len(exp)} python states vs {len(go_states)} go states "
                  f"(map {traces[i]['map_file']})")
    print(f"compared {checked} states across {len(traces)} traces; failures={failures}")
    return failures


if __name__ == "__main__":
    maps = sys.argv[1:] or ["easy", "classic", "easy_circuit"]
    raise SystemExit(1 if compare(maps) else 0)
