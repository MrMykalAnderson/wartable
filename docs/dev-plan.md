# Wartable Development Plan

Instructions for Claude Code (and any other developer) building the Wartable engine and web tools.

## 0. Getting started

The repository is `MrMykalAnderson/wartable` on GitHub. Locally it lives at `~/Projects/wartable`.

Before starting any work session:

```bash
cd ~/Projects/wartable
git pull
```

If the folder doesn't exist yet:

```bash
mkdir -p ~/Projects
git clone https://github.com/MrMykalAnderson/wartable.git ~/Projects/wartable
cd ~/Projects/wartable
```

Work on a branch per milestone (e.g. `m1-hex`), commit often, and push when a milestone's tests pass.

## 1. Sources of truth

| Document | Owns |
| --- | --- |
| [core-rules.md](core-rules.md) | Every game rule and every number. |
| [starter-battle.md](starter-battle.md) | The MVP scenario: map, deployment, armies, win conditions. |
| This file | How the software is built. It does **not** restate rules. |

Rules for working with them:

- **Never invent a rule silently.** If the rules are ambiguous or don't cover a case, implement the simplest reading, mark it in code with `// RULES-QUESTION:`, and add it to section 12 of core-rules.md under "Open design questions". Mention it in your summary.
- **Every number lives in data**, not code (see section 3). The data files must match the tables in the rules docs exactly. If a rule's number changes, update the doc and the data file in the same commit.
- **Worked examples are tests.** Each `EX-n` in core-rules.md section 11 must have a test named `TestEX<n>...`. If a test and its example disagree, the doc is right unless the humans say otherwise.

## 2. Goals

1. A **deterministic Go engine** that plays the Starter Battle exactly as the rules describe.
2. A **command-line tool** to play by order files (works for play-by-mail and for testing).
3. A **web tool** to view the map, enter orders, replay execution step by step, and tweak rules data to explore variants.

Wartable is a base for other games, so the engine must be data-driven: new units, scenarios and rule values should not require code changes wherever reasonable.

## 3. Rules as data

Use YAML (`gopkg.in/yaml.v3`). Suggested layout:

```
data/
  units/standard.yaml        # Infantry, Cavalry, Artillery stat blocks
  rules/core.yaml            # position bonuses, ambush penalty, half-strength penalty, etc.
  scenarios/starter-battle.yaml
```

Example `rules/core.yaml`:

```yaml
position_bonus: { front: 0, flank: 1, rear: 3 }
ambush_def_penalty: 1
half_strength_penalty: 1
support_per_ally: 1
direction_order: [N, NE, SE, S, SW, NW]
```

Example unit:

```yaml
- id: artillery
  name: Artillery
  cost: 15
  move: 4
  def: { ready: 4, mobilised: 2 }
  range: 5
  rng_dmg: 3
  attack: 3
  att_dmg: 0
  melee: false
  states: [mobilised, ready]
  deploy_state: mobilised
```

Keep unit special cases (artillery states) expressed as data where possible: e.g. which states allow moving/firing, and the Def per state.

## 4. Architecture

Go module: `github.com/MrMykalAnderson/wartable`. Standard library first; add dependencies only when they clearly help.

```
cmd/wartable/        CLI
cmd/wartable-web/    web server (later)
internal/hex/        coordinates, neighbours, distance, firing arc, pathfinding
internal/rules/      loading and validating YAML data
internal/game/       state, orders, initiative, execution, combat
internal/orders/     parsing order sheets
web/                 static front end (later)
```

Key design points:

- **Pure engine.** `game.ExecuteTurn(state, ordersNorth, ordersSouth) -> (newState, []Event)`. No I/O inside `game`.
- **Event log.** Every step (initiative result, each step of movement, facing change, contact, each combat check with its numbers, hit, knockback, destruction, skipped order) is emitted as a structured `Event`. The CLI prints them; the web tool replays them. Events must contain the numbers used, so humans can audit a result against the rules.
- **Determinism.** No randomness, no map iteration order leaking into results (Go maps are unordered: always sort). Same inputs must give byte-identical event logs.

