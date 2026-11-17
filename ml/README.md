# Teaching a model to play Tesla Road Trip

This directory is a working, end-to-end example of the loop you see in "I trained
a model to do X" videos: define the task as a state machine, build a simulator,
produce a signal, optimise parameters, evaluate honestly.

Nothing here is a toy stand-in for the game. The simulator in `tgame/env.py`
reproduces `game/engine/movement.go` rule for rule, and `tgame/verify.py` proves
it by replaying random traces through the real Go engine (`cmd/envcheck`) and
diffing every field. If the two ever disagree, the Python env is wrong.

## 0. Setup

```console
$ python3 -m venv .venv-ml && .venv-ml/bin/pip install torch numpy   # ~2 min
$ cd ml && ../.venv-ml/bin/python -m tgame.verify                    # prove the sim matches the Go engine
```

Then `python -m tgame.train_bc` (10 min on CPU), `python -m tgame.train_dqn`,
`python -m tgame.benchmark runs/*.pt`, `python -m tgame.play_live`.

## 1. The vocabulary

* **Policy** `π(s) -> a`: a function from game state to action. A neural network
  with parameters `θ` is just a policy with a lot of knobs.
* **Training** means choosing `θ` to make `π` better at something measurable.
  The whole discipline is deciding *what* "better" is and *how* you know.

Two signals do that here:

| Approach | Signal | What the model learns | Cost |
|---|---|---|---|
| **Behavior cloning (BC)** | an expert action per state | "do what the solver does" | needs a solver, one pass, stable |
| **Deep Q-learning (DQN)** | reward from playing | "take actions that lead to reward" | no solver, many episodes, unstable |

The game is a small deterministic planning problem. That makes the solver cheap
and the reward signal sparse (you only win at the end of a 20–120 move episode),
so BC is the strong baseline and DQN is the interesting contrast.

## 2. The game as an MDP

| Piece | In this game |
|---|---|
| state | car cell, battery, which parks are visited, grid |
| action | `up`, `down`, `left`, `right` |
| transition | move costs 1 battery; `H`/`S` refill to max; parks get marked visited |
| reward | +1 per new park, +10 victory, -5 death |
| terminal | all parks visited (win), battery 0 away from a charger (loss), **moving into water/building/off-grid (loss)** |

Four details the sim encodes that a naive implementation gets wrong:

1. **Walls kill.** `CanMoveTo` failure sets `GameOver` — a wall is not "no move".
   The policy therefore gets an **action mask**: only moves that do not lose the
   game are candidates. This is the single biggest stability trick in the code.
2. **Home/supercharger refill on arrival**, after the battery decrement.
3. **Victory is checked before the stranded check**, so arriving on the last park
   with 0 battery is a win, not a loss.
4. **Multiple `H` cells**: the engine starts the player on the *last* one in scan
   order. `tgame/maps.py` reproduces that quirk; otherwise every trace diverges.

## 3. Representation

A 3×3 CNN cannot see 15 cells down a corridor, and battery planning is exactly a
long-range problem. Feeding only raw cell types forces the network to relearn
pathfinding from scratch, which it will not do from a few thousand samples.

So `tgame/features.py` builds 13 planes from the current state:

| # | plane | why |
|---|---|---|
| 0 | passable | walls |
| 1,2 | home, supercharger | where refills are |
| 3,4 | unvisited / visited park | objective set |
| 5 | car position | the readout location |
| 6 | steps to nearest unvisited park | "which way is progress" |
| 7 | steps to nearest charger | "can I get a refill" |
| 8 | battery / max | fuel |
| 9–12 | exit blocked per direction (N,S,W,E) | one-way roads |

Planes 6 and 7 are BFS distance transforms on the *legal move graph* (direction
restrictions included). A human navigator computes these too; they are not a
sneak peek at the answer — no target park, no planned route.

Planes 9–12 expose the direction restrictions themselves. Only one map
(`one_way_city`) has them, and the first version of the encoder omitted them — a
policy cannot plan around a one-way cell it cannot see, which is why that map is
in the training set rather than the held-out set.

