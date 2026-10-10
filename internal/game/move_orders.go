package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// applyAmbush resolves contact at a moving unit's stopping point
// (docs/core-rules.md section 7.4): first an ambush by every adjacent
// enemy that can make melee attacks, then — if the mover is still on
// the map and still next to enemies that can't (such as artillery) — an
// overrun, where the mover attacks each of them in turn instead. Events
// from both phases carry the board as it stood right after each combat.
func applyAmbush(board Board, core rules.CoreRules, moverID string) (Board, []Event) {
	board, events := resolveAmbush(board, core, moverID)
	if _, ok := board.Unit(moverID); !ok {
		events = append(events, Event{Kind: "unit-destroyed", Unit: moverID, Detail: "destroyed during ambush", Board: board})
		return board, events
	}
	var overrunEvents []Event
	board, overrunEvents = resolveOverrun(board, core, moverID)
	events = append(events, overrunEvents...)
	if _, ok := board.Unit(moverID); !ok {
		events = append(events, Event{Kind: "unit-destroyed", Unit: moverID, Detail: "destroyed during overrun", Board: board})
	}
	return board, events
}

// executeMove executes a Move order (docs/core-rules.md section 6.2): up
// to its full Move, stopping on contact with any enemy, which ambushes it.
func executeMove(state GameState, core rules.CoreRules, mover UnitInstance, o orders.Order) (GameState, []Event) {
	if !mover.CanMove() {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "cannot move in its current state", Board: state.Board}}
	}
	dest, ok := resolveHexTarget(state.Board, o)
	if !ok {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "move target not found", Board: state.Board}}
	}

	startPos, startFacing := mover.Pos, mover.Facing
	outcome := moveTowards(state.Board, mover, dest, mover.Stats(core).Move, "")

	mover.Pos = currentPos(startPos, outcome.Path)
	mover.Facing = startFacing
	if o.HasFacing {
		mover.Facing = o.Facing
	} else if d, ok := directionOfLastStep(startPos, outcome.Path); ok {
		mover.Facing = d
	}
	state.Board = state.Board.WithUnit(mover)

	events := []Event{{Kind: "moved", Unit: mover.ID, Detail: fmt.Sprintf("moved to %s", mover.Pos), Board: state.Board, Path: outcome.Path}}
	if outcome.Contact {
		var ambushEvents []Event
		state.Board, ambushEvents = applyAmbush(state.Board, core, mover.ID)
		events = append(events, ambushEvents...)
	}
	return state, events
}

// executeCloseAndAttack executes a Close and Attack order
// (docs/core-rules.md section 6.3): closing up to half Move (rounded
// down), then attacking the named target in melee as soon as adjacent to
// it, or being ambushed if it contacts a different enemy first.
func executeCloseAndAttack(state GameState, core rules.CoreRules, mover UnitInstance, o orders.Order) (GameState, []Event) {
	if !mover.Template.Melee {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "cannot make melee attacks", Board: state.Board}}
	}
	target, ok := state.Board.Unit(o.TargetUnit)
	if !ok {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "target not found", Board: state.Board}}
	}

	fight := func(s GameState) (GameState, []Event) {
		result, err := ResolveMelee(s.Board, core, mover.ID, target.ID, false)
		if err != nil {
			return s, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: err.Error(), Board: s.Board}}
		}
		s.Board = applyMelee(s.Board, result)
		return s, []Event{{Kind: "melee", Unit: mover.ID, Detail: "close and attack", Board: s.Board, Melee: &result}}
	}

	if hex.Distance(mover.Pos, target.Pos) == 1 {
		return fight(state)
	}
	if !mover.CanMove() {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "cannot move in its current state", Board: state.Board}}
	}

	startPos := mover.Pos
	half := mover.Stats(core).Move / 2
	outcome := moveTowards(state.Board, mover, target.Pos, half, target.ID)

	mover.Pos = currentPos(startPos, outcome.Path)
	if d, ok := directionOfLastStep(startPos, outcome.Path); ok {
		mover.Facing = d
	} else {
		// Section 4.1 (provisional): a Close order that doesn't move turns
		// the unit toward its target as closely as possible.
		mover.Facing = hex.DirectionToward(startPos, target.Pos)
	}
	state.Board = state.Board.WithUnit(mover)

	events := []Event{}
	if len(outcome.Path) > 0 {
		events = append(events, Event{
			Kind:   "moved",
			Unit:   mover.ID,
			Detail: fmt.Sprintf("closing on %s (now at %s)", target.ID, mover.Pos),
			Board:  state.Board,
			Path:   outcome.Path,
		})
	}

	if hex.Distance(mover.Pos, target.Pos) == 1 {
		s, fightEvents := fight(state)
		return s, append(events, fightEvents...)
	}
	if outcome.Contact {
		var ambushEvents []Event
		state.Board, ambushEvents = applyAmbush(state.Board, core, mover.ID)
		events = append(events, ambushEvents...)
	}
	return state, events
}

