# Changelog

## M1: `hex` package

Implemented `internal/hex`: offset/cube coordinate conversion, neighbours,
distance, firing arcs, edge-attacked (front/flank/rear), knockback, and
pathfinding (BFS distance field with clockwise tie-break, falling back to
the closest reachable hex per core-rules.md 7.2 when the destination is
unreachable). Table-driven tests cover the worked examples in core-rules.md
section 11 (EX-3, EX-4) and the direction table in section 2.1.

`go vet ./...` and `go test ./...` pass.

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
