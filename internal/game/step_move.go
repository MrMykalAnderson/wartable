package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// MoveStepOption is one open neighbour considered for a single step of
// the step rule (docs/core-rules.md section 7.2), with notes explaining
// why it might lose a tie ("forest", "next to enemy"), for the engine's
// own record of what it did (dev-plan.md section 7.11).
type MoveStepOption struct {
	Hex   hex.Offset
	Notes []string
}

// MoveStep is one step of an executed move: the hex entered, what kind
// of step it was ("closer", "slide" or "road"), the options it was
// chosen from, and why it won the tie ("only option", "avoids the
// enemy", "avoids the forest" or "keep left").
type MoveStep struct {
	Hex     hex.Offset
	Dir     hex.Direction
	Kind    string
	Reason  string
	Options []MoveStepOption
}

// MoveResult is the outcome of a full step-rule walk toward a
// destination (docs/core-rules.md section 7): Path is the hexes entered,
// in order (excluding the start); Steps is the matching per-step record
// for the event log; Contact is whether it stopped because it entered a
// hex adjacent to an enemy (7.3-7.4); StopReason is why it stopped, for
// display.
type MoveResult struct {
	Path       []hex.Offset
	Steps      []MoveStep
	Contact    bool
	StopReason string
}

// openOption is a candidate neighbour considered mid-step: open (not
// closed per 7.1), paired with the direction it lies in from the hex
// being stepped from, since keepLeft needs that direction to break ties.
type openOption struct {
	dir hex.Direction
	hex hex.Offset
}

// StepMove walks mover toward dest one step at a time by the step rule
// (docs/core-rules.md section 7.2), up to budget worth of movement cost,
// stopping at the destination, on contact, when the next step can't be
// paid, or when there's no open option (7.3). closeTarget is a Close
// order's named target (empty for a plain Move): adjacency to it is
// exempt from the tie order and from counting as contact, and the walk
// also stops as soon as it's adjacent to it (6.3, 7.2). A road march
// (2.3) applies automatically when this is a plain Move (closeTarget
// == "") starting and ending on road hexes, adding
// core.Terrain.RoadMarchMoveBonus to budget.
func StepMove(core rules.CoreRules, board Board, mover UnitInstance, dest hex.Offset, budget int, closeTarget string) MoveResult {
	var targetPos hex.Offset
	hasTarget := false
	if closeTarget != "" {
		if t, ok := board.Unit(closeTarget); ok {
			targetPos, hasTarget = t.Pos, true
		}
	}
	roadMarch := !hasTarget && board.Terrain.IsRoad(mover.Pos) && board.Terrain.IsRoad(dest)
	if roadMarch {
		budget += core.Terrain.RoadMarchMoveBonus
	}

	tieExempt := ""
	if hasTarget {
		tieExempt = closeTarget
	}
	nextToTarget := func(h hex.Offset) bool {
		if !hasTarget || board.Terrain.RiverBlocks(h, targetPos) {
			return false
		}
		_, adjacent := hex.AdjacentDirection(h, targetPos)
		return adjacent
	}

	cur := mover.Pos
	visited := map[hex.Offset]bool{cur: true}
	var result MoveResult
	used := 0
	for {
		if cur == dest {
			result.StopReason = "reached the destination"
			return result
		}
		if nextToTarget(cur) {
			result.StopReason = "next to the target"
			return result
		}

		step, ok := nextMoveStep(board, mover, cur, dest, visited, tieExempt, roadMarch)
		if !ok {
			if roadMarch {
				result.StopReason = "the road ahead is blocked or ends"
			} else {
				result.StopReason = "no open hex gets any closer"
			}
			return result
		}

		cost := 1
		if !roadMarch {
			cost = StepCostFor(core, board)(step.Hex)
		}
		if len(result.Path) > 0 && used+cost > budget {
			result.StopReason = fmt.Sprintf("can't pay %d for %s (%d Move left)", cost, step.Hex, budget-used)
			return result
		}
		used += cost
		result.Path = append(result.Path, step.Hex)
		result.Steps = append(result.Steps, step)
		visited[step.Hex] = true
		cur = step.Hex

		// A hex adjacent to an enemy is contact (7.4) whether or not it's
		// also the destination: reaching the destination still ambushes a
		// unit that lands next to an enemy, so Contact is set regardless,
		// even though "reached the destination" is reported as why it
		// stopped (the reference model's own wording for this case).
		contact := board.AdjacentEnemy(mover.Side, cur) && !nextToTarget(cur)
		if cur == dest {
			result.Contact = contact
			result.StopReason = "reached the destination"
			return result
		}
		if contact {
			result.Contact = true
			result.StopReason = fmt.Sprintf("contact at %s", cur)
			return result
		}
		if used >= budget {
			result.StopReason = "used all its Move"
			return result
		}
	}
}

