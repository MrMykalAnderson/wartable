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
	wantHit := RangedHitCheck{ShooterTotal: 3, TargetTotal: 2, Hit: true}
	if result.HitCheck != wantHit {
		t.Errorf("HitCheck = %+v, want %+v", result.HitCheck, wantHit)
	}
	wantDamage := RangedDamageCheck{RngDmg: 3, Def: 2, Hit: true}
	if result.Damage != wantDamage {
		t.Errorf("Damage = %+v, want %+v", result.Damage, wantDamage)
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
	wantHit := RangedHitCheck{ShooterTotal: 3, TargetTotal: 3, Hit: false}
	if result.HitCheck != wantHit {
		t.Errorf("HitCheck = %+v, want %+v", result.HitCheck, wantHit)
	}
	if result.Damage.Hit {
		t.Errorf("Damage.Hit = true, want false (no hit check success means no damage check)")
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
	if result.HitCheck != (RangedHitCheck{}) {
		t.Errorf("HitCheck = %+v, want zero value (attack not attempted)", result.HitCheck)
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
