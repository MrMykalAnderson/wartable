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
// unit in it"); Bank is Capacity not spent on the starting army,
// spendable from turn 2 via a buying Deploy order (section 10,
// dev-plan.md section 7.10); PendingCapture records, for a scenario
// with capture objectives, which side (if any) held exclusive capture
// of each enemy town as of the end of the previous turn, since capture
// must be held through the following turn to win (see CheckCapture).
type GameState struct {
	Board          Board
	Reserves       map[string][]UnitInstance
	Objectives     map[string]string
	Bank           map[string]int
	PendingCapture map[string]string
}

// Event is one auditable step of turn execution (docs/dev-plan.md section
// 4): Melee/Ranged carry the exact numbers used when set, Board is the
// board exactly as it stood immediately after this event, for replaying
// a turn step by step (docs/dev-plan.md section 7.3), Path is the hexes
// a "moved" event's unit actually entered, in order (excluding where it
// started), for showing the route it took rather than just where it
// ended up, and Steps/StopReason are the step rule's own record of that
// route — each step's options and why one won the tie, plus why the
// move stopped (dev-plan.md section 7.11) — for the replay's expandable
// step-by-step breakdown.
type Event struct {
	Kind       string
	Unit       string
	Detail     string
	Board      Board
	Path       []hex.Offset
	Steps      []MoveStep
	StopReason string
	Melee      *MeleeResult
	Ranged     *RangedResult
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

func (s GameState) withBank(side string, amount int) GameState {
	next := make(map[string]int, len(s.Bank))
	for k, v := range s.Bank {
		next[k] = v
	}
	next[side] = amount
	s.Bank = next
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

// StepCostFor returns the movement cost to enter a hex (docs/core-
// rules.md section 2.3): a forest hex costs core.Terrain.ForestMoveCost,
// everything else costs 1. Safe to use even when board.Terrain is nil
// (open ground: nothing ever costs more than 1). Exported for the web
// viewer's reach-guide API (dev-plan.md section 7.10, interface item
// "Reach preview uses forest costs").
func StepCostFor(core rules.CoreRules, board Board) func(hex.Offset) int {
	return func(o hex.Offset) int {
		if board.Terrain.IsForest(o) {
			return core.Terrain.ForestMoveCost
		}
		return 1
	}
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

func currentPos(start hex.Offset, path []hex.Offset) hex.Offset {
	if len(path) == 0 {
		return start
	}
	return path[len(path)-1]
}
