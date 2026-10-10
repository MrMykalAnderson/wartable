package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// TestRoadMarchEX10 checks EX-10 from docs/core-rules.md section 11:
// North's 2nd Foot (infantry, Move 4) at K7 on the road, ordered to
// S5, also on the road: a road march with Move 4+2=6, following the
// road through the bridge at M7|N6 to end at Q5 (six steps).
func TestRoadMarchEX10(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "2nd Foot", "north", "infantry", "K7", hex.N)
	state := GameState{Board: boardFor(scenario, mover)}

	o := orders.Order{Unit: "2nd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "S5")}
	state, _ = executeMove(state, core, mover, o)

	moved, ok := state.Board.Unit("2nd Foot")
	if !ok {
		t.Fatalf("2nd Foot missing after move")
	}
	if moved.Pos != mustParse(t, "Q5") {
		t.Errorf("2nd Foot.Pos = %s, want Q5 (road march, Move 4+2=6)", moved.Pos)
	}
}

// TestRoadMarchStopsBehindBlockingUnit checks EX-10's note: a friendly
// unit on the road blocks a road march, which stops behind it.
func TestRoadMarchStopsBehindBlockingUnit(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "2nd Foot", "north", "infantry", "K7", hex.N)
	blocker := newUnit(t, templates, "Blocker", "north", "infantry", "N6", hex.N)
	state := GameState{Board: boardFor(scenario, mover, blocker)}

	o := orders.Order{Unit: "2nd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "S5")}
	state, _ = executeMove(state, core, mover, o)

	moved, _ := state.Board.Unit("2nd Foot")
	if moved.Pos != mustParse(t, "M7") {
		t.Errorf("2nd Foot.Pos = %s, want M7 (stopped behind the unit blocking N6)", moved.Pos)
	}
}

// TestPreviewMovePathIgnoresBlockingUnit checks docs/dev-plan.md section
// 7.9: unlike actually executing the order (TestRoadMarchStopsBehindBlockingUnit),
// a Move prediction's path estimate ignores every unit, so a friendly
// unit standing on the road doesn't shorten the previewed road march.
func TestPreviewMovePathIgnoresBlockingUnit(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "2nd Foot", "north", "infantry", "K7", hex.N)
	blocker := newUnit(t, templates, "Blocker", "north", "infantry", "N6", hex.N)
	board := boardFor(scenario, mover, blocker)

	path := PreviewMovePath(core, board, mover, mustParse(t, "S5"))
	if len(path) == 0 || path[len(path)-1] != mustParse(t, "Q5") {
		t.Errorf("PreviewMovePath = %v, want it to reach Q5 regardless of the blocker on N6", path)
	}
}

// TestFordingEX11 checks EX-11 from docs/core-rules.md section 11:
// North's 3rd Foot (infantry) at K6, on the river bank, ordered to L5
// directly across the (unbridged) river edge: it fords, moving that
// one hex and stopping.
func TestFordingEX11(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "north", "infantry", "K6", hex.N)
	state := GameState{Board: boardFor(scenario, mover)}

	o := orders.Order{Unit: "3rd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "L5")}
	state, events := executeMove(state, core, mover, o)

	moved, _ := state.Board.Unit("3rd Foot")
	if moved.Pos != mustParse(t, "L5") {
		t.Errorf("3rd Foot.Pos = %s, want L5 (forded)", moved.Pos)
	}
	for _, e := range events {
		if e.Kind == "melee" || e.Kind == "unit-destroyed" {
			t.Errorf("events = %+v, want no combat (L5 is empty)", events)
		}
	}
}

// TestFordingNeverAutomatic checks EX-11's note: an order to a hex not
// directly across the river (M4) is not a ford, and the pathfinder
// never fords on its own — it must go by the bridge instead, so it
// never takes the direct K6->L5 hop.
func TestFordingNeverAutomatic(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "north", "infantry", "K6", hex.N)
	board := boardFor(scenario, mover)

	if _, ok := fordPath(board, mover, mustParse(t, "M4")); ok {
		t.Errorf("fordPath(K6, M4) = ok, want false (M4 is not directly across a river edge from K6)")
	}
	path := safeShortestPath(board, mover, mustParse(t, "M4"), "")
	if len(path) > 0 && path[0] == mustParse(t, "L5") {
		t.Errorf("safeShortestPath(K6, M4) = %v, want it not to cross the unbridged K6|L5 river edge", path)
	}
}

// TestFordingBlockedWhenFarBankOccupied checks EX-11's note: 3rd Foot
// can't ford into L5 while it's occupied.
func TestFordingBlockedWhenFarBankOccupied(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "north", "infantry", "K6", hex.N)
	occupant := newUnit(t, templates, "Occupant", "south", "infantry", "L5", hex.S)
	board := boardFor(scenario, mover, occupant)

	if _, ok := fordPath(board, mover, mustParse(t, "L5")); ok {
		t.Errorf("fordPath(K6, L5) = ok, want false (L5 is occupied)")
	}
}

// TestRiverBlocksContactAcrossUnbridgedEdge checks EX-11's note: an
// enemy across the river (K6/L5) isn't in contact, but one at L4 (next
// to L5, no river between them) is, once 3rd Foot fords into L5.
func TestRiverBlocksContactAcrossUnbridgedEdge(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "north", "infantry", "K6", hex.N)
	acrossRiver := newUnit(t, templates, "AcrossRiver", "south", "infantry", "L5", hex.S)
	board := boardFor(scenario, mover, acrossRiver)

	if board.AdjacentEnemy("north", mover.Pos) {
		t.Errorf("AdjacentEnemy(K6) = true, want false (L5 is across an unbridged river edge)")
	}

	fordedBoard := board.WithoutUnit("AcrossRiver").WithUnit(newUnit(t, templates, "AtL4", "south", "infantry", "L4", hex.S))
	if !fordedBoard.AdjacentEnemy("north", mustParse(t, "L5")) {
		t.Errorf("AdjacentEnemy(L5) = false, want true (L4 is a plain neighbour of L5, no river between them)")
	}
}

