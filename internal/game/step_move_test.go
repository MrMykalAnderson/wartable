package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
)

// TestStepRuleReferenceExamples ports every example from
// docs/tools/movement_reference.py (the authoritative cross-check for
// dev-plan.md section 7.11's step rule) against the real Two Towns map,
// checking the exact hex sequence, each step's kind and reason, and why
// the move stopped — all taken verbatim from that script's own printed
// output for data/maps/two-towns.yaml.
func TestStepRuleReferenceExamples(t *testing.T) {
	type wantStep struct {
		hex    string
		kind   string
		reason string
	}
	cases := []struct {
		name        string
		start       string
		dest        string
		enemy       string
		friend      string
		wantSteps   []wantStep
		wantReason  string
		wantContact bool
	}{
		{
			name: "Across the river, next to a bridge", start: "J14", dest: "S13",
			wantSteps: []wantStep{
				{"K14", "closer", "keep left"},
				{"L13", "closer", "keep left"},
				{"M13", "closer", "keep left"},
				{"N12", "closer", "keep left"},
			},
			wantReason: "used all its Move",
		},
		{
			name: "Across the river, far from a bridge", start: "H4", dest: "P2",
			wantSteps: []wantStep{
				{"I4", "closer", "keep left"},
				{"J3", "closer", "keep left"},
				{"J2", "slide", "only option"},
			},
			wantReason: "no open hex gets any closer",
		},
		{
			name: "Straight through the forest", start: "R12", dest: "R4",
			wantSteps: []wantStep{
				{"R11", "closer", "only option"},
				{"R10", "closer", "only option"},
				{"R9", "closer", "only option"},
			},
			wantReason: "used all its Move",
		},
		{
			name: "Into the forest", start: "H13", dest: "M9",
			wantSteps: []wantStep{
				{"H12", "closer", "keep left"},
				{"H11", "closer", "keep left"},
				{"I11", "closer", "only option"},
			},
			wantReason: "can't pay 2 for J10 (1 Move left)",
		},
		{
			name: "Road march over the upper bridge", start: "K7", dest: "S5",
			wantSteps: []wantStep{
				{"L6", "road", "only option"},
				{"M7", "road", "only option"},
				{"N6", "road", "only option"},
				{"O6", "road", "only option"},
				{"P5", "road", "only option"},
				{"Q5", "road", "only option"},
			},
			wantReason: "used all its Move",
		},
		{
			name: "Past an enemy: ties let it dodge", start: "D9", dest: "K9", enemy: "H7",
			wantSteps: []wantStep{
				{"E9", "closer", "keep left"},
				{"F8", "closer", "keep left"},
				{"G9", "closer", "avoids the enemy"},
				{"H9", "closer", "avoids the enemy"},
			},
			wantReason: "used all its Move",
		},
		{
			name: "Past an enemy: one straight line, so contact", start: "G13", dest: "G3", enemy: "H8",
			wantSteps: []wantStep{
				{"G12", "closer", "only option"},
				{"G11", "closer", "only option"},
				{"G10", "closer", "only option"},
				{"G9", "closer", "only option"},
			},
			wantReason:  "contact at G9",
			wantContact: true,
		},
		{
			name: "A friend in the way: keep left", start: "E14", dest: "E6", friend: "E11",
			wantSteps: []wantStep{
				{"E13", "closer", "only option"},
				{"E12", "closer", "only option"},
				{"D11", "slide", "keep left"},
				{"D10", "closer", "only option"},
			},
			wantReason: "used all its Move",
		},
	}

	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mover := newUnit(t, templates, "Mover", "north", "infantry", c.start, hex.N)
			units := []UnitInstance{mover}
			if c.enemy != "" {
				units = append(units, newUnit(t, templates, "Enemy", "south", "infantry", c.enemy, hex.N))
			}
			if c.friend != "" {
				units = append(units, newUnit(t, templates, "Friend", "north", "infantry", c.friend, hex.N))
			}
			board := boardFor(scenario, units...)

			result := StepMove(core, board, mover, mustParse(t, c.dest), mover.Stats(core).Move, "")

			if len(result.Path) != len(c.wantSteps) {
				t.Fatalf("Path = %v, want %d hexes matching %+v", result.Path, len(c.wantSteps), c.wantSteps)
			}
			if len(result.Steps) != len(c.wantSteps) {
				t.Fatalf("Steps = %+v, want %d entries", result.Steps, len(c.wantSteps))
			}
			for i, want := range c.wantSteps {
				if got := result.Path[i].String(); got != want.hex {
					t.Errorf("Path[%d] = %s, want %s (full path %v)", i, got, want.hex, result.Path)
				}
				if result.Steps[i].Kind != want.kind {
					t.Errorf("Steps[%d].Kind = %q, want %q (hex %s)", i, result.Steps[i].Kind, want.kind, want.hex)
				}
				if result.Steps[i].Reason != want.reason {
					t.Errorf("Steps[%d].Reason = %q, want %q (hex %s)", i, result.Steps[i].Reason, want.reason, want.hex)
				}
			}
			if result.StopReason != c.wantReason {
				t.Errorf("StopReason = %q, want %q", result.StopReason, c.wantReason)
			}
			if result.Contact != c.wantContact {
				t.Errorf("Contact = %v, want %v", result.Contact, c.wantContact)
			}
		})
	}
}

