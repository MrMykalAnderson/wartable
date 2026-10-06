package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// TestMeleeEX2FrontAttackRebuffed checks EX-2 from docs/core-rules.md
// section 11: Infantry A attacks Infantry B across B's front; B has one
// friendly unit adjacent.
func TestMeleeEX2FrontAttackRebuffed(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "infantry", "F5", hex.S)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	c := newUnit(t, templates, "C", "north", "infantry", "F7", hex.N) // B's ally.
	board := starterBoard(a, b, c)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}

	if result.Edge != hex.Front {
		t.Errorf("Edge = %v, want Front", result.Edge)
	}
	if result.AttackerTotal != 2 || result.DefenderTotal != 3 || result.Margin != -1 {
		t.Errorf("AttackerTotal/DefenderTotal/Margin = %d/%d/%d, want 2/3/-1", result.AttackerTotal, result.DefenderTotal, result.Margin)
	}
	if result.Hits != 0 {
		t.Errorf("Hits = %d, want 0 (repelled)", result.Hits)
	}
	if result.WinnerID != "B" || result.LoserID != "A" {
		t.Errorf("Winner/Loser = %s/%s, want B/A", result.WinnerID, result.LoserID)
	}
	if result.LoserDestroyed {
		t.Errorf("LoserDestroyed = true, want false")
	}
	wantTo, _ := hex.ParseOffset("F4")
	if result.Knockback == nil || result.Knockback.To != wantTo || result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:F4 Destroyed:false}", result.Knockback)
	}
}

// TestMeleeEX3FlankAttack checks EX-3: Infantry B at F6 facing N; Infantry A
// at G6 (B's NE neighbour) attacks B's flank.
func TestMeleeEX3FlankAttack(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "infantry", "G6", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	board := starterBoard(a, b)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}

	if result.Edge != hex.Flank {
		t.Errorf("Edge = %v, want Flank", result.Edge)
	}
	if result.AttackerTotal != 3 || result.DefenderTotal != 2 || result.Margin != 1 {
		t.Errorf("AttackerTotal/DefenderTotal/Margin = %d/%d/%d, want 3/2/1", result.AttackerTotal, result.DefenderTotal, result.Margin)
	}
	if result.Hits != 1 {
		t.Errorf("Hits = %d, want 1", result.Hits)
	}
	if result.WinnerID != "A" || result.LoserID != "B" {
		t.Errorf("Winner/Loser = %s/%s, want A/B", result.WinnerID, result.LoserID)
	}
	wantTo, _ := hex.ParseOffset("E7")
	if result.Knockback == nil || result.Knockback.To != wantTo || result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:E7 Destroyed:false}", result.Knockback)
	}
}

// TestMeleeEX4RearAttack checks EX-4: Infantry B at F6 facing N; Cavalry A
// at F7 (directly S of B) attacks B's rear. Margin 4 destroys B outright,
// at full strength, without a knockback being attempted.
func TestMeleeEX4RearAttack(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	board := starterBoard(a, b)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}

	if result.Edge != hex.Rear {
		t.Errorf("Edge = %v, want Rear", result.Edge)
	}
	if result.AttackerTotal != 6 || result.DefenderTotal != 2 || result.Margin != 4 {
		t.Errorf("AttackerTotal/DefenderTotal/Margin = %d/%d/%d, want 6/2/4", result.AttackerTotal, result.DefenderTotal, result.Margin)
	}
	if result.Hits != 2 {
		t.Errorf("Hits = %d, want 2", result.Hits)
	}
	if !result.LoserDestroyed {
		t.Errorf("LoserDestroyed = false, want true (margin 3+ destroys any unit outright)")
	}
	if result.Knockback != nil {
		t.Errorf("Knockback = %+v, want nil (destroyed outright, no knockback attempted)", result.Knockback)
	}
}

