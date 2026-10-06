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