// TestStepRuleDodgeOptionsNoteTheEnemy checks that the avoided option in
// a tie is itself recorded with a "next to enemy" note, matching
// movement_reference.py's step 3 of "Past an enemy: ties let it dodge"
// (G8 noted, G9 chosen).
func TestStepRuleDodgeOptionsNoteTheEnemy(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "Mover", "north", "infantry", "D9", hex.N)
	enemy := newUnit(t, templates, "Enemy", "south", "infantry", "H7", hex.N)
	board := boardFor(scenario, mover, enemy)

	result := StepMove(core, board, mover, mustParse(t, "K9"), mover.Stats(core).Move, "")
	if len(result.Steps) < 3 {
		t.Fatalf("Steps = %+v, want at least 3", result.Steps)
	}
	step3 := result.Steps[2]
	if len(step3.Options) != 2 {
		t.Fatalf("step 3 Options = %+v, want 2 (G8, G9)", step3.Options)
	}
	var sawNotedG8 bool
	for _, o := range step3.Options {
		if o.Hex.String() == "G8" {
			sawNotedG8 = true
			if len(o.Notes) != 1 || o.Notes[0] != "next to enemy" {
				t.Errorf("G8 option Notes = %v, want [\"next to enemy\"]", o.Notes)
			}
		}
	}
	if !sawNotedG8 {
		t.Errorf("step 3 Options = %+v, want G8 among them", step3.Options)
	}
}

// TestStepRuleEX8DodgeOnTwoTowns checks EX-8 ("Dodging an enemy") from
// docs/core-rules.md section 11 via executeMove end to end: South's 2nd
// Foot at D9 ordered to K9 ends the turn at H9, having dodged North's
// 1st Foot at H7 twice.
func TestStepRuleEX8DodgeOnTwoTowns(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "2nd Foot", "south", "infantry", "D9", hex.N)
	enemy := newUnit(t, templates, "1st Foot", "north", "infantry", "H7", hex.N)
	state := GameState{Board: boardFor(scenario, mover, enemy)}

	o := orders.Order{Unit: "2nd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "K9")}
	state, events := executeMove(state, core, mover, o)

	moved, ok := state.Board.Unit("2nd Foot")
	if !ok {
		t.Fatalf("2nd Foot missing after move")
	}
	if moved.Pos != mustParse(t, "H9") {
		t.Errorf("2nd Foot.Pos = %s, want H9 (dodged twice, ran out of Move)", moved.Pos)
	}
	for _, e := range events {
		if e.Kind == "melee" || e.Kind == "unit-destroyed" {
			t.Errorf("events = %+v, want no combat (it never made contact)", events)
		}
	}
}

// TestStepRuleEX8ContactOnTwoTowns checks EX-8's "Contact on a straight
// line": South's 3rd Foot at G13 ordered to G3, straight up the column,
// is ambushed at G9 by North's 2nd Foot at H8 (there's no choice of hex
// to dodge through on this line).
func TestStepRuleEX8ContactOnTwoTowns(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "3rd Foot", "south", "infantry", "G13", hex.N)
	enemy := newUnit(t, templates, "2nd Foot", "north", "infantry", "H8", hex.SW)
	state := GameState{Board: boardFor(scenario, mover, enemy)}

	o := orders.Order{Unit: "3rd Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "G3")}
	_, events := executeMove(state, core, mover, o)

	// Checked against the "moved" event's own path, not 3rd Foot's final
	// board position: the ambush that follows contact may knock it back
	// from G9 to somewhere else, same as TestExecuteMoveAmbush.
	if len(events) == 0 || events[0].Kind != "moved" {
		t.Fatalf("events = %+v, want a 'moved' event first", events)
	}
	if n := len(events[0].Path); n == 0 || events[0].Path[n-1] != mustParse(t, "G9") {
		t.Errorf("moved event's Path = %v, want it to end at G9 (contact, no choice on a straight line)", events[0].Path)
	}
	foundMelee := false
	for _, e := range events {
		if e.Kind == "melee" {
			foundMelee = true
		}
	}
	if !foundMelee {
		t.Errorf("events = %+v, want a melee event from the ambush", events)
	}
}

// TestStepRuleEX8FriendInTheWay checks EX-8's "A friend in the way":
// North's 4th Foot at E14 ordered to E6 slides round a friendly unit at
// E11, keeping left to D11.
func TestStepRuleEX8FriendInTheWay(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	mover := newUnit(t, templates, "4th Foot", "north", "infantry", "E14", hex.N)
	friend := newUnit(t, templates, "Friend", "north", "infantry", "E11", hex.N)
	state := GameState{Board: boardFor(scenario, mover, friend)}

	o := orders.Order{Unit: "4th Foot", Type: orders.Move, HasTargetHex: true, TargetHex: mustParse(t, "E6")}
	state, _ = executeMove(state, core, mover, o)

	moved, ok := state.Board.Unit("4th Foot")
	if !ok {
		t.Fatalf("4th Foot missing after move")
	}
	if moved.Pos != mustParse(t, "D10") {
		t.Errorf("4th Foot.Pos = %s, want D10 (slid left round the friend at E11)", moved.Pos)
	}
}
