package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// executeDeploy executes a Deploy order (docs/core-rules.md section 6.1):
// placing a unit from the side's reserves onto an empty hex in its
// deployment zone, not adjacent to an enemy (section 12, provisional rule
// 4).
func executeDeploy(state GameState, scenario rules.Scenario, side string, o orders.Order) (GameState, []Event) {
	reserves := state.Reserves[side]
	idx := -1
	for i, u := range reserves {
		if u.ID == o.Unit {
			idx = i
			break
		}
	}
	if idx == -1 {
		return state, []Event{{Kind: "order-skipped", Unit: o.Unit, Detail: "not in reserves", Board: state.Board}}
	}
	unit := reserves[idx]

	dz, ok := scenario.DeploymentZones[side]
	if !ok || !dz.Contains(o.TargetHex, scenario.Terrain) {
		return state, []Event{{Kind: "deploy-failed", Unit: o.Unit, Detail: fmt.Sprintf("%s is outside %s's deployment zone", o.TargetHex, side), Board: state.Board}}
	}
	if _, occupied := state.Board.UnitAt(o.TargetHex); occupied {
		return state, []Event{{Kind: "deploy-failed", Unit: o.Unit, Detail: fmt.Sprintf("%s is occupied", o.TargetHex), Board: state.Board}}
	}
	if state.Board.AdjacentEnemy(side, o.TargetHex) {
		return state, []Event{{Kind: "deploy-failed", Unit: o.Unit, Detail: fmt.Sprintf("%s is adjacent to an enemy", o.TargetHex), Board: state.Board}}
	}

	facing, _ := orders.ParseDirection(scenario.DefaultFacing[side])
	if o.HasFacing {
		facing = o.Facing
	}

	unit.Pos = o.TargetHex
	unit.Facing = facing
	unit.Strength = Full

	newReserves := append(append([]UnitInstance{}, reserves[:idx]...), reserves[idx+1:]...)
	state = state.withReserve(side, newReserves)
	state.Board = state.Board.WithUnit(unit)

	return state, []Event{{Kind: "deployed", Unit: o.Unit, Detail: fmt.Sprintf("deployed to %s facing %s", o.TargetHex, facing), Board: state.Board}}
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