// TestMeleeEX4SupportedHitKnockedBack extends EX-4: "Had B been supported
// by two adjacent allies, it would be 6 vs 4, margin 2: a hit, and B
// would be knocked back N to F5."
func TestMeleeEX4SupportedHitKnockedBack(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	ally1 := newUnit(t, templates, "Ally1", "north", "infantry", "E6", hex.N)
	ally2 := newUnit(t, templates, "Ally2", "north", "infantry", "G6", hex.N)
	board := starterBoard(a, b, ally1, ally2)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	if result.AttackerTotal != 6 || result.DefenderTotal != 4 || result.Margin != 2 {
		t.Errorf("AttackerTotal/DefenderTotal/Margin = %d/%d/%d, want 6/4/2", result.AttackerTotal, result.DefenderTotal, result.Margin)
	}
	if result.Hits != 1 || result.LoserDestroyed {
		t.Errorf("Hits/LoserDestroyed = %d/%v, want 1/false (a hit, not a destroy)", result.Hits, result.LoserDestroyed)
	}
	wantTo, _ := hex.ParseOffset("F5")
	if result.Knockback == nil || result.Knockback.To != wantTo || result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:F5 Destroyed:false}", result.Knockback)
	}
}

// TestMeleeEX4DestroyedIfRetreatBlocked extends the supported-hit scenario
// above: "or destroyed if F5 were occupied or next to another enemy."
func TestMeleeEX4DestroyedIfRetreatBlocked(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	ally1 := newUnit(t, templates, "Ally1", "north", "infantry", "E6", hex.N)
	ally2 := newUnit(t, templates, "Ally2", "north", "infantry", "G6", hex.N)
	blocker := newUnit(t, templates, "Blocker", "south", "infantry", "F5", hex.N)
	board := starterBoard(a, b, ally1, ally2, blocker)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	wantTo, _ := hex.ParseOffset("F5")
	if result.Knockback == nil || result.Knockback.To != wantTo || !result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:F5 Destroyed:true}", result.Knockback)
	}
}

// TestMeleeHalfStrengthLoserDestroyedByHit checks docs/core-rules.md
// section 3.2: a half-strength unit that takes any hit is destroyed, not
// knocked back.
func TestMeleeHalfStrengthLoserDestroyedByHit(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "infantry", "G6", hex.N) // flank attack, margin 1.
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	b.Strength = Half
	board := starterBoard(a, b)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	if result.Hits != 1 {
		t.Fatalf("Hits = %d, want 1", result.Hits)
	}
	if !result.LoserDestroyed {
		t.Errorf("LoserDestroyed = false, want true")
	}
	if result.Knockback != nil {
		t.Errorf("Knockback = %+v, want nil", result.Knockback)
	}
}

// TestMeleeEX5Ambush checks EX-5: Cavalry A, ambushed, is attacked across
// its front by Infantry B. Even ambushed, cavalry holds off infantry
// head-on: margin 0 repels the attacker B.
func TestMeleeEX5Ambush(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "cavalry", "F5", hex.S)
	b := newUnit(t, templates, "B", "south", "infantry", "F6", hex.N)
	board := starterBoard(a, b)

	// B is the attacker (it ambushed A); A is the defender, ambushed.
	result, err := ResolveMelee(board, core, "B", "A", true)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}

	if result.Edge != hex.Front {
		t.Errorf("Edge = %v, want Front", result.Edge)
	}
	if result.AttackerTotal != 2 || result.DefenderTotal != 2 || result.Margin != 0 {
		t.Errorf("AttackerTotal/DefenderTotal/Margin = %d/%d/%d, want 2/2/0 (tie goes to the defender)", result.AttackerTotal, result.DefenderTotal, result.Margin)
	}
	if result.Hits != 0 {
		t.Errorf("Hits = %d, want 0 (repelled)", result.Hits)
	}
	if result.WinnerID != "A" || result.LoserID != "B" {
		t.Errorf("Winner/Loser = %s/%s, want A/B", result.WinnerID, result.LoserID)
	}
	wantTo, _ := hex.ParseOffset("F7")
	if result.Knockback == nil || result.Knockback.To != wantTo || result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:F7 Destroyed:false}", result.Knockback)
	}
}

func TestResolveMeleeUnknownUnit(t *testing.T) {
	_, core := loadTestRules(t)
	if _, err := ResolveMelee(Board{}, core, "A", "B", false); err == nil {
		t.Fatalf("ResolveMelee: want error for unknown units, got none")
	}
}

func TestResolveMeleeNotAdjacent(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "infantry", "A1", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "L10", hex.N)
	board := starterBoard(a, b)
	if _, err := ResolveMelee(board, core, "A", "B", false); err == nil {
		t.Fatalf("ResolveMelee: want error for non-adjacent units, got none")
	}
}
