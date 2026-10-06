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
	wantHit := HitCheck{AttackerTotal: 2, DefenderTotal: 3, AttackerWins: false}
	if result.Hit != wantHit {
		t.Errorf("Hit = %+v, want %+v", result.Hit, wantHit)
	}
	wantDamage := DamageCheck{Damage: 2, Def: 2, Hit: false}
	if result.Damage != wantDamage {
		t.Errorf("Damage = %+v, want %+v", result.Damage, wantDamage)
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
	wantHit := HitCheck{AttackerTotal: 3, DefenderTotal: 2, AttackerWins: true}
	if result.Hit != wantHit {
		t.Errorf("Hit = %+v, want %+v", result.Hit, wantHit)
	}
	wantDamage := DamageCheck{Damage: 3, Def: 2, Hit: true}
	if result.Damage != wantDamage {
		t.Errorf("Damage = %+v, want %+v", result.Damage, wantDamage)
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
// at F7 (directly S of B) attacks B's rear.
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
	wantHit := HitCheck{AttackerTotal: 6, DefenderTotal: 2, AttackerWins: true}
	if result.Hit != wantHit {
		t.Errorf("Hit = %+v, want %+v", result.Hit, wantHit)
	}
	wantDamage := DamageCheck{Damage: 6, Def: 2, Hit: true}
	if result.Damage != wantDamage {
		t.Errorf("Damage = %+v, want %+v", result.Damage, wantDamage)
	}
	wantTo, _ := hex.ParseOffset("F5")
	if result.Knockback == nil || result.Knockback.To != wantTo || result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want {To:F5 Destroyed:false}", result.Knockback)
	}
}

// TestMeleeEX4DestroyedIfRetreatBlocked extends EX-4: "If F5 were occupied
// or adjacent to another enemy, B would be destroyed instead."
func TestMeleeEX4DestroyedIfRetreatBlocked(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	blocker := newUnit(t, templates, "Blocker", "south", "infantry", "F5", hex.N)
	board := starterBoard(a, b, blocker)

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
// section 3.2: a half-strength unit that takes a hit is destroyed, not
// knocked back.
func TestMeleeHalfStrengthLoserDestroyedByHit(t *testing.T) {
	templates, core := loadTestRules(t)
	a := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	b := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	b.Strength = Half
	board := starterBoard(a, b)

	result, err := ResolveMelee(board, core, "A", "B", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	if !result.Damage.Hit {
		t.Fatalf("Damage.Hit = false, want true")
	}
	if !result.LoserDestroyed {
		t.Errorf("LoserDestroyed = false, want true")
	}
	if result.Knockback != nil {
		t.Errorf("Knockback = %+v, want nil", result.Knockback)
	}
}

// TestMeleeEX5Ambush checks EX-5: Cavalry A, ambushed, is attacked across
// its front by Infantry B. Even ambushed, cavalry beats infantry head-on.
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
	wantHit := HitCheck{AttackerTotal: 2, DefenderTotal: 2, AttackerWins: false}
	if result.Hit != wantHit {
		t.Errorf("Hit = %+v, want %+v (tie goes to the defender)", result.Hit, wantHit)
	}
	wantDamage := DamageCheck{Damage: 3, Def: 2, Hit: true}
	if result.Damage != wantDamage {
		t.Errorf("Damage = %+v, want %+v", result.Damage, wantDamage)
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