The network (`tgame/net.py`) is three 3×3 convolutions then a 1×1 head that emits
four logits, read off the car's cell. Fully convolutional, so it is
translation-equivariant and works on any map size; maps are zero-padded to a
common square for batching.

## 4. The expert: exact solver

`tgame/planner.py` runs a **retrograde BFS**: every state is
`(cell, battery, visited-park bitmask)`, all moves cost 1, so seeding the all-parks
visited states and expanding predecessors gives the exact minimum number of moves
to win from *every* reachable state.

Verification that it is exact: follow `best_action` greedily from the start and
the game is won in exactly `optimal_moves` steps on every map (21 on `easy`,
124 on `classic`, 86 on `medium_maze`).

Two subtle rules that cost a debugging session:

* Arriving on a charger refills to max, so the arrival battery carries no
  information — every non-zero predecessor battery is a valid reverse edge.
* Only some `(cell, battery)` pairs can exist: leaving a non-charger always costs
  a unit, so full battery only occurs *on* a charger. Without that filter the
  reverse search invents "road with a full battery" states and their distances
  leak into real states, producing a solver that "wins" in 119 moves on a map
  that really needs 124.


The resulting map zoo (optimal = minimum moves from the start to all parks):

| map | grid | battery | parks | state space | optimal | solve |
|---|---|---|---|---|---|---|
| coastal_crossing | 14x14 | 18 | 3 | 27k | 14 | 0.01s |
| easy | 10x10 | 15 | 4 | 26k | 21 | 0.03s |
| easy_highway | 12x12 | 18 | 2 | 9k | 21 | 0.01s |
| keys_run | 20x20 | 22 | 4 | 74k | 24 | 0.02s |
| one_way_city | 12x12 | 25 | 2 | 7k | 24 | 0.00s |
| easy_trapped_park | 10x10 | 15 | 4 | 26k | 25 | 0.03s |
| bayou_braids | 16x16 | 20 | 5 | 185k | 31 | 0.04s |
| stella_map | 16x16 | 25 | 10 | 6.8M | 32 | 1.6s |
| easy_gardens | 12x12 | 15 | 7 | 295k | 43 | 0.20s |
| easy_circuit | 14x14 | 18 | 7 | 477k | 49 | 0.25s |
| sarah_map | 15x15 | 20 | 10 | 4.8M | 55 | 3.1s |
| medium_maze | 16x16 | 22 | 5 | 188k | 86 | 0.12s |
| classic | 15x15 | 20 | 10 | 4.8M | 124 | 3.9s |
| local_hard_50 | 50x50 | 50 | 24 | 1.7e13 | — | too big |

Complexity is `cells × battery × 2^parks`. `classic` (10 parks) is 4.8M states and
takes ~3s; `local_hard_50` (24 parks, 50×50) is 10^13 states and is rejected.

## 5. Behavior cloning

`tgame/train_bc.py`:

1. Roll out `explore`-greedy episodes (20% random legal moves). Random steps drag
   the agent into states a pure expert would never visit.
2. Label every visited state with the solver's best action *from that state*.
3. Minimise masked cross-entropy between the 4 logits and that label.

Why label states and not actions-taken: that is the difference between imitating
a trajectory and imitating the policy. It is also one step away from DAgger —
relabel the states your *own* policy visits, retrain, repeat.

## 6. DQN

`tgame/train_dqn.py` is the same network trained as `Q(s,a)`:

* replay buffer of 50k transitions,
* ε-greedy exploration 1.0 → 0.05,
* frozen target network refreshed every 500 steps,
* Huber loss on `r + γ·max Q(s')`, with terminal states bootstrapped to zero,
* action masking at every argmax and at the bootstrap.

Two game-specific tweaks: `-0.01` per step (otherwise "drive in circles" has the
same value as "go to a park" until the first win ever happens) and masking
(because an untrained network's first instinct is to drive into a wall and end
the episode).

## 7. Evaluation

