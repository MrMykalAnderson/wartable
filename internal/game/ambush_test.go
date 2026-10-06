package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// TestAmbushEndsWhenKnockbackWouldBeAdjacentToAnotherEnemy checks the
// docs/core-rules.md section 7.4 note: if the ambushed unit loses a
// combat, its knockback can never put it next to another enemy (it is
// destroyed instead, per section 8.3's CanRetreatTo check), so an ambush
// never draws in an enemy that wasn't adjacent at the start.
func TestAmbushEndsWhenKnockbackWouldBeAdjacentToAnotherEnemy(t *testing.T) {
	templates, core := loadTestRules(t)
	ambushed := newUnit(t, templates, "X", "north", "infantry", "F6", hex.N)
	attacker := newUnit(t, templates, "A", "south", "cavalry", "G6", hex.N)

	// Find a hex adjacent to X's knockback destination (if it loses) that
	// isn't already adjacent to X's original hex: a genuine "third
	// enemy" that could only ever join by knockback, not from the start.
	knockbackHex := hex.Knockback(attacker.Pos, ambushed.Pos)
	var thirdPos hex.Offset
	found := false
	for _, d := range hex.Directions {
		candidate := knockbackHex.Neighbor(d)
		if candidate == ambushed.Pos || candidate == attacker.Pos {
			continue
		}
		if _, ok := hex.AdjacentDirection(ambushed.Pos, candidate); ok {
			continue
		}
		thirdPos, found = candidate, true
		break
	}
	if !found {
		t.Fatalf("test setup: no hex adjacent to knockback destination %s avoids X's original neighbours", knockbackHex)
	}

	third := UnitInstance{ID: "C", Side: "south", Template: templates["infantry"], Pos: thirdPos, Facing: hex.N, Strength: Full}
	board := starterBoard(ambushed, attacker, third)

	board, results := resolveAmbush(board, core, "X")

	if len(results) != 1 {
		t.Fatalf("resolveAmbush: got %d combat(s), want exactly 1 (losing always ends the ambush)", len(results))
	}
	if results[0].Knockback == nil || !results[0].Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want a blocked (Destroyed) knockback to %s", results[0].Knockback, knockbackHex)
	}
	if _, ok := board.Unit("X"); ok {
		t.Errorf("X should have been destroyed rather than knocked back next to C")
	}
	if _, ok := board.Unit("C"); !ok {
		t.Errorf("C should be untouched: an ambush never draws in an enemy that wasn't adjacent at the start")
	}
}
