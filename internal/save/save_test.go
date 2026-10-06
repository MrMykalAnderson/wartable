package save

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
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
