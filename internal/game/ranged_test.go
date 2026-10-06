package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// TestRangedEX6ArtilleryFire checks EX-6 from docs/core-rules.md section
// 11: Ready Artillery A fires at Infantry B, 4 hexes away and inside A's
// arc.
func TestRangedEX6ArtilleryFire(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "artillery", "F6", hex.N)
	a.State = "ready"
	b := newUnit(t, templates, "B", "south", "infantry", "F2", hex.S)
	board := starterBoard(a, b)

	result, err := ResolveRanged(board, core, "A", "B")
	if err != nil {
		t.Fatalf("ResolveRanged: %v", err)
	}
	if !result.InRange || !result.InArc {
		t.Fatalf("InRange, InArc = %v, %v, want true, true", result.InRange, result.InArc)
	}
	if result.ShooterTotal != 3 || result.TargetTotal != 2 || result.Margin != 1 {
		t.Errorf("ShooterTotal/TargetTotal/Margin = %d/%d/%d, want 3/2/1", result.ShooterTotal, result.TargetTotal, result.Margin)
	}
	if result.Hits != 1 || result.Destroyed {
		t.Errorf("Hits/Destroyed = %d/%v, want 1/false (a hit)", result.Hits, result.Destroyed)
	}
}

// TestRangedEX6WithAllySupportMisses checks EX-6's closing note: "If B had
// one adjacent ally, the hit check would be 3 vs 3: a miss."
func TestRangedEX6WithAllySupportMisses(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "artillery", "F6", hex.N)
	a.State = "ready"
	b := newUnit(t, templates, "B", "south", "infantry", "F2", hex.S)
	ally := newUnit(t, templates, "Ally", "south", "infantry", "E2", hex.S)
	board := starterBoard(a, b, ally)

	result, err := ResolveRanged(board, core, "A", "B")
	if err != nil {
		t.Fatalf("ResolveRanged: %v", err)
	}
	if result.ShooterTotal != 3 || result.TargetTotal != 3 || result.Margin != 0 {
		t.Errorf("ShooterTotal/TargetTotal/Margin = %d/%d/%d, want 3/3/0", result.ShooterTotal, result.TargetTotal, result.Margin)
	}
	if result.Hits != 0 {
		t.Errorf("Hits = %d, want 0 (miss)", result.Hits)
	}
}

func TestRangedOutOfRange(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "infantry", "A1", hex.S)
	b := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	board := starterBoard(a, b)

	result, err := ResolveRanged(board, core, "A", "B")
	if err != nil {
		t.Fatalf("ResolveRanged: %v", err)
	}
	if result.InRange {
		t.Errorf("InRange = true, want false")
	}
	if result.Margin != 0 || result.Hits != 0 {
		t.Errorf("Margin/Hits = %d/%d, want 0/0 (attack not attempted)", result.Margin, result.Hits)
	}
}

func TestRangedOutsideArc(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "artillery", "F6", hex.N)
	a.State = "ready"
	b := newUnit(t, templates, "B", "south", "infantry", "F7", hex.N) // directly behind A.
	board := starterBoard(a, b)

	result, err := ResolveRanged(board, core, "A", "B")
	if err != nil {
		t.Fatalf("ResolveRanged: %v", err)
	}
	if result.InArc {
		t.Errorf("InArc = true, want false")
	}
}
