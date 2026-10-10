package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// TestSafeRouteEX8 checks EX-8 from docs/core-rules.md section 11:
// South's 2nd Foot (infantry, Move 4) is at F9 with a Move order to F4;
// North's 1st Foot is at G6. The plain shortest route (F8, F7, F6, F5,
// F4) passes F6 and F5, both next to 1st Foot; the safe route (F8, F7,
// E7, E6, E5, F4) is one hex longer but never goes near it. With Move
// 4, 2nd Foot ends the turn at E6.
func TestSafeRouteEX8(t *testing.T) {
	templates, core := loadTestRules(t)
	mover := newUnit(t, templates, "2nd Foot", "south", "infantry", "F9", hex.N)
	enemy := newUnit(t, templates, "1st Foot", "north", "infantry", "G6", hex.S)
	board := starterBoard(mover, enemy)

	wantSafePath := []hex.Offset{
		mustParse(t, "F8"), mustParse(t, "F7"), mustParse(t, "E7"),
		mustParse(t, "E6"), mustParse(t, "E5"), mustParse(t, "F4"),
	}
	gotPath := safeShortestPath(board, mover, mustParse(t, "F4"), "")
	if len(gotPath) != len(wantSafePath) {
		t.Fatalf("safeShortestPath = %v, want %v", gotPath, wantSafePath)
	}
	for i, h := range gotPath {
		if h != wantSafePath[i] {
			t.Errorf("safeShortestPath[%d] = %s, want %s (full route %v)", i, h, wantSafePath[i], gotPath)
		}
	}

	state := GameState{Board: board}
	o := orders.Order{Unit: "2nd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "F4")}
	state, events := executeMove(state, core, mover, o)

	moved, ok := state.Board.Unit("2nd Foot")
	if !ok {
		t.Fatalf("2nd Foot missing after move")
	}
	if moved.Pos != mustParse(t, "E6") {
		t.Errorf("2nd Foot.Pos = %s, want E6 (Move 4 along the safe route)", moved.Pos)
	}
	for _, e := range events {
		if e.Kind == "melee" || e.Kind == "unit-destroyed" {
			t.Errorf("events = %+v, want no combat (the safe route never goes near 1st Foot)", events)
		}
	}
}

// TestSafeRouteFallsBackWhenNoneExists checks docs/core-rules.md section
// 7.2: if no route avoids every enemy's adjacency, the unit falls back
// to the plain shortest path and risks contact on the way, rather than
// refusing to move.
func TestSafeRouteFallsBackWhenNoneExists(t *testing.T) {
	templates, _ := loadTestRules(t)
	// A corridor along column A: with the map edge on one side and a
	// line of enemies down column B, every route up column A passes
	// next to one of them. A1 is the only reachable hex far enough
	// north to not be adjacent to a column-B enemy itself.
	mover := newUnit(t, templates, "Scout", "south", "infantry", "A10", hex.N)
	var board Board = starterBoard(mover,
		newUnit(t, templates, "B9", "north", "infantry", "B9", hex.N),
		newUnit(t, templates, "B8", "north", "infantry", "B8", hex.N),
		newUnit(t, templates, "B7", "north", "infantry", "B7", hex.N),
		newUnit(t, templates, "B6", "north", "infantry", "B6", hex.N),
		newUnit(t, templates, "B5", "north", "infantry", "B5", hex.N),
		newUnit(t, templates, "B4", "north", "infantry", "B4", hex.N),
		newUnit(t, templates, "B3", "north", "infantry", "B3", hex.N),
	)

	path := safeShortestPath(board, mover, mustParse(t, "A2"), "")
	if len(path) == 0 || path[len(path)-1] != mustParse(t, "A2") {
		t.Errorf("safeShortestPath = %v, want it to still reach A2 (falling back to the plain shortest path)", path)
	}
}

// TestSafeRouteExemptsCloseOrderTarget checks docs/core-rules.md
// section 7.2: adjacency to a Close order's own named target doesn't
// make a hex unsafe, since the unit is trying to get there.
func TestSafeRouteExemptsCloseOrderTarget(t *testing.T) {
	templates, _ := loadTestRules(t)
	mover := newUnit(t, templates, "Rider", "south", "cavalry", "F9", hex.N)
	target := newUnit(t, templates, "Target", "north", "infantry", "G6", hex.S)
	board := starterBoard(mover, target)

	// Without the exemption, every hex adjacent to Target (including
	// F6, F7) would be unsafe, forcing a detour even though Target is
	// exactly what this unit is closing on.
	withExemption := safeShortestPath(board, mover, mustParse(t, "F6"), "Target")
	plain, _ := hex.ShortestPath(mover.Pos, mustParse(t, "F6"), PassableFor(board, mover.ID), nil)
	if len(withExemption) != len(plain) {
		t.Errorf("safeShortestPath (with Target exempt) = %v, want the plain shortest path %v", withExemption, plain)
	}
}