// nextMoveStep decides a single step of the step rule from cur toward
// dest (docs/core-rules.md section 7.2). tieExempt is a Close order's
// named target (empty for a plain Move): adjacency to it doesn't count
// against a hex in the tie order. ok is false if there's no open option
// (closer, or failing that a slide; for a road march, the next road
// link).
func nextMoveStep(board Board, mover UnitInstance, cur, dest hex.Offset, visited map[hex.Offset]bool, tieExempt string, roadMarch bool) (MoveStep, bool) {
	passable := PassableFor(board, mover.ID)
	var open []openOption
	for _, d := range hex.Directions {
		o := cur.Neighbor(d)
		if visited[o] || !passable(o) || board.Terrain.RiverBlocks(cur, o) {
			continue
		}
		open = append(open, openOption{dir: d, hex: o})
	}

	var opts []openOption
	kind := ""
	if roadMarch {
		kind = "road"
		for _, o := range open {
			if board.Terrain.IsRoadLink(cur, o.hex) {
				opts = append(opts, o)
			}
		}
		if len(opts) == 0 {
			return MoveStep{}, false
		}
		if len(opts) > 1 {
			best := hex.Distance(opts[0].hex, dest)
			for _, o := range opts[1:] {
				if d := hex.Distance(o.hex, dest); d < best {
					best = d
				}
			}
			var closer []openOption
			for _, o := range opts {
				if hex.Distance(o.hex, dest) == best {
					closer = append(closer, o)
				}
			}
			opts = closer
		}
	} else {
		here := hex.Distance(cur, dest)
		var closer, slide []openOption
		for _, o := range open {
			d := hex.Distance(o.hex, dest)
			switch {
			case d < here:
				closer = append(closer, o)
			case d == here && hex.CenterDistance(o.hex, dest) < hex.CenterDistance(cur, dest)-1e-9:
				slide = append(slide, o)
			}
		}
		if len(closer) > 0 {
			opts, kind = closer, "closer"
		} else {
			opts, kind = slide, "slide"
		}
		if len(opts) == 0 {
			return MoveStep{}, false
		}
	}

	options := make([]MoveStepOption, len(opts))
	for i, o := range opts {
		options[i] = MoveStepOption{Hex: o.hex, Notes: stepNotes(board, mover.Side, o.hex, tieExempt)}
	}

	reason := ""
	if len(opts) == 1 {
		reason = "only option"
	}
	pick := opts
	if len(pick) > 1 {
		var safe []openOption
		for _, o := range pick {
			if !nextToEnemy(board, mover.Side, o.hex, tieExempt) {
				safe = append(safe, o)
			}
		}
		if len(safe) > 0 && len(safe) < len(pick) {
			pick, reason = safe, "avoids the enemy"
		}
	}
	if len(pick) > 1 {
		var clear []openOption
		for _, o := range pick {
			if !board.Terrain.IsForest(o.hex) {
				clear = append(clear, o)
			}
		}
		if len(clear) > 0 && len(clear) < len(pick) {
			pick, reason = clear, "avoids the forest"
		}
	}
	if len(pick) > 1 {
		pick, reason = []openOption{keepLeft(pick)}, "keep left"
	}

	chosen := pick[0]
	return MoveStep{Hex: chosen.hex, Dir: chosen.dir, Kind: kind, Reason: reason, Options: options}, true
}

// stepNotes annotates a candidate hex for the event log: "forest" and/or
// "next to enemy" (ignoring tieExempt), in that order.
func stepNotes(board Board, side string, h hex.Offset, tieExempt string) []string {
	var notes []string
	if board.Terrain.IsForest(h) {
		notes = append(notes, "forest")
	}
	if nextToEnemy(board, side, h, tieExempt) {
		notes = append(notes, "next to enemy")
	}
	return notes
}

// nextToEnemy reports whether any unit not of side (other than
// tieExempt, a Close order's named target) is adjacent to h, respecting
// river edges the same way contact itself does (docs/core-rules.md
// section 7.2's tie order and 2.3's "across a river").
func nextToEnemy(board Board, side string, h hex.Offset, tieExempt string) bool {
	for _, nb := range board.adjacentHexes(h) {
		if u, ok := board.UnitAt(nb); ok && u.Side != side && u.ID != tieExempt {
			return true
		}
	}
	return false
}

// keepLeft picks, from tied options, the one anticlockwise of all the
// others as the unit faces its destination (docs/core-rules.md section
// 7.2's final tiebreak): the candidate c such that every other option's
// direction is 1 or 2 steps clockwise of c's, in hex.Directions' order.
// Falls back to the first option if none qualifies (shouldn't happen
// with well-formed tie sets).
func keepLeft(opts []openOption) openOption {
	for _, c := range opts {
		allClockwise := true
		for _, o := range opts {
			if o == c {
				continue
			}
			diff := (((int(o.dir) - int(c.dir)) % 6) + 6) % 6
			if diff != 1 && diff != 2 {
				allClockwise = false
				break
			}
		}
		if allClockwise {
			return c
		}
	}
	return opts[0]
}
