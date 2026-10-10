package main

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

func TestNewUnitView(t *testing.T) {
	core := rules.CoreRules{HalfStrengthPenalty: 1}
	u := game.UnitInstance{
		ID:   "1st Guns",
		Side: "north",
		Template: rules.Unit{
			ID: "artillery", Name: "Artillery", Cost: 15,
			Def:      rules.Def{ByState: map[string]int{"ready": 4, "mobilised": 2}},
			MinRange: 3, Range: 5, Attack: 3,
		},
		Pos:      hex.Offset{Col: 2, Row: 3},
		Facing:   hex.SE,
		Strength: game.Half,
		State:    "ready",
	}
	got := newUnitView(u, core)
	want := UnitView{
		ID: "1st Guns", Side: "north", Type: "artillery", TypeName: "Artillery", Cost: 15,
		Col: 2, Row: 3, Facing: "SE", Strength: "half", State: "ready",
		// Half strength: Def 4-1=3, MaxRange 5-1=4, Attack 3-1=2; MinRange
		// is unaffected (docs/core-rules.md section 3.2).
		Stats: StatsView{Def: 3, MinRange: 3, MaxRange: 4, Attack: 2},
	}
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
	got := newGameView(f, rules.CoreRules{HalfStrengthPenalty: 1}, rules.Scenario{ID: "starter-battle"})
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

// TestNewTerrainView checks docs/dev-plan.md section 7.8's interface
// item 2 against the real Two Towns map: terrain is nil for a scenario
// with none (the Starter Battle), and populated, with the real bridge
// edges, for one that has it.
func TestNewTerrainView(t *testing.T) {
	chdirToRepoRoot(t)
	units, err := rules.LoadUnits("data/units/standard.yaml")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	starter, err := rules.LoadScenario("data/scenarios/starter-battle.yaml", units)
	if err != nil {
		t.Fatalf("LoadScenario(starter-battle): %v", err)
	}
	if got := newTerrainView(starter.Terrain); got != nil {
		t.Errorf("newTerrainView(starter-battle) = %+v, want nil (no terrain)", got)
	}

	twoTowns, err := rules.LoadScenario("data/scenarios/two-towns.yaml", units)
	if err != nil {
		t.Fatalf("LoadScenario(two-towns): %v", err)
	}
	got := newTerrainView(twoTowns.Terrain)
	if got == nil {
		t.Fatalf("newTerrainView(two-towns) = nil, want populated")
	}
	if len(got.Bridges) != 2 {
		t.Errorf("Bridges = %+v, want 2 (M7|N6 and O12|P11)", got.Bridges)
	}
	if len(got.Towns["North town"]) == 0 || len(got.Towns["South town"]) == 0 {
		t.Errorf("Towns = %+v, want both North town and South town", got.Towns)
	}
	if len(got.Forest) == 0 {
		t.Errorf("Forest is empty, want the forest straddling the river")
	}
}

func TestNewEventView(t *testing.T) {
	core := rules.CoreRules{HalfStrengthPenalty: 1}
	board := game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{
		{ID: "A", Side: "north", Template: rules.Unit{ID: "infantry"}, Pos: hex.Offset{Col: 1, Row: 1}, Facing: hex.S, Strength: game.Full},
	}}
	e := game.Event{Kind: "moved", Unit: "A", Detail: "moved to B2", Board: board}
	got := newEventView(e, core)
	if got.Kind != "moved" || got.Unit != "A" || got.Summary != "[moved] A: moved to B2" {
		t.Errorf("newEventView header fields = %+v", got)
	}
	if len(got.Board.Units) != 1 || got.Board.Units[0].ID != "A" {
		t.Errorf("newEventView.Board = %+v, want the one unit A", got.Board)
	}
	if got.Melee != nil || got.Ranged != nil {
		t.Errorf("newEventView.Melee/Ranged = %+v/%+v, want both nil (no combat result given)", got.Melee, got.Ranged)
	}
}

func TestNewMeleeView(t *testing.T) {
	m := &game.MeleeResult{
		AttackerID: "A", DefenderID: "B", Edge: hex.Flank,
		PositionBonus: 1, AttackerSupport: 1, DefenderSupport: 0, Ambushed: false,
		AttackerBase: 2, DefenderBase: 2,
		AttackerTotal: 3, DefenderTotal: 2, Margin: 1,
		WinnerID: "A", LoserID: "B", Hits: 1,
		Knockback: &game.Knockback{To: hex.Offset{Col: 4, Row: 6}, Destroyed: false},
	}
	got := newMeleeView(m)
	if got == nil {
		t.Fatalf("newMeleeView returned nil")
	}
	if got.AttackerID != "A" || got.DefenderID != "B" || got.Edge != "flank" {
		t.Errorf("newMeleeView identity fields = %+v", got)
	}
	if got.AttackerTotal != 3 || got.DefenderTotal != 2 || got.Margin != 1 {
		t.Errorf("newMeleeView totals = %+v", got)
	}
	if got.Hits != 1 || got.LoserDestroyed {
		t.Errorf("newMeleeView outcome = %+v", got)
	}
	if got.KnockbackTo == nil || *got.KnockbackTo != (HexView{Col: 4, Row: 6}) || got.KnockbackBlocked {
		t.Errorf("newMeleeView knockback = %+v", got.KnockbackTo)
	}
}

func TestNewMeleeViewNil(t *testing.T) {
	if got := newMeleeView(nil); got != nil {
		t.Errorf("newMeleeView(nil) = %+v, want nil", got)
	}
}

func TestNewRangedView(t *testing.T) {
	r := &game.RangedResult{
		ShooterID: "A", TargetID: "B", InRange: true, InArc: true,
		ShooterSupport: 0, TargetSupport: 1,
		ShooterBase: 3, TargetBase: 2,
		ShooterTotal: 3, TargetTotal: 3, Margin: 0,
	}
	got := newRangedView(r)
	if got == nil {
		t.Fatalf("newRangedView returned nil")
	}
	if got.ShooterTotal != 3 || got.TargetTotal != 3 || got.Hits != 0 {
		t.Errorf("newRangedView = %+v, want a miss at 3/3", got)
	}
}

func TestNewCoreRulesView(t *testing.T) {
	core := rules.CoreRules{
		PositionBonus:       rules.PositionBonus{Front: 0, Flank: 1, Rear: 3},
		AmbushDefPenalty:    1,
		HalfStrengthPenalty: 1,
		SupportPerAlly:      1,
	}
	got := newCoreRulesView(core)
	if got.PositionBonus.Front != 0 || got.PositionBonus.Flank != 1 || got.PositionBonus.Rear != 3 {
		t.Errorf("newCoreRulesView.PositionBonus = %+v", got.PositionBonus)
	}
	if got.AmbushDefPenalty != 1 || got.HalfStrengthPenalty != 1 || got.SupportPerAlly != 1 {
		t.Errorf("newCoreRulesView = %+v", got)
	}
}