// executeFire executes a Fire order (docs/core-rules.md section 6.4): a
// ranged attack against the named target. ResolveRanged itself does
// nothing if the target is out of range or outside the firing arc.
func executeFire(state GameState, core rules.CoreRules, shooter UnitInstance, o orders.Order) (GameState, []Event) {
	if !shooter.CanFire() {
		return state, []Event{{Kind: "order-skipped", Unit: shooter.ID, Detail: "cannot fire in its current state", Board: state.Board}}
	}
	target, ok := state.Board.Unit(o.TargetUnit)
	if !ok {
		return state, []Event{{Kind: "order-skipped", Unit: shooter.ID, Detail: "target not found", Board: state.Board}}
	}
	result, err := ResolveRanged(state.Board, core, shooter.ID, target.ID)
	if err != nil {
		return state, []Event{{Kind: "order-skipped", Unit: shooter.ID, Detail: err.Error(), Board: state.Board}}
	}
	state.Board = applyRanged(state.Board, result)
	return state, []Event{{Kind: "ranged", Unit: shooter.ID, Detail: "fire", Board: state.Board, Ranged: &result}}
}

// executeCloseAndFire executes a Close and Fire order (docs/core-rules.md
// section 6.5): closing up to half Move (rounded down), checking before
// each step and after the last whether it can fire on the named target,
// and firing as soon as it can. Artillery can never use this order.
func executeCloseAndFire(state GameState, core rules.CoreRules, mover UnitInstance, o orders.Order) (GameState, []Event) {
	if len(mover.Template.States) > 0 {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "artillery cannot use Close and Fire", Board: state.Board}}
	}
	target, ok := state.Board.Unit(o.TargetUnit)
	if !ok {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "target not found", Board: state.Board}}
	}

	startPos, startFacing := mover.Pos, mover.Facing
	half := mover.Stats(core).Move / 2
	full := safeShortestPath(state.Board, mover, target.Pos, target.ID)

	facingFor := func(path []hex.Offset) hex.Direction {
		if d, ok := directionOfLastStep(startPos, path); ok {
			return d
		}
		return startFacing
	}
	tryFire := func(pos hex.Offset, path []hex.Offset) (RangedResult, bool) {
		probe := mover
		probe.Pos, probe.Facing = pos, facingFor(path)
		r, err := ResolveRanged(state.Board.WithUnit(probe), core, mover.ID, target.ID)
		return r, err == nil && r.InRange && r.InArc
	}

	var path []hex.Offset
	pos := mover.Pos
	result, fired := tryFire(pos, path)
	contact := false
	if !fired {
		for i := 0; i < half && i < len(full); i++ {
			pos = full[i]
			path = append(path, pos)
			if state.Board.AdjacentEnemy(mover.Side, pos) {
				contact = true
				break
			}
			if result, fired = tryFire(pos, path); fired {
				break
			}
		}
	}

	mover.Pos = pos
	mover.Facing = facingFor(path)
	state.Board = state.Board.WithUnit(mover)

	var events []Event
	if len(path) > 0 {
		events = append(events, Event{
			Kind:   "moved",
			Unit:   mover.ID,
			Detail: fmt.Sprintf("closing on %s to fire (now at %s)", target.ID, mover.Pos),
			Board:  state.Board,
			Path:   path,
		})
	}
	if fired {
		state.Board = applyRanged(state.Board, result)
		events = append(events, Event{Kind: "ranged", Unit: mover.ID, Detail: "close and fire", Board: state.Board, Ranged: &result})
		return state, events
	}
	if contact {
		var ambushEvents []Event
		state.Board, ambushEvents = applyAmbush(state.Board, core, mover.ID)
		events = append(events, ambushEvents...)
	}
	return state, events
}

