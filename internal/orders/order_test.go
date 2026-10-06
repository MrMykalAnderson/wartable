package orders

import (
	"strings"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// TestParseDevPlanExample parses the example sheet format from
// docs/dev-plan.md section 6 (the sheet there reuses "1st Guns" on two
// lines purely to illustrate syntax; a real sheet gives each unit at most
// one order per turn, so the last line here is renamed to "2nd Guns").
func TestParseDevPlanExample(t *testing.T) {
	sheet := `
# North, turn 3
1st Horse | Move             | F9 | facing S
2nd Foot  | Move             | 1st Guns
1st Guns  | Fire             | 3rd Foot
3rd Foot  | Close and Attack | 4th Horse
2nd Guns  | Ready            | facing SE
`
	got, err := Parse(sheet)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("Parse: got %d orders, want 5: %+v", len(got), got)
	}

	f9, _ := hex.ParseOffset("F9")
	want := []Order{
		{Line: 3, Unit: "1st Horse", Type: Move, HasTargetHex: true, TargetHex: f9, HasFacing: true, Facing: hex.S},
		{Line: 4, Unit: "2nd Foot", Type: Move, TargetUnit: "1st Guns"},
		{Line: 5, Unit: "1st Guns", Type: Fire, TargetUnit: "3rd Foot"},
		{Line: 6, Unit: "3rd Foot", Type: CloseAndAttack, TargetUnit: "4th Horse"},
		{Line: 7, Unit: "2nd Guns", Type: Ready, HasFacing: true, Facing: hex.SE},
	}
	for i, o := range got {
		if o != want[i] {
			t.Errorf("order %d = %+v, want %+v", i, o, want[i])
		}
	}
}

func TestParseCommentsAndBlankLines(t *testing.T) {
	sheet := "# just a comment\n\n1st Horse | Move | A1 # inline comment\n\n"
	got, err := Parse(sheet)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Unit != "1st Horse" {
		t.Fatalf("Parse: got %+v, want one order for 1st Horse", got)
	}
}

func TestParseRejectsUnknownOrder(t *testing.T) {
	if _, err := Parse("1st Horse | Charge | A1"); err == nil {
		t.Fatalf("Parse: want error for unknown order type, got none")
	}
}

func TestParseRejectsDuplicateUnit(t *testing.T) {
	_, err := Parse("1st Horse | Move | A1\n1st Horse | Fire | 2nd Foot\n")
	if err == nil {
		t.Fatalf("Parse: want error for duplicate unit, got none")
	}
}

func TestParseRejectsMissingTarget(t *testing.T) {
	cases := []string{
		"1st Horse | Deploy",
		"1st Horse | Move",
		"1st Horse | Close and Attack",
		"1st Horse | Fire",
		"1st Horse | Close and Fire",
	}
	for _, sheet := range cases {
		if _, err := Parse(sheet); err == nil {
			t.Errorf("Parse(%q): want error for missing target, got none", sheet)
		}
	}
}

func TestParseRejectsTargetOnReadyMobilise(t *testing.T) {
	if _, err := Parse("1st Guns | Ready | 2nd Foot"); err == nil {
		t.Fatalf("Parse: want error for Ready with a target, got none")
	}
}

func TestParseRejectsBadFacing(t *testing.T) {
	if _, err := Parse("1st Horse | Move | A1 | facing NNE"); err == nil {
		t.Fatalf("Parse: want error for bad facing, got none")
	}
}

func TestParseAccumulatesMultipleErrors(t *testing.T) {
	sheet := "1st Horse | Charge | A1\n2nd Foot | Move\n"
	_, err := Parse(sheet)
	if err == nil {
		t.Fatalf("Parse: want error, got none")
	}
	// errors.Join: both underlying problems should be mentioned.
	msg := err.Error()
	if !strings.Contains(msg, "Charge") || !strings.Contains(msg, "Move needs") {
		t.Errorf("Parse error = %q, want both line errors mentioned", msg)
	}
}

func TestParseReadyMobiliseNoTarget(t *testing.T) {
	got, err := Parse("1st Guns | Mobilise")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Type != Mobilise || got[0].HasFacing {
		t.Errorf("Parse = %+v, want one bare Mobilise order", got)
	}
}

func TestMoveTargetHexVsUnit(t *testing.T) {
	got, err := Parse("A | Move | A1\nB | Move | 1st Guns\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !got[0].HasTargetHex || got[0].TargetUnit != "" {
		t.Errorf("order 0 = %+v, want a hex target", got[0])
	}
	if got[1].HasTargetHex || got[1].TargetUnit != "1st Guns" {
		t.Errorf("order 1 = %+v, want a unit target", got[1])
	}
}
