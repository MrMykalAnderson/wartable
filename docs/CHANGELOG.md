# Changelog

## M1: `hex` package

Implemented `internal/hex`: offset/cube coordinate conversion, neighbours,
distance, firing arcs, edge-attacked (front/flank/rear), knockback, and
pathfinding (BFS distance field with clockwise tie-break, falling back to
the closest reachable hex per core-rules.md 7.2 when the destination is
unreachable). Table-driven tests cover the worked examples in core-rules.md
section 11 (EX-3, EX-4) and the direction table in section 2.1.

`go vet ./...` and `go test ./...` pass.

## Web viewer: show the path taken and the order's actual target

Playtesting found two more things: a Close order's "moved" event said
things like "closed to D6", which reads like a Move order to a hex even
though Close and Attack/Close and Fire always target a *unit*; and
there was no way to see the route a unit took to get where it ended up,
only its final position.

Added a `Path` field to `game.Event` (the hexes a "moved" event's unit
actually entered, in order; already computed by `moveTowards` and the
Close and Fire step loop, just not kept) and exposed it on `EventView`.
Close and Attack/Close and Fire's move-leg `Detail` now names the
target unit ("closing on South 1st Infantry (now at F5)" instead of
"closed to F5"); plain Move keeps "moved to X", since a Move order
really does target a hex. The web viewer's step detail panel now shows
"Path: F3 → F4 → F5", and the map draws the actual route as a solid
line with waypoint dots while that step is selected (reusing the
existing hex-center math, under a new `#path-layer`, separate from the
pending-order ghost layer).

Also regenerated `state.json` via `wartable new`: the previous save's
units had their stats embedded at creation time, so it was still
playing with pre-range-band artillery (0-5 instead of 3-5) even though
`data/units/standard.yaml` was already correct — a fresh game picks up
current rules data. The stale save was kept as a timestamped backup.

`go vet ./...` and `go test ./...` pass (regenerated the `annihilation`
golden replay's expected output for the new "closing on X" wording;
`turnlimit` was unaffected). Verified end-to-end in a real browser:
deployed both sides, ran a Close and Attack across several hexes, and
confirmed the event text, the "Path: ..." line, and the drawn path line
on the map all matched the actual route taken.

## Web viewer: step-by-step map replay, turn history, calculation breakdowns

Four interface fixes requested after playing through a real game in the
viewer: the map only ever showed the final post-turn state while
stepping through events; North's and South's orders were stacked
instead of side by side; there was no way to look back at a previous
turn; and combat events only showed the final hit-check totals, not the
bonuses that made them up.

**Engine:** added a `Board` field to `game.Event`, set to the board
exactly as it stood right after that event (not the turn's final
board), threaded through every event-construction site in
`move_orders.go`, `deploy.go`, `ambush.go`, and `barrage.go` — including
each ambush round and each barrage shot getting its own incremental
snapshot, even though a barrage's shots are conceptually simultaneous.