Teacher-forced accuracy ("does it match the solver on states it was trained on")
flatters every imitation model: one wrong turn puts the car in a state the
training distribution barely covers, and errors compound. The metric that matters
is **full-game win rate on maps never trained on**, plus moves-to-win against the
solver's optimum.

```console
$ cd ml && ../.venv-ml/bin/python -m tgame.train_bc
$ ../.venv-ml/bin/python -m tgame.train_dqn
$ ../.venv-ml/bin/python -m tgame.verify          # env vs Go engine
```

Watch it play the real server:

```console
$ (cd .. && ./tesla-road-trip -port 9191 &)
$ ../.venv-ml/bin/python -m tgame.play_live --map easy --url http://localhost:9191
```

## 8. Results

Numbers from the runs saved in `ml/runs/` (gitignored; CPU only, ~10 minutes of
training per model). "Optimal" is the solver's move count.

| model | signal | labels | teacher-forced acc | training maps | held-out maps |
|---|---|---|---|---|---|
| BC, 9 planes (first attempt) | solver actions | 28,500 | 99.0% | 17/19 | 3/5 |
| **BC, 13 planes + DAgger** | solver actions | 41,690 | 99.3% | **20/20** | **3/4** |
| DQN | reward only (12k steps) | — | — | 3/20 | 0/4 |

Per map, `runs/bc13.pt` against the optimum:

| map | optimal | result |
|---|---|---|
| easy | 21 | win, 21 moves |
| coastal_crossing | 14 | win, 14 |
| stella_map | 32 | win, 32 |
| one_way_city | 24 | win, 24 |
| classic | 124 | win, 124 |
| easy_circuit | 49 | win, 49 |
| easy_trapped_park | 25 | win, 27 |
| medium_maze | 86 | win, 100 |
| sarah_map | 55 | **timeout, 3/10 parks** |

What the numbers say:

* **A wall was never hit.** The action mask makes the most common failure mode
  impossible; every remaining loss is battery management or a loop.
* **99% accuracy is not 99% wins.** One wrong turn leaves the policy in states
  the solver never visited, and the error compounds over 20–120 moves.
* **DAgger is what closed the gap** (held-out 2/4 → 3/4, and all 20 training
  maps won afterwards) by adding exactly the states the policy gets wrong.
* **The representation matters as much as the algorithm.** The first encoder had
  no planes for direction restrictions, so the policy was blind to one-way cells
  — information it cannot see, no amount of training recovers. Planes 9–12 fix
  that, and `one_way_city` now wins in the optimal 24 moves.
* **`sarah_map` shows the honest limit**: a deterministic Markov policy can
  enter a cycle (it wanders with 3/10 parks forever). The solver cannot, because
  it carries an exact distance for every state.
* **DQN is far behind.** 12k steps (~4 min) won 16/113 training episodes while
  still exploring, but the greedy policy wins only 3/20 training maps and 0/4
  held-out: with a sparse win signal and 20–120 move episodes, a replay buffer
  of 12k transitions is not enough to learn credit assignment that the solver
  hands over for free inside its labels.

The network has 59,460 parameters (~230 KB) and is *not* doing search: it reads
the BFS distance planes and applies a learned rule. That is the honest
description of what was trained -- and it drives the real server to a 124-move
optimal victory on `classic`:

```console
$ ../.venv-ml/bin/python -m tgame.play_live --map classic --model runs/bc13.pt
...
124 moves, score=10, victory=True, gameOver=True -- Victory! All 10 parks visited!
```

## 9. Where to go next

* **More DAgger**: more rounds, more rollouts per round, and relabel visits to
  the held-out maps too if you are willing to give up a clean test set.
* **AlphaZero-style search**: the solver already gives exact distances; a learned
  value function could replace the BFS for maps too large to enumerate.
* **Fog**: `GameState.FogEnabled` turns this from a planning problem into a POMDP —
  the network would need a memory (RNN or frame stack) because the map is hidden.
* **Big maps**: `local_hard_50` (24 parks) is out of reach for the exact solver;
  that is where learning stops being a demo and starts being necessary.
