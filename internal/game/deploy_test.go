package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

func TestExecuteDeploySuccess(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)

	unit := reserveUnit(t, templates, "1st Foot", "north", "infantry")
	state := GameState{
		Board:    starterBoard(),
		Reserves: map[string][]UnitInstance{"north": {unit}},
	}
	o := orders.Order{Unit: "1st Foot", Type: orders.Deploy, HasTargetHex: true, TargetHex: mustParse(t, "B2")}

	state, events := executeDeploy(state, scenario, "north", o)

	if len(events) != 1 || events[0].Kind != "deployed" {
		t.Fatalf("events = %+v, want one 'deployed' event", events)
	}
	deployed, ok := state.Board.Unit("1st Foot")
	if !ok {
		t.Fatalf("unit not found on board after deploy")
	}
	if deployed.Pos != mustParse(t, "B2") {
		t.Errorf("Pos = %s, want B2", deployed.Pos)
	}
	if deployed.Facing != hex.S {
		t.Errorf("Facing = %v, want S (starter battle default for north)", deployed.Facing)
	}
	if len(state.Reserves["north"]) != 0 {
		t.Errorf("Reserves[north] = %+v, want empty", state.Reserves["north"])
	}
}

func TestExecuteDeployWithExplicitFacing(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	unit := reserveUnit(t, templates, "1st Foot", "north", "infantry")
	state := GameState{Board: starterBoard(), Reserves: map[string][]UnitInstance{"north": {unit}}}
	o := orders.Order{Unit: "1st Foot", Type: orders.Deploy, HasTargetHex: true, TargetHex: mustParse(t, "B2"), HasFacing: true, Facing: hex.NE}

	state, _ = executeDeploy(state, scenario, "north", o)
	deployed, _ := state.Board.Unit("1st Foot")
	if deployed.Facing != hex.NE {
		t.Errorf("Facing = %v, want NE (explicit)", deployed.Facing)
	}
}

func TestExecuteDeployFailsOutsideZone(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	unit := reserveUnit(t, templates, "1st Foot", "north", "infantry")
	state := GameState{Board: starterBoard(), Reserves: map[string][]UnitInstance{"north": {unit}}}
	// North's zone is rows 1-2; row 5 is outside it.
	o := orders.Order{Unit: "1st Foot", Type: orders.Deploy, HasTargetHex: true, TargetHex: mustParse(t, "B5")}

	state, events := executeDeploy(state, scenario, "north", o)
	if len(events) != 1 || events[0].Kind != "deploy-failed" {
		t.Fatalf("events = %+v, want one 'deploy-failed' event", events)
	}
	if len(state.Reserves["north"]) != 1 {
		t.Errorf("Reserves[north] = %+v, want the unit to stay in reserves", state.Reserves["north"])
	}
}

func TestExecuteDeployFailsOccupied(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	occupant := newUnit(t, templates, "Occupant", "north", "infantry", "B2", hex.S)
	unit := reserveUnit(t, templates, "1st Foot", "north", "infantry")
	state := GameState{Board: starterBoard(occupant), Reserves: map[string][]UnitInstance{"north": {unit}}}
	o := orders.Order{Unit: "1st Foot", Type: orders.Deploy, HasTargetHex: true, TargetHex: mustParse(t, "B2")}

	_, events := executeDeploy(state, scenario, "north", o)
	if len(events) != 1 || events[0].Kind != "deploy-failed" {
		t.Fatalf("events = %+v, want one 'deploy-failed' event", events)
	}
}

func TestExecuteDeployFailsAdjacentEnemy(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	enemy := newUnit(t, templates, "Enemy", "south", "infantry", "B1", hex.N) // adjacent to B2.
	unit := reserveUnit(t, templates, "1st Foot", "north", "infantry")
	state := GameState{Board: starterBoard(enemy), Reserves: map[string][]UnitInstance{"north": {unit}}}
	o := orders.Order{Unit: "1st Foot", Type: orders.Deploy, HasTargetHex: true, TargetHex: mustParse(t, "B2")}

	_, events := executeDeploy(state, scenario, "north", o)
	if len(events) != 1 || events[0].Kind != "deploy-failed" {
		t.Fatalf("events = %+v, want one 'deploy-failed' event", events)
	}
}

func TestExecuteReadyAndMobilise(t *testing.T) {
	templates, _ := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F6", hex.N)
	state := GameState{Board: starterBoard(gun)}

	state, events := executeReadyMobilise(state, gun, orders.Order{Unit: "1st Guns", Type: orders.Ready, HasFacing: true, Facing: hex.SE})
	if len(events) != 1 || events[0].Kind != "state-changed" {
		t.Fatalf("events = %+v, want one 'state-changed' event", events)
	}
	ready, _ := state.Board.Unit("1st Guns")
	if ready.State != "ready" {
		t.Errorf("State = %q, want ready", ready.State)
	}
	if ready.Facing != hex.SE {
		t.Errorf("Facing = %v, want SE (set by the Ready order)", ready.Facing)
	}

	state, _ = executeReadyMobilise(state, ready, orders.Order{Unit: "1st Guns", Type: orders.Mobilise})
	mobilised, _ := state.Board.Unit("1st Guns")
	if mobilised.State != "mobilised" {
		t.Errorf("State = %q, want mobilised", mobilised.State)
	}
}

func TestExecuteReadyInvalidForNonArtillery(t *testing.T) {
	templates, _ := loadTestRules(t)
	foot := newUnit(t, templates, "1st Foot", "north", "infantry", "F6", hex.N)
	state := GameState{Board: starterBoard(foot)}

	_, events := executeReadyMobilise(state, foot, orders.Order{Unit: "1st Foot", Type: orders.Ready})
	if len(events) != 1 || events[0].Kind != "order-skipped" {
		t.Fatalf("events = %+v, want one 'order-skipped' event", events)
	}
}

func mustParse(t *testing.T, s string) hex.Offset {
	t.Helper()
	o, err := hex.ParseOffset(s)
	if err != nil {
		t.Fatalf("hex.ParseOffset(%q): %v", s, err)
	}
	return o
}
