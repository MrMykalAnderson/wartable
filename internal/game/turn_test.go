package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// TestInitiativeAndExecutionOrderEX1 checks EX-1 from docs/core-rules.md
// section 11: North writes Move 1st Horse (Move 7), Move 2nd Foot (Move
// 4), Fire 1st Guns; South writes Move 3rd Foot (Move 4), Close and Attack
// [a unit] vs 4th Horse, Move 5th Foot (Move 4). North's total is 11,
// South's is 8 (Close and Attack doesn't count), so South has initiative
// and goes: South1, North1, South2, North2, South3, North3.
func TestInitiativeAndExecutionOrderEX1(t *testing.T) {
	templates, core := loadTestRules(t)

	northSheet := `
1st Horse | Move | A5
2nd Foot  | Move | A6
1st Guns  | Fire | Enemy Guns
`
	southSheet := `
3rd Foot | Move | L5
1st Foot | Close and Attack | 4th Horse
5th Foot | Move | L6
`
	northOrders, err := orders.Parse(northSheet)
	if err != nil {
		t.Fatalf("Parse(north): %v", err)
	}
	southOrders, err := orders.Parse(southSheet)
	if err != nil {
		t.Fatalf("Parse(south): %v", err)
	}

	board := starterBoard(
		newUnit(t, templates, "1st Horse", "north", "cavalry", "A1", hex.S),
		newUnit(t, templates, "2nd Foot", "north", "infantry", "A2", hex.S),
		newUnit(t, templates, "3rd Foot", "south", "infantry", "L9", hex.N),
		newUnit(t, templates, "5th Foot", "south", "infantry", "L10", hex.N),
	)

	northTotal, err := Initiative(board, core, northOrders)
	if err != nil {
		t.Fatalf("Initiative(north): %v", err)
	}
	if northTotal != 11 {
		t.Errorf("northTotal = %d, want 11", northTotal)
	}
	southTotal, err := Initiative(board, core, southOrders)
	if err != nil {
		t.Fatalf("Initiative(south): %v", err)
	}
	if southTotal != 8 {
		t.Errorf("southTotal = %d, want 8", southTotal)
	}

	tb := &TieBreak{Holder: "north"}
	first := tb.Resolve("north", northTotal, "south", southTotal)
	if first != "south" {
		t.Fatalf("initiative winner = %q, want south (lower total)", first)
	}
	if tb.Holder != "north" {
		t.Errorf("TieBreak.Holder = %q, want unchanged (north): no tie occurred", tb.Holder)
	}

	steps := BuildExecutionOrder("south", southOrders, "north", northOrders)
	wantSides := []string{"south", "north", "south", "north", "south", "north"}
	if len(steps) != len(wantSides) {
		t.Fatalf("len(steps) = %d, want %d", len(steps), len(wantSides))
	}
	for i, want := range wantSides {
		if steps[i].Side != want {
			t.Errorf("steps[%d].Side = %q, want %q", i, steps[i].Side, want)
		}
	}
}

func TestTieBreakAlternates(t *testing.T) {
	tb := &TieBreak{Holder: "north"}
	if got := tb.Resolve("north", 5, "south", 5); got != "north" {
		t.Errorf("first tie winner = %q, want north (holder)", got)
	}
	if tb.Holder != "south" {
		t.Fatalf("TieBreak.Holder after first tie = %q, want south", tb.Holder)
	}
	if got := tb.Resolve("north", 5, "south", 5); got != "south" {
		t.Errorf("second tie winner = %q, want south (token passed)", got)
	}
	if tb.Holder != "north" {
		t.Errorf("TieBreak.Holder after second tie = %q, want north", tb.Holder)
	}
	// A non-tied initiative doesn't touch the token.
	if got := tb.Resolve("north", 3, "south", 9); got != "north" {
		t.Errorf("non-tied winner = %q, want north (lower total)", got)
	}
	if tb.Holder != "north" {
		t.Errorf("TieBreak.Holder after non-tied resolve = %q, want unchanged (north)", tb.Holder)
	}
}

func TestBuildExecutionOrderUnequalLengths(t *testing.T) {
	first := []orders.Order{{Unit: "A"}, {Unit: "B"}}
	second := []orders.Order{{Unit: "X"}}
	steps := BuildExecutionOrder("first", first, "second", second)
	want := []string{"first:A", "second:X", "first:B"}
	if len(steps) != len(want) {
		t.Fatalf("len(steps) = %d, want %d", len(steps), len(want))
	}
	for i, w := range want {
		got := steps[i].Side + ":" + steps[i].Order.Unit
		if got != w {
			t.Errorf("steps[%d] = %q, want %q", i, got, w)
		}
	}
}