// TestTerrainDefBonus checks docs/core-rules.md 2.3: defenders in
// forest, a town or a hamlet get +1 Def (melee and ranged), and the
// three don't stack.
func TestTerrainDefBonus(t *testing.T) {
	_, core := loadTestRules(t)
	templates, _ := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)

	cases := []struct {
		name string
		pos  string
		want int
	}{
		{"forest", "L8", core.Terrain.ForestDefBonus},
		{"town", "B15", core.Terrain.TownDefBonus},
		{"hamlet", "B11", core.Terrain.HamletDefBonus},
		{"open ground", "A1", 0},
	}
	for _, c := range cases {
		got := terrainDefBonus(core.Terrain, scenario.Terrain, mustParse(t, c.pos))
		if got != c.want {
			t.Errorf("%s (%s): terrainDefBonus = %d, want %d", c.name, c.pos, got, c.want)
		}
	}
}

// TestCavalryForestPenalty and TestBridgeAttackPenalty check docs/core-
// rules.md 2.3's melee modifiers via ResolveMelee directly, against the
// real Two Towns map.
func TestCavalryForestPenalty(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	attacker := newUnit(t, templates, "Rider", "south", "cavalry", "M8", hex.N) // forest.
	defender := newUnit(t, templates, "Foot", "north", "infantry", "L8", hex.S) // forest, N of attacker: defender faces S, attacker is on its N edge (front).
	board := boardFor(scenario, attacker, defender)

	result, err := ResolveMelee(board, core, "Rider", "Foot", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	if result.CavalryForestPenalty != core.Terrain.CavalryForestPenalty {
		t.Errorf("CavalryForestPenalty = %d, want %d (cavalry attacking in forest)", result.CavalryForestPenalty, core.Terrain.CavalryForestPenalty)
	}
}

func TestBridgeAttackPenalty(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	attacker := newUnit(t, templates, "Attacker", "north", "infantry", "M7", hex.N)
	defender := newUnit(t, templates, "Defender", "south", "infantry", "N6", hex.S)
	board := boardFor(scenario, attacker, defender)

	result, err := ResolveMelee(board, core, "Attacker", "Defender", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	if result.BridgeAttackPenalty != core.Terrain.BridgeAttackPenalty {
		t.Errorf("BridgeAttackPenalty = %d, want %d (attacking across the M7|N6 bridge)", result.BridgeAttackPenalty, core.Terrain.BridgeAttackPenalty)
	}
}

// TestArtilleryCannotReadyInForest checks docs/core-rules.md 2.3.
func TestArtilleryCannotReadyInForest(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	gun := newUnit(t, templates, "Gun", "north", "artillery", "L8", hex.N) // forest.
	gun.State = "mobilised"
	state := GameState{Board: boardFor(scenario, gun)}

	o := orders.Order{Unit: "Gun", Type: orders.Ready}
	state, events := executeReadyMobilise(state, gun, o)

	stillMobilised, _ := state.Board.Unit("Gun")
	if stillMobilised.State != "mobilised" {
		t.Errorf("Gun.State = %q, want still mobilised (can't go Ready in forest)", stillMobilised.State)
	}
	if len(events) != 1 || events[0].Kind != "order-skipped" {
		t.Errorf("events = %+v, want one order-skipped event", events)
	}
}

// TestForestHalvesMove checks docs/core-rules.md 2.3: a unit that
// starts in forest, or enters it, may move only half its Move
// (rounded down) that order.
func TestForestHalvesMove(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)

	// Infantry (Move 4) starting in forest (L8): capped at 2.
	startsInForest := newUnit(t, templates, "A", "north", "infantry", "L8", hex.N)
	state := GameState{Board: boardFor(scenario, startsInForest)}
	o := orders.Order{Unit: "A", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "L4")}
	state, _ = executeMove(state, core, startsInForest, o)
	moved, _ := state.Board.Unit("A")
	if got := hex.Distance(mustParse(t, "L8"), moved.Pos); got != 2 {
		t.Errorf("unit starting in forest moved %d hexes, want 2 (half of Move 4)", got)
	}

	// Infantry starting in open ground (K8) ordered toward N10, straight
	// through forest the whole way (L8, M9, N9, N10 are all forest, with
	// no river crossing to detour around): it enters forest on the very
	// first step, capping the whole order at half Move (2), so it stops
	// at M9, two steps in, well short of N10.
	entersForest := newUnit(t, templates, "B", "north", "infantry", "K8", hex.N)
	if scenario.Terrain.IsForest(mustParse(t, "K8")) {
		t.Fatalf("test setup: K8 should be open ground, not forest")
	}
	state2 := GameState{Board: boardFor(scenario, entersForest)}
	o2 := orders.Order{Unit: "B", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "N10")}
	state2, _ = executeMove(state2, core, entersForest, o2)
	moved2, _ := state2.Board.Unit("B")
	if moved2.Pos != mustParse(t, "M9") {
		t.Errorf("unit entering forest on its first step ended at %s, want M9 (half of Move 4, stopped two steps in)", moved2.Pos)
	}
}
