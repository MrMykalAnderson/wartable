package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// GameState is a full game's state: the board plus each side's
// not-yet-deployed units (docs/core-rules.md section 10), plus which
// side owns each objective, if the scenario has any (docs/dev-plan.md
// section 7.8: towns and hamlets, "owned by the last side to have had a
// unit in it").
type GameState struct {
	Board      Board
	Reserves   map[string][]UnitInstance
	Objectives map[string]string
}

// Event is one auditable step of turn execution (docs/dev-plan.md section
// 4): Melee/Ranged carry the exact numbers used when set, Board is the
// board exactly as it stood immediately after this event, for replaying
// a turn step by step (docs/dev-plan.md section 7.3), and Path is the
// hexes a "moved" event's unit actually entered, in order (excluding
// where it started), for showing the route it took rather than just
// where it ended up.
type Event struct {
	Kind   string
	Unit   string
	Detail string
	Board  Board
	Path   []hex.Offset
	Melee  *MeleeResult
	Ranged *RangedResult
}

func (s GameState) withReserve(side string, units []UnitInstance) GameState {
	next := make(map[string][]UnitInstance, len(s.Reserves))
	for k, v := range s.Reserves {
		next[k] = v
	}
	next[side] = units
	s.Reserves = next
	return s
}

// PassableFor is the hex.PassableFunc for mover's movement: on the map,
// and empty except for mover's own current hex (docs/core-rules.md
// section 7.1). River-crossing is a separate, edge-level concern (see
// edgeBlockedFor), not a property of a hex itself. Exported for the web
// viewer's order-preview API.
func PassableFor(board Board, moverID string) hex.PassableFunc {
	return func(o hex.Offset) bool {
		if !board.InBounds(o) {
			return false
		}
		u, occupied := board.UnitAt(o)
		return !occupied || u.ID == moverID
	}
}

// edgeBlockedFor is the hex.EdgeBlockedFunc for board's terrain: a river
// edge with no bridge blocks movement directly across it (docs/core-
// rules.md section 7.1 and 2.3), regardless of whether either hex is
// itself passable. Safe to use even when board.Terrain is nil (open
// ground: nothing is ever blocked).
func edgeBlockedFor(board Board) hex.EdgeBlockedFunc {
	return board.Terrain.RiverBlocks
}

// roadPassable is the hex.PassableFunc for finding a road march's route
// (docs/core-rules.md section 2.3): on the map, and on a road. It
// deliberately ignores unit occupancy — the road network is a sparse
// graph, so routing around a blocking unit the way the safe route does
// could send a road march far out of its way, by some entirely
// different road, instead of simply stopping at the blockage (EX-10's
// note). moveTowards's own stepping loop is what actually stops a road
// march one hex short of a unit in its way.
func roadPassable(board Board) hex.PassableFunc {
	return func(o hex.Offset) bool {
		return board.InBounds(o) && board.Terrain.IsRoad(o)
	}
}

// resolveHexTarget resolves a Move order's detail (a hex, or a unit's
// current hex) to a hex.
func resolveHexTarget(board Board, o orders.Order) (hex.Offset, bool) {
	if o.HasTargetHex {
		return o.TargetHex, true
	}
	u, ok := board.Unit(o.TargetUnit)
	if !ok {
		return hex.Offset{}, false
	}
	return u.Pos, true
}

// directionOfLastStep returns the direction of the last hex entered,
// relative to the one before it (or startPos, if there was only one
// step). ok is false if path is empty.
func directionOfLastStep(startPos hex.Offset, path []hex.Offset) (hex.Direction, bool) {
	if len(path) == 0 {
		return 0, false
	}
	from := startPos
	if len(path) > 1 {
		from = path[len(path)-2]
	}
	return hex.AdjacentDirection(from, path[len(path)-1])
}

// MoveOutcome is how far a unit got toward a destination.
type MoveOutcome struct {
	Path    []hex.Offset // hexes entered, in order; excludes the start.
	Contact bool         // stopped because it entered a hex adjacent to an enemy.
}

// safePassable is the hex.PassableFunc for the safe route (docs/core-
// rules.md section 7.2): passable, and never adjacent to an enemy
// (ignoring any pair separated by a river edge, same as contact itself),
// except dest itself (which may be next to an enemy) and, for a Close
// order, any hex next to closeTarget (empty for a plain Move — the
// unit is trying to get there, so that adjacency doesn't make a hex
// unsafe).
func safePassable(board Board, mover UnitInstance, dest hex.Offset, closeTarget string) hex.PassableFunc {
	base := PassableFor(board, mover.ID)
	return func(o hex.Offset) bool {
		if !base(o) {
			return false
		}
		if o == dest {
			return true
		}
		for _, nb := range board.adjacentHexes(o) {
			u, ok := board.UnitAt(nb)
			if !ok || u.Side == mover.Side {
				continue
			}
			if closeTarget != "" && u.ID == closeTarget {
				continue
			}
			return false
		}
		return true
	}
}

