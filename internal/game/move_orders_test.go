package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

func TestExecuteMoveBasic(t *testing.T) {
	templates, core := loadTestRules(t)
	mover := newUnit(t, templates, "1st Horse", "north", "cavalry", "A1", hex.N)
	state := GameState{Board: starterBoard(mover)}

	o := orders.Order{Unit: "1st Horse", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "A5")}
	state, events := executeMove(state, core, mover, o)

	moved, ok := state.Board.Unit("1st Horse")
	if !ok {
		t.Fatalf("unit missing after move")
	}
	if moved.Pos != mustParse(t, "A5") {
		t.Errorf("Pos = %s, want A5", moved.Pos)
	}
	if moved.Facing != hex.S {
		t.Errorf("Facing = %v, want S (direction of last step)", moved.Facing)
	}
	if len(events) != 1 || events[0].Kind != "moved" {
		t.Errorf("events = %+v, want one 'moved' event", events)
	}
}

func TestExecuteMoveFacingOverride(t *testing.T) {
	templates, core := loadTestRules(t)
	mover := newUnit(t, templates, "1st Horse", "north", "cavalry", "A1", hex.N)
	state := GameState{Board: starterBoard(mover)}

	o := orders.Order{Unit: "1st Horse", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "A5"), HasFacing: true, Facing: hex.NE}
	state, _ = executeMove(state, core, mover, o)

	moved, _ := state.Board.Unit("1st Horse")
	if moved.Facing != hex.NE {
		t.Errorf("Facing = %v, want NE (explicit order facing)", moved.Facing)
	}
}

// TestExecuteMoveAmbush checks docs/core-rules.md section 6.2/7.4: a Move
// order that contacts an enemy stops there and is ambushed.
func TestExecuteMoveAmbush(t *testing.T) {
	templates, core := loadTestRules(t)
	mover := newUnit(t, templates, "Rider", "north", "cavalry", "F8", hex.N)
	enemy := newUnit(t, templates, "Watcher", "south", "infantry", "G6", hex.SW)
	state := GameState{Board: starterBoard(mover, enemy)}

	o := orders.Order{Unit: "Rider", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "F2")}
	state, events := executeMove(state, core, mover, o)

	// Watcher attacks across Rider's flank (NE of F6, Rider facing N) and
	// wins (infantry flank bonus beats ambushed cavalry's -1 Def), so
	// Rider takes a hit and is knocked back to E7 (the same winner/loser
	// geometry as EX-3) rather than staying adjacent.
	moved, ok := state.Board.Unit("Rider")
	if !ok {
		t.Fatalf("Rider missing after move")
	}
	if moved.Strength != Half {
		t.Errorf("Rider.Strength = %v, want Half (took a hit in the ambush)", moved.Strength)
	}
	if moved.Pos != mustParse(t, "E7") {
		t.Errorf("Rider.Pos = %s, want E7 (knocked back from the ambush)", moved.Pos)
	}

	foundMelee := false
	for _, e := range events {
		if e.Kind == "melee" {
			foundMelee = true
			if e.Melee.AttackerID != "Watcher" || e.Melee.DefenderID != "Rider" {
				t.Errorf("melee event = %+v, want Watcher attacking ambushed Rider", e.Melee)
			}
			if !e.Melee.Ambushed {
				t.Errorf("melee event Ambushed = false, want true")
			}
		}
	}
	if !foundMelee {
		t.Errorf("events = %+v, want a melee event from the ambush", events)
	}
}

// TestExecuteMoveAmbushEventBoardsAreIncremental checks that each event's
// Board reflects the board exactly as it stood right after that event,
// not the final state of the whole order (docs/dev-plan.md section 7.3:
// step-by-step map replay).
func TestExecuteMoveAmbushEventBoardsAreIncremental(t *testing.T) {
	templates, core := loadTestRules(t)
	mover := newUnit(t, templates, "Rider", "north", "cavalry", "F8", hex.N)
	enemy := newUnit(t, templates, "Watcher", "south", "infantry", "G6", hex.SW)
	state := GameState{Board: starterBoard(mover, enemy)}

	o := orders.Order{Unit: "Rider", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "F2")}
	_, events := executeMove(state, core, mover, o)

	if len(events) != 2 {
		t.Fatalf("events = %+v, want exactly 2 (moved, then the ambush melee)", events)
	}
	if events[0].Kind != "moved" {
		t.Fatalf("events[0].Kind = %q, want moved", events[0].Kind)
	}
	moveBoardRider, ok := events[0].Board.Unit("Rider")
	if !ok || moveBoardRider.Pos != mustParse(t, "F6") || moveBoardRider.Strength != Full {
		t.Errorf("events[0].Board's Rider = %+v, %v, want full strength at F6 (contact, before the ambush resolves)", moveBoardRider, ok)
	}

	if events[1].Kind != "melee" {
		t.Fatalf("events[1].Kind = %q, want melee", events[1].Kind)
	}
	meleeBoardRider, ok := events[1].Board.Unit("Rider")
	if !ok || meleeBoardRider.Pos != mustParse(t, "E7") || meleeBoardRider.Strength != Half {
		t.Errorf("events[1].Board's Rider = %+v, %v, want half strength at E7 (after the ambush)", meleeBoardRider, ok)
	}
}

func TestExecuteCloseAndAttackAlreadyAdjacent(t *testing.T) {
	templates, core := loadTestRules(t)
	attacker := newUnit(t, templates, "A", "south", "cavalry", "F7", hex.N)
	defender := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	state := GameState{Board: starterBoard(attacker, defender)}

	o := orders.Order{Unit: "A", Type: orders.CloseAndAttack, TargetUnit: "B"}
	state, events := executeCloseAndAttack(state, core, attacker, o)

	if len(events) != 1 || events[0].Kind != "melee" {
		t.Fatalf("events = %+v, want exactly one melee event (no movement needed)", events)
	}
	// EX-4's numbers: rear attack, margin 4 destroys B outright.
	if _, ok := state.Board.Unit("B"); ok {
		t.Errorf("B still on board, want destroyed (margin 3+ destroys any unit outright)")
	}
}