**Web views:** added `BoardView` (an event's board snapshot),
`CoreRulesView` (the position-bonus/ambush-penalty/support numbers
needed to label a breakdown), and `MeleeView`/`RangedView` (every
component of a combat result — support, position bonus, ambush penalty,
totals, damage, knockback — broken out instead of just the final
numbers). `GameView` now carries `coreRules`; `EventView` now carries
`board` and the new `melee`/`ranged` views.

**Frontend:** the map now replays the board at whichever step is
selected, using each event's own snapshot; North's and South's order
lists/text sit in a two-column layout; a new turn-history nav
("« Turn"/"Turn »") lets the viewer step back through previously-run
turns independently of the current event-step controls, with a "Live"
state for building the next turn's orders; and each melee/ranged event
now renders a plain-language breakdown (e.g. "Attack 3 + Support 2 +
Position bonus 1 = 6 vs Def 1 = 1 → attacker wins") computed by simple
arithmetic on the server's own numbers — never re-deciding a rule
client-side. Selecting a unit to build an order always jumps back to
the live state first, so the highlighted hexes/enemies match what's
on screen.

`go vet ./...` and `go test ./...` pass. Verified end-to-end in a real
browser via claude-in-chrome against the user's own saved game: ran a
turn, stepped through its events watching the map update each time
(including a unit being destroyed), confirmed the melee breakdown's
numbers, and navigated back to the turn's start and forward to "Live".

## M5: CLI

Added `cmd/wartable` (`new`, `turn`, `show`) and `internal/game`'s
`CheckAnnihilation`/`ScoreAtTurnLimit` for the Starter Battle's win
conditions. `new` seeds both sides' reserves with the suggested army;
`turn` loads a saved game plus two order-sheet files, runs
`game.ExecuteTurn`, prints a plain ASCII map and the event log (with the
melee/ranged numbers), checks for a win, and saves the updated state as
JSON; `show` prints an existing save's map and reserves. Deliberately
minimal: no flags, no interactive prompts, and only the Starter Battle.
Verified end-to-end with a real multi-turn game (deploy, move, ready,
fire, and an ambush with knockback all resolved correctly, including the
tie-break token correctly carrying over between turns).

`go vet ./...` and `go test ./...` pass.

## Playtest round 1 changes (dev-plan.md section 7.3)

**Rules:** Range is now a band, `min_range`–`range` (artillery 3–5;
other ranged units default to 1–their old range); half strength only
lowers the maximum. Added artillery's passive **Barrage**: at the start
of every Execution Phase, before written orders, every Ready artillery
unit fires at every enemy in its band and arc — all shots for both
sides are computed from the same start-of-phase board, then every hit
is applied together, so a unit hit by two guns in one barrage is
destroyed, and a unit the barrage itself destroys still counted as
support for other shots in the same barrage. The gun still gets its
own written order afterward. Added `internal/game/barrage.go`
(`ResolveBarragePhase`), wired into `ExecuteTurn` before the
alternating written-order sequence, with its own `[barrage]`-prefixed
events. Added EX-7 as a test and regenerated the `turnlimit` golden
replay test — the extra early damage changes the whole rest of that
game, including who wins (north now, not south); see its updated
`README.md`. The `annihilation` golden game was unaffected (no Ready
artillery ever had an enemy in range during it) and needed no changes.

**Interface:** click a unit on the map (not just the orders panel) to
select it; a unit card (full stats, facing, state, flavour text) shows
on selection; a red range visualiser (band + arc) joins the green move
highlight, doubling as a Ready gun's barrage-zone preview; pending
orders now show as ghost tokens/paths/arrows on the map while planning;
event-log stepping auto-scrolls the current event into view. All of
this reuses the existing `/api/options` and the new `stats` field on
`/api/game`'s units — never recomputed in JavaScript.

`go vet ./...` and `go test ./...` pass. Verified in a real browser via
claude-in-chrome: map click-to-select, the range/barrage visualiser,
the unit card, order ghosts, and barrage events in the turn log all
confirmed working, no console errors.

## M7 (part 1): click-to-order

Replaced the web viewer's free-text order textareas with click-to-order
(docs/dev-plan.md section 7.1): click a unit, then click a highlighted
hex, an outlined enemy, or a state button on the map to give it an
order, choosing a facing (or Auto) where one applies; orders collect
into a reorderable, deletable list per side, which also renders as the
same order-sheet text `wartable turn` reads (shown read-only, for
copying). Added `GET /api/options` (new `cmd/wartable-web/options.go`),
which the engine computes — reachable Move hexes, valid Deploy hexes,
and which enemies are valid Close-and-Attack/Fire/Close-and-Fire
targets — so the frontend never re-derives rules in JavaScript.
Exported `UnitInstance.CanMove/CanFire/CanTurn` and `game.PassableFor`
from `internal/game` for the handler to reuse.

Found and fixed two real UI bugs while testing in a live browser (not
just curl): the "targetable" highlight ring was invisible because the
facing-indicator lines painted over the token's own border (fixed with
a separate ring element), and the facing-picker/state-button/type-
chooser panels stayed visible when they should have been hidden,
because a `.chooser { display: flex }` rule unconditionally overrode
the `hidden` attribute (fixed with an explicit `.chooser[hidden]`
override). Verified end-to-end via claude-in-chrome: deploying a full
army, moving, Close and Attack with a type choice offered correctly,
and Ready with an explicit facing all produced the expected order text
and executed correctly, with no console errors.

`go vet ./...` and `go test ./...` pass.

## M6: Web viewer

Added `cmd/wartable-web` (a local HTTP server: `GET /api/game`,
`POST /api/turn`, serving `web/` as static files) and `web/` (a
vanilla-JS single-page viewer): draws the hex grid and unit tokens as
SVG, following the core-rules.md 3.3 token convention (front edge red,
flank edges green, rear unmarked, half-strength tokens faded), loads a
saved game, runs a turn from pasted order sheets, and steps through the
resulting event log one event at a time, highlighting the unit involved.
Extracted `internal/save` (the `SaveFile` format and rules-data loading)
out of `cmd/wartable` so both binaries share it, and moved event
formatting onto `game.Event.Summary()` (and `MeleeResult`/`RangedResult`)
so the CLI and the web server render identical text. Verified in a real
browser: hex stagger, facing-relative edge colours, deploy/move/melee
events, and event stepping all confirmed working, with no console
errors.

`go vet ./...` and `go test ./...` pass.

## Golden replay tests (pre-M6 verification)

Played two complete Starter Battle games by hand through the CLI and
recorded them as golden replay tests (`cmd/wartable/testdata/`,
`golden_test.go`): one ending in annihilation (turn 12), one reaching the
turn-12 limit and scored on points. Together they exercise a failed
Deploy, a rear attack, a flank attack, an ambush by more than one enemy
in sequence, and a knockback destroyed by a blocked (occupied) retreat
hex. Refactored `runNew`/`runTurn`/`runShow` to take an `io.Writer` so
tests can replay a game by direct function calls instead of shelling out.
See the `testdata/*/README.md` files for a turn-by-turn account.

`go vet ./...` and `go test ./...` pass.

## M4: Orders and turns

Added `internal/orders` (order sheet parsing: unit/order/target/facing
fields, "#" comments, duplicate-order and bad-facing detection, with all
sheet errors reported together) and extended `internal/game` with
`GameState`, `Initiative`/`TieBreak`, `BuildExecutionOrder`, and
`ExecuteOrder`/`ExecuteTurn` covering every order type (Deploy, Move,
Close and Attack, Fire, Close and Fire, Ready/Mobilise), movement with
the contact/ambush rules, and artillery state gating (can-move/can-fire,
and the flat Close-and-Fire prohibition). Tests cover EX-1 (initiative and
execution order) plus multi-order scenarios: ambush during a Move, Close
and Attack both adjacent and after closing, ambush-by-a-different-enemy,
Fire and Close and Fire, and a full `ExecuteTurn` mixing Deploy/Move/Fire
across both sides, including a destroyed unit's later order being skipped.

`go vet ./...` and `go test ./...` pass.

## M3: Combat

Added `internal/game`: `UnitInstance`/`Board` (support, adjacency, retreat
checks) and `ResolveMelee`/`ResolveRanged`, covering support, position
bonus, ambush, hit/damage checks, knockback and destruction. Tests cover
worked examples EX-2 through EX-6, plus the half-strength-destroyed and
blocked-retreat-destroyed branches EX-4's note calls out.

`go vet ./...` and `go test ./...` pass.

## M2: Rules data

Added `data/units/standard.yaml`, `data/rules/core.yaml` and
`data/scenarios/starter-battle.yaml`, plus `internal/rules` to load and
validate them (duplicate/unknown unit IDs, deployment zones outside the
map, artillery states without a per-state Def, etc.). Tests check the
loaded numbers against core-rules.md and starter-battle.md.

`go vet ./...` and `go test ./...` pass.
