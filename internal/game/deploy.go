package game

import (
	"fmt"
	"strings"

	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// executeDeploy executes a Deploy order (docs/core-rules.md section
// 6.1): placing a unit onto an empty hex in its deployment zone, not
// adjacent to an enemy (section 12, provisional rule 4). The unit is
// either one of the side's reserves, named by its exact ID, or — from
// turn 2 (dev-plan.md section 7.10) — a unit type allowed in the
// scenario, bought from the bank and named with the next free ordinal
// (e.g. "3rd Cavalry"). A Deploy that fails buys and pays nothing.
//
// RULES-QUESTION (core-rules.md section 12): core-rules.md section 10's
// own example of a buying Deploy, "Deploy | Cavalry | X3", puts the
// order type first, unlike every other order (always "unit | order |
// detail"). This keeps the established field order instead: a buying
// order reads "<unit type> | Deploy | <hex>" (e.g. "Cavalry | Deploy |
// X3"), resolved here by checking the Unit field against reserves
// first, then against the scenario's allowed unit types — no change to
// internal/orders' parser was needed.
func executeDeploy(state GameState, units map[string]rules.Unit, scenario rules.Scenario, turn int, side string, o orders.Order) (GameState, []Event) {
	reserves := state.Reserves[side]
	idx := -1
	for i, u := range reserves {
		if u.ID == o.Unit {
			idx = i
			break
		}
	}

	var unit UnitInstance
	buying := idx == -1
	if idx != -1 {
		unit = reserves[idx]
	} else {
		tmpl, ok := units[o.Unit]
		if ok {
			ok = false
			for _, id := range scenario.UnitsAllowed {
				if id == o.Unit {
					ok = true
					break
				}
			}
		}
		if !ok {
			return state, []Event{{Kind: "order-skipped", Unit: o.Unit, Detail: "not in reserves", Board: state.Board}}
		}
		if turn < 2 {
			return state, []Event{{Kind: "order-skipped", Unit: o.Unit, Detail: "can't buy a unit before turn 2", Board: state.Board}}
		}
		if state.Bank[side] < tmpl.Cost {
			return state, []Event{{Kind: "deploy-failed", Unit: o.Unit, Detail: fmt.Sprintf("not enough in the bank for a %s (%d, have %d)", tmpl.Name, tmpl.Cost, state.Bank[side]), Board: state.Board}}
		}
		unit = UnitInstance{ID: nextUnitName(state, side, o.Unit), Side: side, Template: tmpl, Strength: Full, State: tmpl.DeployState}
	}

	dz, ok := scenario.DeploymentZones[side]
	if !ok || !dz.Contains(o.TargetHex, scenario.Terrain) {
		return state, []Event{{Kind: "deploy-failed", Unit: unit.ID, Detail: fmt.Sprintf("%s is outside %s's deployment zone", o.TargetHex, side), Board: state.Board}}
	}
	if _, occupied := state.Board.UnitAt(o.TargetHex); occupied {
		return state, []Event{{Kind: "deploy-failed", Unit: unit.ID, Detail: fmt.Sprintf("%s is occupied", o.TargetHex), Board: state.Board}}
	}
	if state.Board.AdjacentEnemy(side, o.TargetHex) {
		return state, []Event{{Kind: "deploy-failed", Unit: unit.ID, Detail: fmt.Sprintf("%s is adjacent to an enemy", o.TargetHex), Board: state.Board}}
	}

	facing, _ := orders.ParseDirection(scenario.DefaultFacing[side])
	if o.HasFacing {
		facing = o.Facing
	}

	unit.Pos = o.TargetHex
	unit.Facing = facing
	unit.Strength = Full

	if buying {
		state = state.withBank(side, state.Bank[side]-unit.Template.Cost)
	} else {
		newReserves := append(append([]UnitInstance{}, reserves[:idx]...), reserves[idx+1:]...)
		state = state.withReserve(side, newReserves)
	}
	state.Board = state.Board.WithUnit(unit)

	detail := fmt.Sprintf("deployed to %s facing %s", o.TargetHex, facing)
	if buying {
		detail = fmt.Sprintf("bought %s and %s", unit.Template.Name, detail)
	}
	return state, []Event{{Kind: "deployed", Unit: unit.ID, Detail: detail, Board: state.Board}}
}

// nextUnitName generates a newly bought unit's name (docs/core-rules.md
// section 10): "<Side> <ordinal> <Type>", continuing the numbering of
// existing units of the same side and template — matching
// save.NewGame's naming for the starting army.
func nextUnitName(state GameState, side, templateID string) string {
	count := 0
	for _, u := range state.Board.Units {
		if u.Side == side && u.Template.ID == templateID {
			count++
		}
	}
	for _, u := range state.Reserves[side] {
		if u.Template.ID == templateID {
			count++
		}
	}
	return fmt.Sprintf("%s %s %s", capitalize(side), ordinal(count+1), capitalize(templateID))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// executeReadyMobilise executes a Ready or Mobilise order
// (docs/core-rules.md section 3.3): switching an artillery unit's state,
// optionally also setting its facing (section 12, provisional rule 2).
func executeReadyMobilise(state GameState, mover UnitInstance, o orders.Order) (GameState, []Event) {
	var newState string
	switch o.Type {
	case orders.Ready:
		newState = "ready"
	case orders.Mobilise:
		newState = "mobilised"
	}
	if _, ok := mover.Template.States[newState]; !ok {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: fmt.Sprintf("%s has no %s state", mover.ID, newState), Board: state.Board}}
	}
	if newState == "ready" && state.Board.Terrain.IsForest(mover.Pos) {
		return state, []Event{{Kind: "order-skipped", Unit: mover.ID, Detail: "cannot go ready in forest", Board: state.Board}}
	}

	from := mover.State
	mover.State = newState
	if o.HasFacing {
		mover.Facing = o.Facing
	}
	state.Board = state.Board.WithUnit(mover)

	return state, []Event{{Kind: "state-changed", Unit: mover.ID, Detail: fmt.Sprintf("%s -> %s", from, newState), Board: state.Board}}
}
