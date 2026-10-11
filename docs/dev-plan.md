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
| [drafts/](drafts/) | Rules still being designed. **Not a source of truth:** don't build from them. |

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
| M7 | Web play and sandbox | Enter orders for both players in the browser (hot-seat) **by clicking on the map**, and edit rules data in the browser to replay a turn under different values. See 7.1. |
| M8 | Online two-player play | Games stored on the server; each player gets a private link and sees only their own orders until both have submitted. See 7.2. |

For M6 onward, draw units as described in core-rules.md 3.3 and shown in `docs/images/`: a type symbol in a box, the front edge red, the flank edges green, half-strength units visibly marked. Draw them as SVG (don't embed the PNGs) so tokens can rotate to any facing and scale cleanly.

### 7.1 M7: click-to-order

Typing order text is fine for testing but too slow for playtesting. In M7, orders are built by clicking:

- Click one of your units to select it; the page highlights the hexes it can reach and the enemies it can target.
- Click a hex for a Move, or an enemy for Close and Attack, Fire or Close and Fire (offer the valid order types for that target). Choose a facing with a small six-way picker on the destination hex, or let it default.
- Orders appear in a list beside the map in execution order; they can be reordered (drag or up/down buttons) and deleted.
- The list produces the same order-sheet text as section 6, so the engine, CLI and saved games don't change. Show the text too, so it can be copied.
- Previews are guidance only: they must use the engine's own rules (e.g. via an API call), never a second copy of the rules in JavaScript.

Keep the visual style plain for now. Usability matters now; visual polish (art, typography, animation) waits until the rules settle.

### 7.2 M8: online two-player play

Secret orders are the heart of the game, so hot-seat is for testing only. M8 makes real remote games possible, kept as simple as possible:

- **No accounts yet.** Creating a game returns two private player links (long random tokens in the URL), one for North and one for South, plus an optional read-only spectator link.
- Each player sees the map and their own order list. They can't see the other player's orders, only whether they have submitted.
- When both have submitted, the server runs the turn, saves it, and both players see the event replay.
- A player can edit and resubmit until the other player has also submitted; after that the turn is locked.
- Games are stored on disk (one JSON file per game, or SQLite), and every turn's order sheets are kept so any game can be replayed.
- Simple polling (every few seconds) is enough to notice the other player submitting; no websockets needed yet.
- Include a short `docs/deploy.md` on running it on a small hosted server, but don't pick a host without asking.

- A **Concede** button: ends the game and records the other player as the winner.

Accounts, lobbies and matchmaking come later, if ever.

### 7.3 Playtest round 1 changes (do before finishing M7)

From `Playthrough notes 1.md`. Rules first, then interface.

**Rules (see core-rules.md, already updated):**

- **Range bands.** Range is now minimum–maximum. Add `min_range` to every unit in the data (default 1); artillery is 3–5. Half strength lowers only the maximum. Apply the minimum to every ranged attack (Fire, Close and Fire, Barrage).
- **Artillery Barrage (passive order).** At the start of every Execution Phase, before written orders, every Ready artillery unit fires at every enemy in its band and arc. Resolve all passive fire for both sides simultaneously: compute every attack from start-of-phase positions, then apply all hits. Multiple hits on one unit stack. The artillery still gets its normal order. Emit a clear event per shot.
- Add worked example **EX-7** as a test, and update the golden replay tests (their results will change). Note the changes in the changelog.

**Interface:**

1. **Show pending orders on the map** while planning: a ghost token for Deploy, a path line ending in a ghost token with its facing for Move, an arrow to the target for attacks and Fire.
2. **Click a unit on the map to select it** for ordering (currently only possible from the orders panel).
3. **Unit details**: hovering or selecting a unit shows a card with its current stats (half strength applied), facing, artillery state, and a short description of the unit type.
4. **Step-by-step turn playback**, prominent and obvious: one plain-English line per event, a Next button (and Back if easy), with the map updating at each step. Barrage shots are steps too.
5. **Range visualiser** for the selected unit: green hexes it can move to this turn, red hexes it can fire into (band + arc). For Ready artillery, also show its barrage zone. All of this must come from the engine (e.g. `/api/options`), never recomputed in JavaScript.

Don't start M6 until the engine (M1–M5) is solid; the point of the web tools is to explore rules, which only works if the engine is trustworthy.

### 7.5 Playtest round 2 changes (do next)

**Combat rewrite (core-rules.md 8.3 and 9, already updated).** Combat is now one comparison, and the margin is the damage:

- Margin = attacker's total (Attack + support + position bonus) − defender's total (Def + support, −1 if ambushed). Ranged: no position bonus.
- Margin ≤ 0: melee attacker repelled (knocked back, no damage); ranged miss. 1–2: one hit (melee defender then knocked back if it survives). 3+: two hits (destroyed).
- The defender no longer deals damage when it wins.
- Remove `att_dmg` and `rng_dmg` from the data, the code and the interface. Update EX-2 to EX-7 tests and regenerate the golden replays (results will change); note it in the changelog.
- Add a test that computes **Appendix A (combat tables)** from the engine for the standard units and checks it matches the doc exactly. If they ever disagree, the test should fail.
- Event log lines should state the totals and the margin, e.g. "1st Horse (3 +3 rear) vs 2nd Foot (2): margin 4, destroyed".

**Interface.**

- When a Close and Attack or Close and Fire order is being made, the movement highlight shows the **half-move** reach, clearly different from a full Move (e.g. a different shade and a label "Close: 2 hexes").
- Before an attack or Fire order is confirmed, show the **predicted result** if the target stays put (margin and Repelled/Hit/Destroyed, from the engine). Label it as a prediction, since the target may move first.

### 7.6 Playtest round 3 fix: planning shows reach, not paths (do next)

The planner currently works out movement against the board as it is now, so a unit ordered to move behind a friend that is also moving is shown as blocked. That can't be known while planning: by the time the order runs, other units (friendly and enemy) will have moved. Paths are only worked out at execution (core-rules.md 6.2, 6.6), so planning must not pretend otherwise.

- **Reach highlight = pure distance.** Show every hex within the unit's Move (or half Move for Close orders) of its current position, ignoring all units. Only the map edge limits it. Still from the engine (`/api/options`), just without obstacles.
- **Any hex is a valid Move target.** Clicking any hex on the map, inside or outside the highlight, creates a Move order toward it. Likewise any enemy is a valid Close/Fire target regardless of distance (the prediction can say "out of reach this turn").
- **Pending-order display:** draw a line from the unit toward its objective, and mark where it would stop at full Move on an empty map (straight-line estimate, e.g. a ghost token labelled "about here"). Label it as an estimate.
- Never reject or warn on an order just because of distance or units in the way. Only reject orders that break the order-sheet rules (unknown unit, two orders for one unit, wrong order type for the unit).
- Add a test: two friendly units in a column, the rear one ordered to a hex beyond the front one; both orders accepted, and execution moves the front unit first so the rear one passes through the vacated hex (when the front unit's order runs first).

### 7.7 Playtest round 4 changes (do next)

From Mykal's full game (`Testaftermovementplanningupdate.json`). Rules already updated in core-rules.md.

**Bug.** `resolveAmbush` lets any adjacent enemy attack, including artillery, which can't make melee attacks (core-rules.md 3.4). This destroyed an infantry unit in the playtest. Only units with `melee: true` take part in an ambush.

**Rules changes.**

- **Dug-in artillery** (3.4, 8.3): Ready artillery can't be knocked back; if it loses a melee by any margin it is destroyed. Ranged hits on it are normal. Mobilised artillery is knocked back as usual.
- **Overrun** (7.4): after any ambush, if the moving unit is still on the map and still next to enemies that can't make melee attacks, it attacks each in turn (clockwise from N, normal melee, no ambush penalty), stopping if repelled.
- **Safe route** (7.2): pathfinding prefers the shortest path whose hexes (excluding the destination) are never adjacent to an enemy; adjacency to a Close order's named target doesn't count. Fall back to the plain shortest path only if no safe route exists. Judge the route over the whole way to the destination. Keep the clockwise tie-break and the unreachable-destination fallback.
- Add **EX-8** (safe route) and **EX-9** (overrun) as tests. Appendix A has changed (Ready artillery rows); update `TestAppendixACombatTables`. Regenerate the golden replays.
- **No man's land check.** Add an engine invariant: after every order (and after the barrage), no two enemy units are adjacent. In tests, fail if it's ever broken; in normal play, emit a warning event naming the units, so a broken rule is spotted immediately. Run it over the golden replays.

**Interface.**

- **Order lines in the side's colour**, with the line style showing the order type: Move solid, Close and Attack dashed, Fire dotted, Close and Fire dash-dot (or similar), Ready/Mobilise a small marker on the unit. Add a small legend.
- The pending-move estimate ("about here") follows the route over **terrain only, ignoring every unit** (see 7.9), still labelled as an estimate.

**Saves.**

- Save files must keep the **full history**: every turn's two order sheets and the resulting event log, not just the final position. The web viewer should be able to step through any past turn. (M8 needs this anyway.)
- Keep playtest saves in a `playtests/` folder; move `Testaftermovementplanningupdate.json` there.

### 7.8 Terrain and the Two Towns scenario (do after 7.7)

New rules: core-rules.md 2.3 (terrain, road march, forest, fording, river adjacency), 7.1, 8.3 (knockback across a river), EX-10 and EX-11. New scenario: `docs/two-towns.md`. Map data: `data/maps/two-towns.yaml` (you may change its format to suit the engine, but keep every hex, edge and road the same; `docs/maps/twotowns.py` generated it).

**Engine.**

- Load maps from `data/maps/`. Validate: river and bridge edges join adjacent hexes, every bridge is a river edge with road on both sides, road chains are contiguous.
- Terrain effect numbers go in data (e.g. `data/rules/terrain.yaml`), not code.
- Movement: river edges block movement except at bridges; **fording** is only a Move from a hex with a river edge to the hex directly across it, as the unit's whole order; the pathfinder never fords. **Road march** (start and destination both road hexes): follow road hexes only, +2 Move, no safe route, can't pass units. **Forest**: half Move once the unit starts in or enters forest; stop in the first forest hex if already over that allowance; stacks with Close halving.
- Adjacency for contact, support, ambush, overrun and the no man's land check ignores pairs separated by a river edge (bridges count as adjacent). Knockback across a river edge is blocked.
- Combat: +1 Def for defenders in forest, towns and hamlets (melee and ranged); cavalry −1 Attack if it or its target is in forest; −1 for an attacker attacking across a bridge edge; artillery can't go Ready in forest.
- Scenario: deployment zones (hexes within 3 of own town), objectives with ownership, capture win, turn-16 scoring (core Cost + 20 per town + 10 per hamlet), default facings NE/SW.
- Tests: EX-10, EX-11, plus unit tests for each terrain effect. The Starter Battle has no terrain, so its golden replays should not change.

**Interface.**

- Choose the scenario when starting a new game (Starter Battle or Two Towns).
- Draw terrain: forest, towns and hamlets as hex fills, roads as lines through hex centres, the river along hex edges, bridges marked. Show objective ownership.
- Reach preview: road march reach along the road (+2), forest-limited reach, and a clear ford option when a unit stands on a river bank.
- The map is 26 × 18: add **zoom and pan**. The map-first layout (7.4) will matter more now; ask Mykal before starting it.

### 7.9 Playtest round 5 fixes (do next)

From Mykal's first Two Towns game. All interface; no rules change.

**Planning must ignore units (regression).**

- The move preview (`fetchMovePreview` → `/api/predict`) runs the engine on the current board, so a unit ordered along a road behind a friend is shown as blocked by that friend. Previews and estimates must use **terrain only** (rivers, bridges, fording, road march, forest) and ignore every unit, friend or enemy, exactly like the reach highlight. (7.7 told you to use the current board; that was wrong.) Add a test: with a friendly unit standing on the road, the preview of a road march past it is unaffected.
- **Any hex can be an order target, including occupied ones.** Today, clicking a friendly unit always selects it, so its hex can never be a Move objective. When a unit is selected and you click another unit's hex, offer the choices: "Move here" (any unit's hex) plus "Select this unit" (own side) or the attack types (enemy). With nothing selected, clicking your own unit selects it as now.

**Facing is optional.**

- Clicking a target adds the order straight away with **automatic facing** (the direction of the last step; for Deploy, the scenario's default). No facing step is required.
- Facing can be set afterwards, if wanted, from a small six-way control on the order's list entry and on its ghost. Show the facing the order will end with on the ghost either way.

**Display.**

- **Deployment-zone and reach overlays must be translucent** (e.g. a light tint plus outline), so terrain stays visible underneath.
- **Turn replay loses the terrain:** stepping through events (e.g. deployment) draws an empty hex grid with only units. Every view of the map must always draw the terrain layer.
- **Straight roads.** Draw roads through the **midpoints of the hex edges** they cross (starting and ending at the end hexes' centres), then smooth with 2–3 passes of Chaikin corner-cutting. Alternating two-direction runs then render perfectly straight. Reference implementation: `road_curve` in `docs/maps/twotowns.py`. The rules (which hexes are road) don't change.

### 7.10 Playtest round 6 changes (do next)

From `playtests/newMapTest.json` (analysis: all 7 kills were ranged; cavalry Close and Fire with shooter support killed full-strength units in one shot). Rules already updated in core-rules.md, two-towns.md and scenario-design.md.

**Rules.**

- **Forest costs 2 Move to enter** (replaces "half Move"). Units stop when they can't pay for the next step; every unit can always move at least one hex per order; a Close order's budget is half Move. Movement costs go in the terrain data.
- **Cheapest route** (7.2): the safe route is now the cheapest path by movement cost (Dijkstra), tie-broken by fewer hexes, then the clockwise rule. Same for the plain fallback route and for planning previews. Add **EX-12** as a test.
- **No shooter support** (8.1, 9): Fire, Close and Fire and Barrage use the shooter's Attack alone; the target still gets support. Update `TestAppendixACombatTables` for the ranged table, and regenerate the golden replays (results will change).
- **Starting army and bank** (section 10): scenarios may set a `starting_limit` (Two Towns: 120 of 240). The starting army must fit the starting limit; the rest is the bank, shown to both players. From turn 2, a Deploy order may name a unit type (`Deploy | Cavalry | X3`) to buy and deploy a new unit, paid from the bank, named with the next free number. A failed Deploy buys nothing. Starter Battle has no starting limit, so nothing changes there.
- **Two Towns win conditions**: capture must be held through the following turn; annihilation counts reserves and bank; unspent bank doesn't score.

**Interface.**

- **Unit labels**: each token shows a short code (type letter + number: I1, C2, A1), readable at normal zoom. Increase the default hex size.
- **Facing markers off-centre**: the "≈ here" estimate ghost is placed partway along a straight line, between hexes, so its facing dots aren't centred. Always snap estimate ghosts to a hex centre.
- **History turn numbers are one ahead** (the first played turn is saved as turn 2, with an empty turn 1). Fix the numbering so it matches the turn the players saw.
- **River visibility**: draw river edges above forest fills, with enough contrast to see in forest. Label a Move order that will ford as "Ford (whole turn)" in the order list and on its line, so a ford is never a surprise.
- **Bank**: show each side's bank; in the deploy picker, offer "Buy" with each affordable unit type.
- **Capture status**: show when a town is captured and must be held for one more turn.
- Reach preview uses forest costs.

### 7.4 Parked (don't start without being asked)

- **Map-first layout.** The map should be the main focus and fill the screen. All panels (orders, unit details, event playback, sandbox) become either docked strips around the map edges or floating, movable windows. A large layout change; do it as one piece of work when asked, not piecemeal.

## 8. Definition of done (every change)

- `go vet ./...` and `go test ./...` pass.
- New rules behaviour has a test, ideally tied to a worked example.
- Any `RULES-QUESTION` raised is listed in core-rules.md section 12 and in the commit/PR summary.
