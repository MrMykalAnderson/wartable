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
	// A front attack (margin 1, once ambushed) only hits and knocks back,
	// rather than destroying outright at margin 3+ (docs/core-rules.md
	// section 8.3), so this exercises the blocked-knockback path.
	attacker := newUnit(t, templates, "A", "south", "infantry", "F5", hex.S)

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

	board, events := resolveAmbush(board, core, "X")

	if len(events) != 1 {
		t.Fatalf("resolveAmbush: got %d combat(s), want exactly 1 (losing always ends the ambush)", len(events))
	}
	result := events[0].Melee
	if result.Knockback == nil || !result.Knockback.Destroyed {
		t.Errorf("Knockback = %+v, want a blocked (Destroyed) knockback to %s", result.Knockback, knockbackHex)
	}
	if _, ok := board.Unit("X"); ok {
		t.Errorf("X should have been destroyed rather than knocked back next to C")
	}
	if _, ok := board.Unit("C"); !ok {
		t.Errorf("C should be untouched: an ambush never draws in an enemy that wasn't adjacent at the start")
	}
}

// TestAmbushSkipsNonMeleeEnemies checks the docs/dev-plan.md section 7.7
// bug fix: artillery can't make melee attacks, so it never attacks in
// an ambush, even when it's the only enemy adjacent to the moving unit.
func TestAmbushSkipsNonMeleeEnemies(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "Gun", "north", "artillery", "F3", hex.S)
	gun.State = "mobilised" // not dug in, so any knockback/destroy below is unambiguous.
	mover := newUnit(t, templates, "Rider", "south", "cavalry", "F4", hex.N)
	board := starterBoard(gun, mover)

	board, events := resolveAmbush(board, core, "Rider")
	if len(events) != 0 {
		t.Errorf("resolveAmbush events = %+v, want none (artillery never attacks in an ambush)", events)
	}
	if _, ok := board.Unit("Rider"); !ok {
		t.Errorf("Rider missing, want it untouched by the (non-existent) ambush")
	}
}

// TestOverrunEX9Destroyed checks EX-9's first case: 3rd Foot overruns
// Ready (dug-in) artillery across its rear and destroys it outright at
// margin 1, well below the margin 3+ that would destroy a non-dug-in
// unit.
func TestOverrunEX9Destroyed(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F3", hex.S)
	gun.State = "ready"
	foot := newUnit(t, templates, "3rd Foot", "south", "infantry", "F2", hex.N)
	board := starterBoard(gun, foot)

	board, events := applyAmbush(board, core, "3rd Foot")

	var overrun *MeleeResult
	for _, e := range events {
		if e.Kind == "melee" && e.Detail == "overrun" {
			overrun = e.Melee
		}
	}
	if overrun == nil {
		t.Fatalf("events = %+v, want an overrun melee event", events)
	}
	if overrun.AttackerID != "3rd Foot" || overrun.DefenderID != "1st Guns" || overrun.Edge != hex.Rear {
		t.Errorf("overrun = %+v, want 3rd Foot attacking 1st Guns' rear", overrun)
	}
	if overrun.Ambushed {
		t.Errorf("overrun.Ambushed = true, want false (no ambush penalty)")
	}
	if overrun.Margin != 1 {
		t.Errorf("overrun.Margin = %d, want 1", overrun.Margin)
	}
	if !overrun.LoserDestroyed {
		t.Errorf("overrun.LoserDestroyed = false, want true (Ready artillery is dug in: any margin of loss destroys it)")
	}
	if _, ok := board.Unit("1st Guns"); ok {
		t.Errorf("1st Guns still on board, want destroyed")
	}
	if _, ok := board.Unit("3rd Foot"); !ok {
		t.Errorf("3rd Foot missing, want it to survive (it won)")
	}
}

// TestOverrunEX9Repelled checks EX-9's second case: 3rd Foot attacks
// the gun's front and is repelled, knocked back one hex further away.
func TestOverrunEX9Repelled(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F3", hex.S)
	gun.State = "ready"
	foot := newUnit(t, templates, "3rd Foot", "south", "infantry", "F4", hex.N)
	board := starterBoard(gun, foot)

	board, events := applyAmbush(board, core, "3rd Foot")

	var overrun *MeleeResult
	for _, e := range events {
		if e.Kind == "melee" && e.Detail == "overrun" {
			overrun = e.Melee
		}
	}
	if overrun == nil {
		t.Fatalf("events = %+v, want an overrun melee event", events)
	}
	if overrun.Edge != hex.Front || overrun.Margin != -2 || overrun.Hits != 0 {
		t.Errorf("overrun = %+v, want front attack at margin -2, repelled", overrun)
	}
	foot2, ok := board.Unit("3rd Foot")
	if !ok || foot2.Pos != mustParse(t, "F5") {
		t.Errorf("3rd Foot = %+v, %v, want knocked back to F5", foot2, ok)
	}
	if _, ok := board.Unit("1st Guns"); !ok {
		t.Errorf("1st Guns missing, want it to survive (it won)")
	}
}
