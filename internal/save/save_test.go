package save

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

func TestWriteLoadRoundTrip(t *testing.T) {
	want := File{
		ScenarioID: "starter-battle",
		Turn:       3,
		TieBreak:   game.TieBreak{Holder: "south"},
		State: game.GameState{
			Board: game.Board{Columns: 12, Rows: 10},
		},
	}
	path := filepath.Join(t.TempDir(), "state.json")

	if err := Write(path, want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ScenarioID != want.ScenarioID || got.Turn != want.Turn || got.TieBreak != want.TieBreak {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
	if got.State.Board.Columns != want.State.Board.Columns || got.State.Board.Rows != want.State.Board.Rows {
		t.Errorf("Load.State.Board = %+v, want %+v", got.State.Board, want.State.Board)
	}
}

// TestNewGame checks docs/starter-battle.md "Armies": both sides seeded
// with 2 Infantry, 1 Cavalry, 1 Artillery, named "<Side> <1st/2nd> <Type>".
func TestNewGame(t *testing.T) {
	units := map[string]rules.Unit{
		"infantry":  {ID: "infantry", Name: "Infantry", DeployState: ""},
		"cavalry":   {ID: "cavalry", Name: "Cavalry", DeployState: ""},
		"artillery": {ID: "artillery", Name: "Artillery", DeployState: "mobilised"},
	}
	scenario := rules.Scenario{
		ID: "starter-battle", TieBreakHolder: "north",
		Map: rules.MapSize{Columns: 12, Rows: 10},
	}

	f := NewGame(units, scenario)
	if f.ScenarioID != "starter-battle" || f.Turn != 1 || f.TieBreak.Holder != "north" {
		t.Errorf("NewGame header fields = %+v", f)
	}
	if f.State.Board.Columns != 12 || f.State.Board.Rows != 10 {
		t.Errorf("NewGame.State.Board = %+v, want 12x10", f.State.Board)
	}

	for _, side := range []string{"north", "south"} {
		army := f.State.Reserves[side]
		if len(army) != 4 {
			t.Fatalf("%s reserves = %+v, want 4 units", side, army)
		}
		wantIDs := []string{
			capitalize(side) + " 1st Infantry",
			capitalize(side) + " 2nd Infantry",
			capitalize(side) + " 1st Cavalry",
			capitalize(side) + " 1st Artillery",
		}
		for i, u := range army {
			if u.ID != wantIDs[i] {
				t.Errorf("%s reserves[%d].ID = %q, want %q", side, i, u.ID, wantIDs[i])
			}
			if u.Side != side || u.Strength != game.Full {
				t.Errorf("%s reserves[%d] = %+v, want Side %s, Full strength", side, i, u, side)
			}
		}
		if army[3].State != "mobilised" {
			t.Errorf("%s artillery State = %q, want mobilised (its deploy_state)", side, army[3].State)
		}
	}
}

func TestOrdinalAndCapitalize(t *testing.T) {
	cases := map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th"}
	for n, want := range cases {
		if got := ordinal(n); got != want {
			t.Errorf("ordinal(%d) = %q, want %q", n, got, want)
		}
	}
	if got := capitalize("north"); got != "North" {
		t.Errorf("capitalize(north) = %q, want North", got)
	}
}

// TestLoadRulesData checks that UnitsPath/CoreRulesPath/ScenarioPath
// resolve correctly from the repository root, which is where
// cmd/wartable and cmd/wartable-web are meant to run from.
func TestLoadRulesData(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(filepath.Join(wd, "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(wd)

	units, core, scenario, err := LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}
	if len(units) == 0 {
		t.Errorf("LoadRulesData: no units loaded")
	}
	if core.SupportPerAlly == 0 {
		t.Errorf("LoadRulesData: core rules look empty: %+v", core)
	}
	if scenario.ID != "starter-battle" {
		t.Errorf("LoadRulesData: scenario.ID = %q, want starter-battle", scenario.ID)
	}
}

// TestRecordTurnRoundTrip checks docs/dev-plan.md section 7.7: a save
// keeps every turn's order sheets and event log, and it survives a
// write/load round trip.
func TestRecordTurnRoundTrip(t *testing.T) {
	f := File{ScenarioID: "starter-battle", Turn: 1}
	boardBefore := game.Board{Columns: 12, Rows: 10}
	events := []game.Event{{Kind: "deployed", Unit: "A", Detail: "deployed to B2", Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{{ID: "A"}}}}}
	f.RecordTurn(boardBefore, "A | Deploy | B2", "", events)
	f.Turn++

	if len(f.History) != 1 {
		t.Fatalf("History = %+v, want 1 record", f.History)
	}
	if f.History[0].Turn != 1 || f.History[0].NorthOrders != "A | Deploy | B2" || f.History[0].SouthOrders != "" {
		t.Errorf("History[0] = %+v", f.History[0])
	}
	if len(f.History[0].Events) != 1 || f.History[0].Events[0].Unit != "A" {
		t.Errorf("History[0].Events = %+v", f.History[0].Events)
	}

	path := filepath.Join(t.TempDir(), "state.json")
	if err := Write(path, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.History) != 1 || got.History[0].NorthOrders != "A | Deploy | B2" {
		t.Errorf("reloaded History = %+v", got.History)
	}
	if len(got.History[0].Events) != 1 || got.History[0].Events[0].Board.Units[0].ID != "A" {
		t.Errorf("reloaded History[0].Events[0].Board = %+v", got.History[0].Events[0].Board)
	}
}