// safeShortestPath finds the route mover actually takes to dest
// (docs/core-rules.md section 7.2): the safe route if one reaches dest
// (judged the whole way there, regardless of how much of it mover can
// cover this turn), else the plain shortest path (with its own
// unreachable-destination fallback). Both respect river edges.
func safeShortestPath(board Board, mover UnitInstance, dest hex.Offset, closeTarget string) []hex.Offset {
	edgeBlocked := edgeBlockedFor(board)
	if path, reached := hex.ShortestPath(mover.Pos, dest, safePassable(board, mover, dest, closeTarget), edgeBlocked); reached {
		return path
	}
	path, _ := hex.ShortestPath(mover.Pos, dest, PassableFor(board, mover.ID), edgeBlocked)
	return path
}

// fordPath checks whether dest is directly across a river edge (with no
// bridge) from mover's current position, and if so and dest is
// currently enterable, returns the one-hex ford path (docs/core-
// rules.md section 2.3: fording is only a Move to the hex directly
// across, as the unit's whole order). ok is false otherwise (not a
// ford, or the far bank is occupied) — the pathfinder never fords on
// its own, so callers fall back to normal pathfinding in that case.
func fordPath(board Board, mover UnitInstance, dest hex.Offset) (path []hex.Offset, ok bool) {
	if _, adjacent := hex.AdjacentDirection(mover.Pos, dest); !adjacent {
		return nil, false
	}
	if !board.Terrain.RiverBlocks(mover.Pos, dest) {
		return nil, false
	}
	if !PassableFor(board, mover.ID)(dest) {
		return nil, false
	}
	return []hex.Offset{dest}, true
}

// roadMarchPath finds the route a road march takes (docs/core-rules.md
// section 2.3): only along road hexes, switching roads where they meet,
// ignoring the safe route rule. ok is false unless mover's own hex and
// dest are both road hexes.
func roadMarchPath(board Board, mover UnitInstance, dest hex.Offset) (path []hex.Offset, ok bool) {
	if !board.Terrain.IsRoad(mover.Pos) || !board.Terrain.IsRoad(dest) {
		return nil, false
	}
	path, _ = hex.ShortestPath(mover.Pos, dest, roadPassable(board), edgeBlockedFor(board))
	return path, true
}

// moveTravelPath picks which of core-rules.md section 2.3's route rules
// applies to a plain Move from mover's position to dest — fording, a
// road march, or (the normal case, and always for a Close order's own
// approach, closeTarget != "") the safe route — and the extra Move it
// grants (a road march's +2; 0 otherwise).
func moveTravelPath(core rules.CoreRules, board Board, mover UnitInstance, dest hex.Offset, closeTarget string) (path []hex.Offset, extraMove int) {
	if closeTarget == "" {
		if p, ok := fordPath(board, mover, dest); ok {
			return p, 0
		}
		if p, ok := roadMarchPath(board, mover, dest); ok {
			return p, core.Terrain.RoadMarchMoveBonus
		}
	}
	return safeShortestPath(board, mover, dest, closeTarget), 0
}

// moveTowards advances mover toward dest, up to maxSteps (before any
// road-march bonus), following fording, a road march or the safe route
// as appropriate (falling back to the plain shortest path if no safe
// route exists), and stopping on contact with any enemy (docs/core-
// rules.md section 7.3). Forest halves the order's total step budget,
// from the start if mover begins there, or from the first forest hex
// entered otherwise — stopping immediately if that budget is already
// used up (section 2.3). closeTarget is the named target of a Close
// order (empty for a plain Move), which only ever takes the safe route.
func moveTowards(core rules.CoreRules, board Board, mover UnitInstance, dest hex.Offset, maxSteps int, closeTarget string) MoveOutcome {
	full, extraMove := moveTravelPath(core, board, mover, dest, closeTarget)
	maxSteps += extraMove

	cap := maxSteps
	forestTriggered := board.Terrain.IsForest(mover.Pos)
	if forestTriggered {
		cap = maxSteps / 2
	}

	var out MoveOutcome
	for i := 0; i < len(full); i++ {
		if len(out.Path) >= cap {
			break
		}
		if u, occupied := board.UnitAt(full[i]); occupied && u.ID != mover.ID {
			// A road march's route ignores occupancy when found (see
			// roadPassable), so this is where it actually stops behind
			// a blocking unit (EX-10's note); a no-op for every other
			// route, whose own pathfinding already avoids occupied
			// hexes.
			break
		}
		out.Path = append(out.Path, full[i])
		if board.AdjacentEnemy(mover.Side, full[i]) {
			out.Contact = true
			break
		}
		if !forestTriggered && board.Terrain.IsForest(full[i]) {
			forestTriggered = true
			cap = maxSteps / 2
			if len(out.Path) >= cap {
				break
			}
		}
	}
	return out
}

func currentPos(start hex.Offset, path []hex.Offset) hex.Offset {
	if len(path) == 0 {
		return start
	}
	return path[len(path)-1]
}
