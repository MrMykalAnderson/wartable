package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// TestExecuteTurnRearUnitPassesThroughVacatedHex checks docs/dev-plan.md
// section 7.6: two friendly units in a column, the rear one ordered to a
// hex beyond the front one. Both orders are valid (planning can't know
// the front unit will have moved out of the way by the time the rear
// one's path is actually worked out, at execution); since both orders
// are North's and run in the order written, the front unit moves first,
// so the rear unit can pass straight through the hex it vacates.
func TestExecuteTurnRearUnitPassesThroughVacatedHex(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestScenario(t, templates)

	front := newUnit(t, templates, "Front", "north", "infantry", "F5", hex.N)
	rear := newUnit(t, templates, "Rear", "north", "infantry", "F6", hex.N)
	state := GameState{Board: starterBoard(front, rear)}

	northSheet := `
Front | Move | G5
Rear  | Move | F2
`
	northOrders, err := orders.Parse(northSheet)
	if err != nil {
		t.Fatalf("Parse(north): %v", err)
	}

	tieBreak := &TieBreak{Holder: "north"}
	state, _, err = ExecuteTurn(state, core, scenario, tieBreak, northOrders, nil)
	if err != nil {
		t.Fatalf("ExecuteTurn: %v", err)
	}

	gotFront, ok := state.Board.Unit("Front")
	if !ok || gotFront.Pos != mustParse(t, "G5") {
		t.Errorf("Front = %+v, %v, want G5 (moved out of Rear's way)", gotFront, ok)
	}
	gotRear, ok := state.Board.Unit("Rear")
	if !ok || gotRear.Pos != mustParse(t, "F2") {
		t.Errorf("Rear = %+v, %v, want F2 (straight through F5, vacated by Front earlier in the same turn)", gotRear, ok)
	}
}

// TestExecuteTurnMultiOrderScenario runs a full turn mixing Deploy, Move,
// Fire and Ready orders across both sides, checking that initiative,
// alternating execution and order skipping (for a unit destroyed earlier
// in the same turn) all work together (docs/core-rules.md sections 5-9).
func TestExecuteTurnMultiOrderScenario(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestScenario(t, templates)

	northGun := newUnit(t, templates, "1st Guns", "north", "artillery", "F2", hex.S)
	northGun.State = "ready"
	northFoot := newUnit(t, templates, "1st Foot", "north", "infantry", "A1", hex.S)
	southFoot := newUnit(t, templates, "1st Levy", "south", "infantry", "F6", hex.N) // in 1st Guns' arc and range.
	southReserve := reserveUnit(t, templates, "2nd Levy", "south", "infantry")

	state := GameState{
		Board:    starterBoard(northGun, northFoot, southFoot),
		Reserves: map[string][]UnitInstance{"south": {southReserve}},
	}

	northSheet := `
1st Guns | Fire | 1st Levy
1st Foot | Move | A5
`
	southSheet := `
1st Levy  | Move     | L9
2nd Levy  | Deploy   | L10
`
	northOrders, err := orders.Parse(northSheet)
	if err != nil {
		t.Fatalf("Parse(north): %v", err)
	}
	southOrders, err := orders.Parse(southSheet)
	if err != nil {
		t.Fatalf("Parse(south): %v", err)
	}

	tieBreak := &TieBreak{Holder: "north"}
	state, events, err := ExecuteTurn(state, core, scenario, tieBreak, northOrders, southOrders)
	if err != nil {
		t.Fatalf("ExecuteTurn: %v", err)
	}

	// North's Move total is 4 (1st Foot); South's is 4 too (1st Levy) -
	// Deploy doesn't count. That's a tie: north holds the token at the
	// start, so north goes first and the token passes to south.
	if tieBreak.Holder != "south" {
		t.Errorf("TieBreak.Holder after a tied initiative = %q, want south", tieBreak.Holder)
	}

	// 1st Foot should have moved to A5.
	foot, ok := state.Board.Unit("1st Foot")
	if !ok || foot.Pos != mustParse(t, "A5") {
		t.Errorf("1st Foot = %+v, %v, want at A5", foot, ok)
	}

	// 1st Levy (south) was fired on by 1st Guns (EX-6: a hit) before its
	// own Move order came up later in the sequence, so it should have
	// moved from its half-strength position, not been skipped.
	levy, levyOnBoard := state.Board.Unit("1st Levy")
	if levyOnBoard {
		if levy.Strength != Half {
			t.Errorf("1st Levy.Strength = %v, want Half (hit by 1st Guns)", levy.Strength)
		}
		if levy.Pos == mustParse(t, "F6") {
			t.Errorf("1st Levy didn't move from F6")
		}
	}

	// 2nd Levy should have deployed into south's zone and left the reserves.
	deployed, ok := state.Board.Unit("2nd Levy")
	if !ok || deployed.Pos != mustParse(t, "L10") {
		t.Errorf("2nd Levy = %+v, %v, want deployed at L10", deployed, ok)
	}
	if len(state.Reserves["south"]) != 0 {
		t.Errorf("Reserves[south] = %+v, want empty", state.Reserves["south"])
	}

	foundRanged := false
	for _, e := range events {
		if e.Kind == "ranged" {
			foundRanged = true
		}
	}
	if !foundRanged {
		t.Errorf("events = %+v, want a ranged event from 1st Guns firing", events)
	}
}

// TestExecuteTurnSkipsDestroyedUnitsOrder checks docs/core-rules.md
// section 5.3: "If a unit is destroyed before its order comes up, that
// order is skipped (it still uses its place in the sequence)."
func TestExecuteTurnSkipsDestroyedUnitsOrder(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestScenario(t, templates)

	// Half-strength infantry, destroyed by a single hit from the cavalry's
	// rear attack (EX-4's numbers).
	attacker := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	victim := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	victim.Strength = Half
	bystander := newUnit(t, templates, "C", "north", "infantry", "A1", hex.S)
	state := GameState{Board: starterBoard(attacker, victim, bystander)}

	northSheet := `
B | Move | A5
C | Move | A4
`
	southSheet := `
A | Close and Attack | B
`
	northOrders, err := orders.Parse(northSheet)
	if err != nil {
		t.Fatalf("Parse(north): %v", err)
	}
	southOrders, err := orders.Parse(southSheet)
	if err != nil {
		t.Fatalf("Parse(south): %v", err)
	}

	// South's total (Close and Attack doesn't count) is 0; north's is 8.
	// South goes first, destroying B before B's own Move order comes up.
	tieBreak := &TieBreak{Holder: "north"}
	state, events, err := ExecuteTurn(state, core, scenario, tieBreak, northOrders, southOrders)
	if err != nil {
		t.Fatalf("ExecuteTurn: %v", err)
	}

	if _, ok := state.Board.Unit("B"); ok {
		t.Fatalf("B should have been destroyed by A's attack")
	}
	foundSkip := false
	for _, e := range events {
		if e.Kind == "order-skipped" && e.Unit == "B" {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("events = %+v, want an order-skipped event for B", events)
	}
	// C's order should still have executed normally.
	bystanderAfter, ok := state.Board.Unit("C")
	if !ok || bystanderAfter.Pos != mustParse(t, "A4") {
		t.Errorf("C = %+v, %v, want moved to A4", bystanderAfter, ok)
	}
}