func TestExecuteCloseAndAttackMovesThenAttacks(t *testing.T) {
	templates, core := loadTestRules(t)
	attacker := newUnit(t, templates, "A", "south", "cavalry", "F10", hex.N) // Move 7, half-move 3.
	defender := newUnit(t, templates, "B", "north", "infantry", "F6", hex.N)
	state := GameState{Board: starterBoard(attacker, defender)}

	o := orders.Order{Unit: "A", Type: orders.CloseAndAttack, TargetUnit: "B"}
	state, events := executeCloseAndAttack(state, core, attacker, o)

	kinds := eventKinds(events)
	if kinds[0] != "moved" || kinds[len(kinds)-1] != "melee" {
		t.Fatalf("events kinds = %v, want to start with 'moved' and end with 'melee'", kinds)
	}
	mover, ok := state.Board.Unit("A")
	if !ok {
		t.Fatalf("A missing after closing")
	}
	if mover.Pos == mustParse(t, "F10") {
		t.Errorf("A didn't move from F10")
	}
}

// TestExecuteCloseAndAttackAmbushedByOtherEnemy checks docs/core-rules.md
// section 6.3: contact with a different enemy than the named target stops
// the unit and ambushes it instead of attacking the target.
func TestExecuteCloseAndAttackAmbushedByOtherEnemy(t *testing.T) {
	templates, core := loadTestRules(t)
	attacker := newUnit(t, templates, "A", "south", "cavalry", "F8", hex.N)
	bystander := newUnit(t, templates, "Bystander", "north", "infantry", "G6", hex.SW)
	target := newUnit(t, templates, "Target", "north", "infantry", "F1", hex.S)
	state := GameState{Board: starterBoard(attacker, bystander, target)}

	o := orders.Order{Unit: "A", Type: orders.CloseAndAttack, TargetUnit: "Target"}
	state, events := executeCloseAndAttack(state, core, attacker, o)

	foundAmbush := false
	for _, e := range events {
		if e.Kind == "melee" && e.Melee.AttackerID == "Bystander" {
			foundAmbush = true
		}
		if e.Kind == "melee" && e.Melee.DefenderID == "Target" {
			t.Errorf("A attacked Target directly; want an ambush by Bystander instead")
		}
	}
	if !foundAmbush {
		t.Errorf("events = %+v, want an ambush by Bystander", events)
	}
}

func TestExecuteFireBasic(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F6", hex.N)
	gun.State = "ready"
	target := newUnit(t, templates, "B", "south", "infantry", "F2", hex.S)
	state := GameState{Board: starterBoard(gun, target)}

	state, events := executeFire(state, core, gun, orders.Order{Unit: "1st Guns", Type: orders.Fire, TargetUnit: "B"})
	if len(events) != 1 || events[0].Kind != "ranged" {
		t.Fatalf("events = %+v, want one ranged event", events)
	}
	hitTarget, ok := state.Board.Unit("B")
	if !ok || hitTarget.Strength != Half {
		t.Errorf("B = %+v, %v, want half-strength (EX-6: hit)", hitTarget, ok)
	}
}

func TestExecuteFireOutOfRangeNoEffect(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "A1", hex.S)
	gun.State = "ready"
	target := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	state := GameState{Board: starterBoard(gun, target)}

	_, events := executeFire(state, core, gun, orders.Order{Unit: "1st Guns", Type: orders.Fire, TargetUnit: "B"})
	if len(events) != 1 || events[0].Kind != "ranged" || events[0].Ranged.InRange {
		t.Fatalf("events = %+v, want one ranged event with InRange=false", events)
	}
}

func TestExecuteCloseAndFireMovesThenFires(t *testing.T) {
	templates, core := loadTestRules(t)
	// Cavalry: Move 7, Range 2. Starting 5 away leaves it needing to close
	// at least 3 hexes (half-move caps it at 3) to get within range 2.
	shooter := newUnit(t, templates, "A", "north", "cavalry", "F6", hex.S)
	target := newUnit(t, templates, "B", "south", "infantry", "F11", hex.N)
	state := GameState{Board: Board{Columns: 12, Rows: 12, Units: []UnitInstance{shooter, target}}}

	o := orders.Order{Unit: "A", Type: orders.CloseAndFire, TargetUnit: "B"}
	state, events := executeCloseAndFire(state, core, shooter, o)

	kinds := eventKinds(events)
	if len(kinds) == 0 || kinds[len(kinds)-1] != "ranged" {
		t.Fatalf("events kinds = %v, want to end with 'ranged'", kinds)
	}
}

func TestExecuteCloseAndFireArtilleryProhibited(t *testing.T) {
	templates, core := loadTestRules(t)
	gun := newUnit(t, templates, "1st Guns", "north", "artillery", "F6", hex.S)
	gun.State = "mobilised"
	target := newUnit(t, templates, "B", "south", "infantry", "F8", hex.N)
	state := GameState{Board: starterBoard(gun, target)}

	_, events := executeCloseAndFire(state, core, gun, orders.Order{Unit: "1st Guns", Type: orders.CloseAndFire, TargetUnit: "B"})
	if len(events) != 1 || events[0].Kind != "order-skipped" {
		t.Fatalf("events = %+v, want one 'order-skipped' event", events)
	}
}

func eventKinds(events []Event) []string {
	kinds := make([]string, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	return kinds
}
