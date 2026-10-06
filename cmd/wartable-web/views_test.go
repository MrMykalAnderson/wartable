package main

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

func TestNewUnitView(t *testing.T) {
	u := game.UnitInstance{
		ID:       "1st Guns",
		Side:     "north",
		Template: rules.Unit{ID: "artillery"},
		Pos:      hex.Offset{Col: 2, Row: 3},
		Facing:   hex.SE,
		Strength: game.Half,
		State:    "ready",
	}
	got := newUnitView(u)
	want := UnitView{ID: "1st Guns", Side: "north", Type: "artillery", Col: 2, Row: 3, Facing: "SE", Strength: "half", State: "ready"}
	if got != want {
		t.Errorf("newUnitView = %+v, want %+v", got, want)
	}
}

func TestNewGameView(t *testing.T) {
	f := save.File{
		ScenarioID: "starter-battle",
		Turn:       4,
		TieBreak:   game.TieBreak{Holder: "south"},
		State: game.GameState{
			Board: game.Board{
				Columns: 12,
				Rows:    10,
				Units: []game.UnitInstance{
					{ID: "A", Side: "north", Template: rules.Unit{ID: "infantry"}, Facing: hex.N, Strength: game.Full},
				},
			},
			Reserves: map[string][]game.UnitInstance{
				"south": {{ID: "B"}},
			},
		},
	}
	got := newGameView(f)
	if got.ScenarioID != "starter-battle" || got.Turn != 4 || got.TieBreakHolder != "south" {
		t.Errorf("newGameView header fields = %+v", got)
	}
	if got.Columns != 12 || got.Rows != 10 {
		t.Errorf("newGameView map size = %dx%d, want 12x10", got.Columns, got.Rows)
	}
	if len(got.Units) != 1 || got.Units[0].ID != "A" {
		t.Errorf("newGameView.Units = %+v", got.Units)
	}
	if len(got.Reserves["south"]) != 1 || got.Reserves["south"][0] != "B" {
		t.Errorf("newGameView.Reserves = %+v", got.Reserves)
	}
	if got.Reserves["north"] != nil {
		t.Errorf("newGameView.Reserves[north] = %v, want nil (no reserves given)", got.Reserves["north"])
	}
}

func TestNewEventView(t *testing.T) {
	e := game.Event{Kind: "moved", Unit: "A", Detail: "moved to B2"}
	got := newEventView(e)
	if got.Kind != "moved" || got.Unit != "A" || got.Summary != "[moved] A: moved to B2" {
		t.Errorf("newEventView = %+v", got)
	}
}
