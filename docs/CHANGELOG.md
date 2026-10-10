# Changelog

## M1: `hex` package

Implemented `internal/hex`: offset/cube coordinate conversion, neighbours,
distance, firing arcs, edge-attacked (front/flank/rear), knockback, and
pathfinding (BFS distance field with clockwise tie-break, falling back to
the closest reachable hex per core-rules.md 7.2 when the destination is
unreachable). Table-driven tests cover the worked examples in core-rules.md
section 11 (EX-3, EX-4) and the direction table in section 2.1.

`go vet ./...` and `go test ./...` pass.

## Playtest round 5 fixes (dev-plan.md section 7.9)

From Mykal's first Two Towns game. All interface; no rules change.

- **Planning must ignore units (regression fix).** A Move prediction's
  `/api/predict` path was running the real engine against the live
  board (7.7's design), so a road march behind a friendly unit on the
  road showed as blocked by it. `game.PreviewMovePath` (new,
  `internal/game/preview.go`) computes a terrain-only estimate instead
  — fording, road march and forest still apply, but no unit, friendly
  or enemy, ever blocks or stops it, the same as the reach guide.
  `handlePredict` now only calls the real engine (`game.ExecuteOrder`)
  for an attack/Fire order's combat numbers, which do need the actual
  board. A Close/Fire order's own predicted combat is unaffected — only
  a Move's path preview changes.
- **Any hex can be an order target, including occupied ones.** Clicking
  a unit's hex while another unit is selected used to always reselect
  it, so a friendly unit's hex could never be a Move objective. It now
  offers choices — "Move here" (any unit's hex), "Select this unit"
  (own side, if it doesn't already have an order), or the applicable
  attack types (enemy) — running the one choice straight away if
  there's only one (`handleTargetClick`/`offerChoices` in web/app.js).
  With nothing selected, clicking your own unit still just selects it.
- **Facing is optional.** Clicking a target now adds a Move/Deploy/
  Ready/Mobilise order immediately, with automatic facing (no pick-a-
  facing step); the predicted-result confirmation for attack/Fire
  orders is unchanged. Facing can be adjusted afterward from a small
  six-way control, shown both on the order's list entry and on its
  ghost on the map (`effectiveFacing`/`setOrderFacing`/
  `buildFacingControl`/`appendFacingWheel`) — the control always shows
  the facing the order will actually end with, worked out the same way
  the engine would (a Move's own fetched preview path's last step, a
  Deploy's scenario default, or an unchanged Ready/Mobilise's current
  facing), not just what the player explicitly set. `GameView` gained
  `DefaultFacing` so the frontend can show this for Deploy without
  asking the server.
- **Display.** Deployment-zone and reach overlays are translucent
  (`fill-opacity`) so terrain stays visible underneath. Every view of
  the map — including mid-replay, stepping through a past turn's
  events — now always draws the terrain layer: `renderMap` draws
  terrain/objective ownership from the live game state, never from
  whatever historical board snapshot it's otherwise drawing (a
  `BoardView` carries no terrain of its own). Roads are drawn through
  the midpoints of the hex edges they cross, then smoothed with Chaikin
  corner-cutting (`roadCurve`, porting `docs/maps/twotowns.py`'s
  `road_curve` exactly), so an alternating two-direction run reads as a
  straight line instead of zig-zagging through hex centres.

`go vet ./...` and `go test ./...` pass.

## Terrain and the Two Towns scenario (dev-plan.md section 7.8)

**Engine.**

- **Terrain data.** `internal/rules/map.go` loads and validates
  `data/maps/*.yaml` (`LoadTerrainMap`): forest, roads, towns/hamlets,
  river and bridge edges — river and bridge edges must join adjacent
  hexes, every bridge must be a river edge with road on both sides, road
  chains must be contiguous, and a road hex is never forest. Every
  method on `*TerrainMap` is nil-safe, so a scenario with no terrain
  (the Starter Battle) behaves exactly as before. Terrain's tunable
  numbers live in `data/rules/terrain.yaml` (`TerrainEffects`), loaded
  separately and attached to `CoreRules.Terrain` by `save.LoadRulesData`.
- **`hex.EdgeBlockedFunc`** is a new, separate parameter on
  `FloodFill`/`NextStep`/`ShortestPath` (nil means nothing is ever
  blocked) for river-aware pathfinding, without changing the existing
  `PassableFunc` or any caller that doesn't care about edges.
- **Movement** (2.3): a river edge with no bridge blocks movement except
  fording — a plain Move to the hex directly across, as the unit's whole
  order, never found by the pathfinder on its own (`fordPath`). A
  **road march** (start and destination both road hexes) follows road
  hexes only, +2 Move, ignores the safe route, and can't pass through a
  unit — the route is found ignoring occupancy (`roadMarchPath`,
  `roadPassable`), since routing a sparse road network *around* a
  blocker the way the safe route does could send it miles out of its
  way by some unrelated road; `moveTowards`'s own stepping loop is what
  actually stops it one hex short (EX-10's note). **Forest** halves the
  whole order's Move, from the start if the unit begins there or from
  the first forest hex entered otherwise, stacking with Close's own
  halving (`moveTowards` in `internal/game/state.go`).
- **Adjacency** for contact, support, ambush, overrun and the no man's
  land check ignores pairs separated by a river edge (`Board.adjacentHexes`);
  bridges count as adjacent. Knockback across a river edge is blocked
  (`Board.CanRetreatTo`, now also taking the knocked-back unit's own
  hex to check the edge).
- **Combat** (`internal/game/melee.go`): +1 Def for a defender in
  forest, a town or a hamlet (the three don't stack); cavalry −1 Attack
  if it or the defender is in forest; −1 for an attack across a bridge
  edge; artillery can't go Ready in forest (`executeReadyMobilise`).
- **Scenario and objectives.** `DeploymentZone` now also supports a
  radius-from-town zone (`NearTown`/`Radius`, alongside the Starter
  Battle's row range) via `Contains`/`Hexes`. Towns and hamlets are
  objectives, owned by whichever side alone currently has a unit in
  them (`GameState.Objectives`, `InitialObjectives`/`UpdateObjectives`
  in `internal/game/objectives.go`, updated once per turn). `CheckCapture`
  adds the capture win condition, and `ScoreAtTurnLimit` now also adds
  Terrain's town/hamlet points for owned objectives — both no-ops for a
  scenario with no terrain.
- `save.LoadRulesData` now takes a scenario ID (resolving
  `data/scenarios/<id>.yaml`) instead of being hard-coded to the Starter
  Battle; every caller passes the save's own `ScenarioID`, or (for
  `wartable new`/the web "New game" form) an explicit choice. Each
  scenario has its own suggested army (`suggestedArmies`); Two Towns'
  is 4 Infantry, 2 Cavalry, 2 Artillery (Cost 110, per two-towns.md).
- Tests: EX-10 (road march, including the blocked-unit note) and EX-11
  (fording, including the never-automatic and blocked-far-bank notes)
  against the real Two Towns map and scenario data, plus unit tests for
  every individual terrain combat/movement effect. The Starter Battle's
  golden replay tests are unchanged.

**Interface.**

- A scenario picker (Starter Battle/Two Towns) on the "New game" form,
  both CLI (`wartable new <state.json> [scenario-id]`) and web.
- The server sends its own terrain data (`GameView.Terrain`, nil for a
  scenario with none) and objective ownership (`GameView.Objectives`)
  for the frontend to draw directly — forest/town/hamlet hex fills
  (tinted by owning side), roads as lines through hex centres, river
  and bridge edges along hex boundaries — never recomputed from rules.
- `GET /api/options` adds `roadMarchHexes` (a road unit's Move+2 reach
  along the road network) and `fordHexes` (hexes directly across an
  unbridged river edge), drawn as outlines over whatever fill already
  applies; `moveHexes`/`closeMoveHexes` account for a unit starting in
  forest, since that's certain regardless of what else happens this
  turn (unlike occupancy, which the reach guide still ignores).
- Zoom (wheel, or +/−/Reset view buttons, anchored under the cursor) and
  click-drag pan on the map, via the SVG's `viewBox` rather than its
  own coordinate system, so none of the existing hex-click handling
  changed. Needed now the map can be as large as Two Towns' 26×18.
  Verified in a real browser: terrain rendering, a full deploy-order
  click flow (including the facing picker), zoom in/out/reset and
  click-drag pan, with no console errors.

`go vet ./...` and `go test ./...` pass.

## Playtest round 4 changes (dev-plan.md section 7.7)

From Mykal's full game (`playtests/Testaftermovementplanningupdate.json`).

**Bug fix.** `resolveAmbush` let any adjacent enemy attack in an ambush,
including artillery, which can't make melee attacks at all — this
destroyed an infantry unit in the playtest that should have survived.
Only units with `melee: true` take part now.

**Rules changes.**

- **Dug-in artillery** (core-rules.md 3.4, 8.3): Ready artillery can't
  be knocked back; any melee loss, at any margin, destroys it outright
  (`UnitInstance.DugIn`). Mobilised artillery still knocks back
  normally. Ranged hits work the same as ever either way.
- **Overrun** (7.4): once a moving unit survives any ambush, if it's
  still next to enemies that can't make melee attacks, it attacks each
  in turn as the attacker, no ambush penalty, stopping if repelled
  (`resolveOverrun` in `internal/game/ambush.go`). Not limited to
  melee-capable movers — the rule text only excludes artillery from
  Close and Attack orders and from attacking *in* an ambush, so a
  mobilised gun that contacts another gun overruns it too (this
  actually happened regenerating the `annihilation` golden game).
- **Safe route** (7.2): a moving unit now prefers the shortest path
  whose hexes (other than the destination, and other than any hex next
  to a Close order's own target) are never adjacent to an enemy,
  falling back to the plain shortest path only if no safe route reaches
  the destination (`safeShortestPath`/`safePassable` in
  `internal/game/state.go`). Judged over the whole route to the
  destination, even if the unit can't get there this turn.
- **No man's land invariant**: `checkNoMansLand` runs after every order
  and after the barrage, emitting a `[warning]` event naming any two
  enemy units still left adjacent (should never happen — every contact
  should end in a fight). The golden replay tests now also fail
  outright if one ever appears in a replayed game's output.
- Added `TestSafeRouteEX8`, `TestOverrunEX9Destroyed`/
  `TestOverrunEX9Repelled`, `TestAmbushSkipsNonMeleeEnemies`,
  `TestCheckNoMansLandDetectsViolation` and friends. Updated
  `TestAppendixACombatTables` for the changed Ready-artillery rows.
  Regenerated both golden replays: `annihilation`'s order files grew
  two more turns (the ambush fix means North's units survive longer,
  so the original 9 didn't reach annihilation any more); `turnlimit`
  only shifted one stop-hex (a safe-route detour) with the same 35-35
  outcome.

**Interface.**

- Pending-order lines are now coloured by the ordering side and styled
  by order type (Move solid, Close and Attack dashed, Fire dotted,
  Close and Fire dash-dot; Ready/Mobilise get a small marker on the
  unit instead, having no target), with a legend under the map.
- The pending-Move "≈ here" estimate (section 7.6) now follows the real
  safe route once `GET /api/predict` (extended to accept Move orders,
  returning the route it would actually take) responds, instead of
  staying a straight-line guess; falls back to the straight-line guess
  while waiting or if the request fails.

**Saves.**

- `save.File` gained `History []save.TurnRecord`: every played turn's
  submitted order sheets, the board before it ran, and its event log.
  `save.File.RecordTurn` is the single shared way both `cmd/wartable`'s
  `runTurn` and `cmd/wartable-web`'s `handleTurn` append to it. `GET
  /api/game` now returns it, so the web viewer hydrates turn history
  straight from the save on load — browsing to a past turn, including
  the order sheets that produced it, works right after opening a saved
  game, not only for turns played in that browser session.
- Playtest saves now live in `playtests/`; moved
  `Testaftermovementplanningupdate.json` there.

`go vet ./...` and `go test ./...` pass. Verified end-to-end in a real
browser: confirmed the order-line legend and per-type styling, watched
a pending Move's ghost estimate update from a straight-line guess to
the real safe-route stop point once the prediction arrived, then
played a turn, reloaded the page from scratch, and confirmed the
turn's order sheets and events were still there to browse.

## Playtest round 3 fix: planning shows reach, not paths (dev-plan.md section 7.6)

The planner was working out movement against the board as it is *now*,
so a unit ordered to move behind a friend that's also moving showed as
blocked — impossible to know while planning, since paths are only
worked out at execution (core-rules.md 6.2, 6.6), by which time other
units have moved.

`GET /api/options`'s `moveHexes`/`closeMoveHexes` are now pure
distance from the unit's current position (a new `hexesWithin` helper),
ignoring every other unit on the board — the map edge is the only
limit. `fireTargets` is now unconditional too (every enemy the unit
could fire at, like `meleeTargets`/`closeFireTargets` already were),
not filtered by current range/arc. These fields are a reach *guide*
now, not a legality gate.

To match: the web viewer accepts a click on *any* hex as a Move order
(Deploy stays restricted to its zone, since placing a reserve unit is
immediate, not a path), and the pending-order ghost for a Move beyond
this turn's reach now shows a straight-line stop *estimate* — "≈
here" — instead of a token sitting exactly at the written objective,
which was misleading when that's further than the unit can actually
reach this turn. The estimate is plain client-side geometry (mirroring
`internal/hex`'s offset-to-cube distance formula, the same category as
the existing `hexCenter`/`hexVertices` SVG math), not a rule decision;
the real stat it scales by (the unit's Move) still comes from the
server.

Added `TestHandleOptionsMoveHexesIgnoreObstacles` (`cmd/wartable-web`)
and `TestExecuteTurnRearUnitPassesThroughVacatedHex` (`internal/game`,
the scenario the dev plan asked for directly: two friendly units in a
column, the rear one ordered beyond the front one — both orders
accepted, and the rear unit correctly passes straight through the hex
the front one vacates earlier in the same turn's execution). The
engine itself needed no change for this: paths were already worked out
against the live board at execution time: only the planning-time
preview was showing a false restriction.

`go vet ./...` and `go test ./...` pass. Verified end-to-end in a real
browser: selected a unit, clicked a hex well outside its green reach
highlight, confirmed the order was accepted and the ghost showed a
dashed line to the true objective with a "≈ here" token partway along
it, then ran the turn and confirmed the unit actually moved its full
Move toward that objective, landing close to the estimate.

## Web viewer: new game button and save management

Closed the long-standing web TODO: a "New game" button, plus a proper
save list instead of a free-text path field.

Extracted `save.NewGame(units, scenario) File` out of
`cmd/wartable/new.go`'s `runNew` (the suggested-army seeding logic) so
both the CLI and the web server build a fresh game the same way —
`cmd/wartable new` is unchanged from the outside, just thinner.

Added `cmd/wartable-web/saves.go`: `GET /api/saves` lists every `*.json`
file in the server's working directory that actually loads as a save
(anything else is silently skipped); `POST /api/saves` creates a new
one (refuses to overwrite an existing file); `DELETE /api/saves?path=`
removes one. All three validate the path is a plain filename with no
directory components, so the web API can never read, write or delete
outside its working directory.

The viewer's header is now a save dropdown (Load/Delete) plus a New
game row (name, Create), replacing the old "Save file" text input.
Deleting needs a second click within 4 seconds to confirm (no native
`confirm()` dialog); deleting the save currently open resets the viewer
to its empty state. On page load, the save list populates and — if
there's exactly one save, the common case — it loads automatically, so
opening the page just works instead of requiring a manual Load click
against a guessed filename.

`go vet ./...` and `go test ./...` pass (new `TestValidSavePath`,
`TestHandleNewGameThenList`, `TestHandleNewGameRefusesOverwrite`,
`TestHandleDeleteSave` and friends in `cmd/wartable-web`, plus
`TestNewGame` in `internal/save`). Verified end-to-end in a real
browser against an isolated directory (not the real `state.json`):
created two games, confirmed both listed with their scenario/turn/unit
summary, loaded one, deleted it with the two-click confirm, and
confirmed the viewer cleared and the list updated.

## Playtest round 2 changes: combat rewrite (dev-plan.md section 7.5)

**Combat rewrite.** Melee and ranged combat are now one comparison each:
margin = attacker's total − defender's total. Margin ≤ 0 is Repelled
(melee only: the attacker is knocked back, no damage) or a Miss
(ranged); 1–2 is a Hit (one hit, then knocked back if the defender
survives); 3+ is Destroyed (two hits, which destroys any unit outright
— at margin 3+ no knockback is attempted at all, even at full
strength). A half-strength unit destroyed by any hit, and the defender
no longer deals damage back when it wins (previously it did). Removed
`att_dmg`/`rng_dmg` from the data, `rules.Unit`, `game.Stats`, and every
view/interface that showed them — `AttackerBase`/`DefenderBase` (melee)
and `ShooterBase`/`TargetBase` (ranged) replace the old `AttDmg`/`RngDmg`
hit/damage-check split. Event log lines now state the totals and the
margin, e.g. `1st Horse (3 +3 rear) vs 2nd Foot (2): margin 4,
destroyed`.

Added `TestAppendixACombatTables` (`internal/game`): computes every
matchup in core-rules.md's Appendix A from the engine (support alone
realizing each net modifier, to isolate the margin-tier-to-label
mapping from the half-strength/ambush nuances already covered
elsewhere) and checks it matches the table exactly — 172 cases, all
passing.

Found and fixed a real bug while regenerating the golden replays:
`applyRanged` discarded `UnitInstance.TakeHit`'s `destroyed` return
value, so a second barrage shot landing on a unit already dropped to
half strength by an earlier shot in the same barrage didn't destroy it
(`TestBarrageStacksMultipleHits` caught this once the destroy-outright
rule made the mismatch visible).

Regenerated both golden replay games for the new rules (results
changed, as expected): `annihilation` now ends at **turn 9** instead of
12 (so `north/south-10.txt` through `-12.txt` were deleted, and
`golden_test.go` now takes a turn count per scenario); `turnlimit` now
ends in a **35-35 draw** instead of North winning 40-35, and its turn-8
double ambush destroys both units outright rather than demonstrating a
blocked knockback (that case is still covered directly by
`TestMeleeEX4DestroyedIfRetreatBlocked` and
`TestAmbushEndsWhenKnockbackWouldBeAdjacentToAnotherEnemy`). See both
`testdata/*/README.md` for the turn-by-turn account.

**Interface.** The order builder now highlights a Close and Attack/Fire
order's shorter half-move reach as purple dots, separate from (and
inside) the full-Move green highlight, labelled "Close: N hexes".
Clicking an enemy to attack or Fire at no longer commits the order
immediately: it shows a predicted result first (new `GET /api/predict`,
which runs the real `game.ExecuteOrder` against the saved state but
never writes it back), labelled as a prediction since the target may
move before the order is actually carried out, with a Confirm button to
add it to the order list.

`go vet ./...` and `go test ./...` pass. Verified end-to-end in a real
browser: deployed both sides, selected a unit eligible for both Close
orders and confirmed the purple close-move dots render distinctly from
the green move highlight, triggered a prediction against a
not-yet-reachable target ("won't make contact this turn") and against
an adjacent target (full margin breakdown, matching a direct
`/api/predict` call), and confirmed the order correctly lands in the
order list.

Regenerated `state.json` via `wartable new` partway through this work
(see the previous changelog entry): worth noting again here since the
stale-save problem it fixed (embedded unit stats not picking up a rules
change) is exactly what this combat rewrite would otherwise have hit
too.

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
