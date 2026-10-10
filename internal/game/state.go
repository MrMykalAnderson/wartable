package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// GameState is a full game's state: the board plus each side's
// not-yet-deployed units (docs/core-rules.md section 10).
type GameState struct {
	Board    Board
	Reserves map[string][]UnitInstance
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
// and empty except for mover's own current hex (docs/core-rules.md section
// 7.1). Exported for the web viewer's order-preview API.
func PassableFor(board Board, moverID string) hex.PassableFunc {
	return func(o hex.Offset) bool {
		if !board.InBounds(o) {
			return false
		}
		u, occupied := board.UnitAt(o)
		return !occupied || u.ID == moverID
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
// rules.md section 7.2): passable, and never adjacent to an enemy,
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
		for _, nb := range o.Neighbors() {
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
// unreachable-destination fallback).
func safeShortestPath(board Board, mover UnitInstance, dest hex.Offset, closeTarget string) []hex.Offset {
	if path, reached := hex.ShortestPath(mover.Pos, dest, safePassable(board, mover, dest, closeTarget)); reached {
		return path
	}
	path, _ := hex.ShortestPath(mover.Pos, dest, PassableFor(board, mover.ID))
	return path
}

// moveTowards advances mover toward dest, up to maxSteps, following the
// safe route if one exists (falling back to the plain shortest path,
// with the docs/core-rules.md section 7.2 unreachable-destination
// fallback) and stopping on contact with any enemy (section 7.3).
// closeTarget is the named target of a Close order (empty for a plain
// Move), whose adjacency doesn't make a hex unsafe.
func moveTowards(board Board, mover UnitInstance, dest hex.Offset, maxSteps int, closeTarget string) MoveOutcome {
	full := safeShortestPath(board, mover, dest, closeTarget)
	var out MoveOutcome
	for i := 0; i < len(full) && i < maxSteps; i++ {
		out.Path = append(out.Path, full[i])
		if board.AdjacentEnemy(mover.Side, full[i]) {
			out.Contact = true
			break
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