## 5. Hex implementation notes

Coordinates in the rules are offset coordinates: column letters, row numbers, flat-topped hexes, with lettered columns B, D, F … shifted down half a hex. With **0-based** indices (A = column 0, row 1 = row 0) this is the **"odd-q"** layout in Red Blob Games' terms. Convert to cube/axial coordinates internally for maths:

```
q = col
r = row - (col - (col & 1)) / 2     // 0-based col and row
```

Cube direction vectors for flat-top hexes (q, r, s):

| Dir | Vector |
| --- | --- |
| N | (0, −1, +1) |
| NE | (+1, −1, 0) |
| SE | (+1, 0, −1) |
| S | (0, +1, −1) |
| SW | (−1, +1, 0) |
| NW | (−1, 0, +1) |

Check your conversion against the table in core-rules.md 2.1 (e.g. SE of A1 is B1; NW of B1 is A1; SW of F6 is E7).

- **Distance:** standard cube distance.
- **Firing arc:** for a unit facing direction `F`, a hex `H` is in the arc if `H − unit = a·dir(F−1) + b·dir(F+1)` with `a ≥ 0` and `b ≥ 0`. The two flank directions form a basis of the hex lattice, so `a` and `b` are integers. Include the boundary (a = 0 or b = 0).
- **Edge attacked:** the direction from the defender to the attacker; compare with the defender's facing: same = front, ±1 = flank, else rear.
- **Knockback hex:** loser + (loser − winner).
- **Pathfinding:** BFS distance field from the destination over passable hexes. Step greedily to a neighbour whose distance is one less, choosing the first in N, NE, SE, S, SW, NW order. Destination unreachable → see core-rules.md 7.2. Movement also stops on entering any hex adjacent to an enemy (core-rules.md 7.3).

## 6. Order sheet format

Plain text, one order per line, in execution order. `#` starts a comment. Fields are separated by `|`:

```
# North, turn 3
1st Horse | Move             | F9 | facing S
2nd Foot  | Move             | 1st Guns
1st Guns  | Fire             | 3rd Foot
3rd Foot  | Close and Attack | 4th Horse
1st Guns  | Ready            | facing SE
```

Parse errors (unknown unit, unknown order, two orders for one unit) are reported before execution and reject the sheet. Orders that can't be completed during execution follow the "best attempt" rules and produce events, not errors.

## 7. Milestones

Each milestone ends with passing tests, a short note in `docs/CHANGELOG.md`, and a push.

| # | Milestone | Done when |
| --- | --- | --- |
| M1 | `hex` package | Neighbours, offset↔cube conversion, distance, arc, knockback hex, pathfinding with tie-break. Table-driven tests incl. the examples in 5. |
| M2 | Rules data | YAML files for units, core rules and the Starter Battle load and validate. A test checks they match the rules docs' numbers. |
| M3 | Combat | Melee (support, position, ambush, hit, damage, knockback, destruction) and ranged. Tests EX-2 to EX-6. |
| M4 | Orders and turns | Order parsing, initiative with tie-break token, alternating execution, all order types, artillery states. Test EX-1, plus multi-order scenarios. |
| M5 | CLI | `wartable new`, `wartable turn <state> <north.txt> <south.txt>`, prints an ASCII map and the event log; game state saved as JSON. Win conditions from the scenario. |
| M6 | Web viewer | Local web server: draws the hex map with units and facing, loads a saved game, steps through a turn's events one at a time. |
| M7 | Web play and sandbox | Enter orders for both players in the browser (hot-seat), and edit rules data in the browser to replay a turn under different values. |

Don't start M6 until the engine (M1–M5) is solid; the point of the web tools is to explore rules, which only works if the engine is trustworthy.

## 8. Definition of done (every change)

- `go vet ./...` and `go test ./...` pass.
- New rules behaviour has a test, ideally tied to a worked example.
- Any `RULES-QUESTION` raised is listed in core-rules.md section 12 and in the commit/PR summary.