// checkNoMansLand returns a warning event for every pair of adjacent
// enemy units still on the board (docs/core-rules.md section 5.3's "no
// man's land" table tip): every contact should end in a fight that
// knocks one side back or destroys it, so two enemies still adjacent
// means a rule was missed somewhere. Called after every order and
// after the barrage (docs/dev-plan.md section 7.7).
func checkNoMansLand(board Board) []Event {
	var events []Event
	reported := map[[2]string]bool{}
	for _, u := range board.Units {
		for _, enemy := range board.AdjacentEnemies(u.Side, u.Pos) {
			key := [2]string{u.ID, enemy.ID}
			if u.ID > enemy.ID {
				key = [2]string{enemy.ID, u.ID}
			}
			if reported[key] {
				continue
			}
			reported[key] = true
			events = append(events, Event{
				Kind:   "warning",
				Unit:   u.ID,
				Detail: fmt.Sprintf("no man's land violation: %s and %s are adjacent enemies", key[0], key[1]),
				Board:  board,
			})
		}
	}
	return events
}

// ExecuteOrder executes a single order against state, returning the new
// state and the events it produced.
func ExecuteOrder(state GameState, core rules.CoreRules, scenario rules.Scenario, side string, o orders.Order) (GameState, []Event) {
	if o.Type == orders.Deploy {
		return executeDeploy(state, scenario, side, o)
	}
	mover, ok := state.Board.Unit(o.Unit)
	if !ok {
		return state, []Event{{Kind: "order-skipped", Unit: o.Unit, Detail: "unit destroyed or not on the map", Board: state.Board}}
	}
	switch o.Type {
	case orders.Move:
		return executeMove(state, core, mover, o)
	case orders.CloseAndAttack:
		return executeCloseAndAttack(state, core, mover, o)
	case orders.Fire:
		return executeFire(state, core, mover, o)
	case orders.CloseAndFire:
		return executeCloseAndFire(state, core, mover, o)
	case orders.Ready, orders.Mobilise:
		return executeReadyMobilise(state, mover, o)
	default:
		return state, []Event{{Kind: "order-skipped", Unit: o.Unit, Detail: fmt.Sprintf("unhandled order type %q", o.Type), Board: state.Board}}
	}
}

// ExecuteTurn resolves initiative (advancing tieBreak if it was tied),
// resolves the passive Barrage phase, builds the alternating execution
// order, and executes every order in it (docs/core-rules.md sections
// 5.2-5.3). An order whose unit is no longer on the map (destroyed
// earlier this turn) is skipped but still uses its place in the
// sequence. Every returned event's Board is the board exactly as it
// stood right after that event, for step-by-step replay.
func ExecuteTurn(state GameState, core rules.CoreRules, scenario rules.Scenario, tieBreak *TieBreak, northOrders, southOrders []orders.Order) (GameState, []Event, error) {
	northTotal, err := Initiative(state.Board, core, northOrders)
	if err != nil {
		return state, nil, err
	}
	southTotal, err := Initiative(state.Board, core, southOrders)
	if err != nil {
		return state, nil, err
	}

	var steps []Step
	if tieBreak.Resolve("north", northTotal, "south", southTotal) == "north" {
		steps = BuildExecutionOrder("north", northOrders, "south", southOrders)
	} else {
		steps = BuildExecutionOrder("south", southOrders, "north", northOrders)
	}

	var events []Event
	newBoard, barrageShots := ResolveBarragePhase(state.Board, core)
	state.Board = newBoard
	for _, shot := range barrageShots {
		r := shot.Result
		events = append(events, Event{Kind: "barrage", Unit: r.ShooterID, Detail: "barrage", Board: shot.Board, Ranged: &r})
	}
	events = append(events, checkNoMansLand(state.Board)...)

	for _, step := range steps {
		var stepEvents []Event
		state, stepEvents = ExecuteOrder(state, core, scenario, step.Side, step.Order)
		events = append(events, stepEvents...)
		events = append(events, checkNoMansLand(state.Board)...)
	}
	return state, events, nil
}
