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

// TestGoingRoundTheForestEX12 checks EX-12 from docs/core-rules.md
// section 11: South's 1st Guns (artillery, Mobilised, Move 4) at Q11
// with a Move order to S9. The direct route through the forest
// (Q10, R9, S9, cost 2+2+1=5) costs more than Move 4, so the cheapest
// route goes round the edge instead (R10, S10, S9, cost 1+1+1=3),
// reaching S9 this turn.
func TestGoingRoundTheForestEX12(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	guns := newUnit(t, templates, "1st Guns", "south", "artillery", "Q11", hex.S)
	state := GameState{Board: boardFor(scenario, guns)}

	o := orders.Order{Unit: "1st Guns", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "S9")}
	state, events := executeMove(state, core, guns, o)

	moved, ok := state.Board.Unit("1st Guns")
	if !ok {
		t.Fatalf("1st Guns missing after move")
	}
	if moved.Pos != mustParse(t, "S9") {
		t.Errorf("1st Guns.Pos = %s, want S9 (the cheaper route round the forest)", moved.Pos)
	}
	for _, e := range events {
		if e.Kind == "moved" {
			want := []hex.Offset{mustParse(t, "R10"), mustParse(t, "S10"), mustParse(t, "S9")}
			if len(e.Path) != len(want) {
				t.Fatalf("path = %v, want %v (round the edge, not through the forest)", e.Path, want)
			}
			for i, h := range want {
				if e.Path[i] != h {
					t.Errorf("path[%d] = %s, want %s (full path %v)", i, e.Path[i], h, e.Path)
				}
			}
		}
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
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "north", "infantry", "K6", hex.N)
	board := boardFor(scenario, mover)

	if _, ok := fordPath(board, mover, mustParse(t, "M4")); ok {
		t.Errorf("fordPath(K6, M4) = ok, want false (M4 is not directly across a river edge from K6)")
	}
	path := StepMove(core, board, mover, mustParse(t, "M4"), mover.Stats(core).Move, "").Path
	if len(path) > 0 && path[0] == mustParse(t, "L5") {
		t.Errorf("StepMove(K6, M4) = %v, want it not to cross the unbridged K6|L5 river edge", path)
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

// TestForestMoveCost checks docs/core-rules.md 2.3 (dev-plan.md section
// 7.10): entering a forest hex costs 2 Move, every other hex costs 1,
// and a unit stops once it can't pay for its next step.
func TestForestMoveCost(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)

	// Infantry (Move 4) starting in open ground (K8) ordered toward
	// N10, straight through forest the whole way (L8, M9, N9, N10 are
	// all forest, with no river crossing to detour around): each step
	// costs 2, so with Move 4 it affords exactly two of them, stopping
	// at M9.
	entersForest := newUnit(t, templates, "B", "north", "infantry", "K8", hex.N)
	if scenario.Terrain.IsForest(mustParse(t, "K8")) {
		t.Fatalf("test setup: K8 should be open ground, not forest")
	}
	state := GameState{Board: boardFor(scenario, entersForest)}
	o := orders.Order{Unit: "B", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "N10")}
	state, _ = executeMove(state, core, entersForest, o)
	moved, _ := state.Board.Unit("B")
	if moved.Pos != mustParse(t, "M9") {
		t.Errorf("unit entering forest moved to %s, want M9 (Move 4, two forest hexes at cost 2 each)", moved.Pos)
	}
}

// TestForestMoveCostAlwaysAtLeastOneHex checks docs/core-rules.md 2.3's
// "every unit can always move at least one hex per order, whatever it
// costs": with forest cost inflated past the unit's entire Move, the
// unit still takes that first step.
func TestForestMoveCostAlwaysAtLeastOneHex(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	core.Terrain.ForestMoveCost = 10 // well past infantry's Move (4).

	unit := newUnit(t, templates, "A", "north", "infantry", "K8", hex.N)
	if scenario.Terrain.IsForest(mustParse(t, "K8")) {
		t.Fatalf("test setup: K8 should be open ground, not forest")
	}
	state := GameState{Board: boardFor(scenario, unit)}
	o := orders.Order{Unit: "A", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "L8")}
	state, _ = executeMove(state, core, unit, o)
	moved, _ := state.Board.Unit("A")
	if moved.Pos != mustParse(t, "L8") {
		t.Errorf("unit moved to %s, want L8 (one hex, however much forest costs)", moved.Pos)
	}
}
