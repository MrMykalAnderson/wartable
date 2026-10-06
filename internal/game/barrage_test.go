package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// TestBarrageEX7 checks EX-7 from docs/core-rules.md section 11: Ready
// artillery's passive Barrage fires at every enemy in its range band and
// arc; a target inside the minimum range is safe, and support from that
// safe unit still applies to the shot against its ally.
func TestBarrageEX7(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F2", hex.S)
	gun.State = "ready"
	foot := newUnit(t, templates, "3rd Foot", "south", "infantry", "F5", hex.N)
	horse := newUnit(t, templates, "4th Horse", "south", "cavalry", "F4", hex.N)
	board := starterBoard(gun, foot, horse)

	newBoard, shots := ResolveBarragePhase(board, core)

	if len(shots) != 1 {
		t.Fatalf("ResolveBarragePhase: got %d shots, want 1 (4th Horse is inside the minimum range)", len(shots))
	}
	shot := shots[0].Result
	if shot.TargetID != "3rd Foot" {
		t.Errorf("shot target = %q, want 3rd Foot", shot.TargetID)
	}
	if shot.ShooterTotal != 3 || shot.TargetTotal != 3 || shot.Margin != 0 {
		t.Errorf("ShooterTotal/TargetTotal/Margin = %d/%d/%d, want 3/3/0 (3rd Foot's adjacent ally 4th Horse gives it support)", shot.ShooterTotal, shot.TargetTotal, shot.Margin)
	}
	if shot.Hits != 0 {
		t.Errorf("Hits = %d, want 0 (miss)", shot.Hits)
	}

	foot2, ok := newBoard.Unit("3rd Foot")
	if !ok || foot2.Strength != Full {
		t.Errorf("3rd Foot = %+v, %v, want Full strength (the hit check missed)", foot2, ok)
	}
	if _, ok := newBoard.Unit("4th Horse"); !ok {
		t.Errorf("4th Horse missing from board: it should never have been fired on")
	}
}

// TestBarrageEX7WithoutSupport checks EX-7's closing note: without 4th
// Horse's support, the same shot would hit and drop 3rd Foot to half
// strength.
func TestBarrageEX7WithoutSupport(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F2", hex.S)
	gun.State = "ready"
	foot := newUnit(t, templates, "3rd Foot", "south", "infantry", "F5", hex.N)
	board := starterBoard(gun, foot)

	newBoard, shots := ResolveBarragePhase(board, core)
	if len(shots) != 1 {
		t.Fatalf("ResolveBarragePhase: got %d shots, want 1", len(shots))
	}
	if shots[0].Result.ShooterTotal != 3 || shots[0].Result.TargetTotal != 2 || shots[0].Result.Margin != 1 {
		t.Errorf("ShooterTotal/TargetTotal/Margin = %d/%d/%d, want 3/2/1", shots[0].Result.ShooterTotal, shots[0].Result.TargetTotal, shots[0].Result.Margin)
	}
	if shots[0].Result.Hits != 1 || shots[0].Result.Destroyed {
		t.Errorf("Hits/Destroyed = %d/%v, want 1/false (a hit)", shots[0].Result.Hits, shots[0].Result.Destroyed)
	}
	foot2, ok := newBoard.Unit("3rd Foot")
	if !ok || foot2.Strength != Half {
		t.Errorf("3rd Foot = %+v, %v, want Half strength", foot2, ok)
	}
}

// TestBarrageStacksMultipleHits checks docs/core-rules.md section 5.3: "A
// unit hit by more than one gun takes a hit from each, so a full-strength
// unit hit twice is destroyed." Both guns sit on the same hex column as
// the target so it's dead ahead (inside the firing arc) of both.
func TestBarrageStacksMultipleHits(t *testing.T) {
	templates, core := loadTestRules(t)
	gunA := newUnit(t, templates, "Gun A", "north", "artillery", "F1", hex.S)
	gunA.State = "ready"
	gunB := newUnit(t, templates, "Gun B", "north", "artillery", "F2", hex.S)
	gunB.State = "ready"
	foot := newUnit(t, templates, "Foot", "south", "infantry", "F5", hex.N)
	board := starterBoard(gunA, gunB, foot)

	newBoard, shots := ResolveBarragePhase(board, core)
	if len(shots) != 2 {
		t.Fatalf("ResolveBarragePhase: got %d shots, want 2 (both guns dead ahead of Foot)", len(shots))
	}
	for _, s := range shots {
		if s.Result.TargetID != "Foot" {
			t.Fatalf("shot target = %q, want Foot", s.Result.TargetID)
		}
	}
	if _, ok := newBoard.Unit("Foot"); ok {
		t.Errorf("Foot should have been destroyed: hit by two guns in the same barrage")
	}
}
