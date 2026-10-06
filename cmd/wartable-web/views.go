package main

import (
	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

// StatsView is a unit's effective stats (docs/core-rules.md sections
// 3.1-3.2: half strength already applied), for the unit detail card
// (docs/dev-plan.md section 7.3, interface item 3).
type StatsView struct {
	Def      int `json:"def"`
	Move     int `json:"move"`
	MinRange int `json:"minRange"`
	MaxRange int `json:"maxRange"`
	RngDmg   int `json:"rngDmg"`
	Attack   int `json:"attack"`
	AttDmg   int `json:"attDmg"`
}

// UnitView is a unit as seen by the web viewer.
type UnitView struct {
	ID       string    `json:"id"`
	Side     string    `json:"side"`
	Type     string    `json:"type"`
	TypeName string    `json:"typeName"`
	Cost     int       `json:"cost"`
	Col      int       `json:"col"`
	Row      int       `json:"row"`
	Facing   string    `json:"facing"`
	Strength string    `json:"strength"`
	State    string    `json:"state,omitempty"`
	Stats    StatsView `json:"stats"`
}

func newUnitView(u game.UnitInstance, core rules.CoreRules) UnitView {
	strength := "full"
	if u.Strength == game.Half {
		strength = "half"
	}
	s := u.Stats(core)
	return UnitView{
		ID:       u.ID,
		Side:     u.Side,
		Type:     u.Template.ID,
		TypeName: u.Template.Name,
		Cost:     u.Template.Cost,
		Col:      u.Pos.Col,
		Row:      u.Pos.Row,
		Facing:   u.Facing.String(),
		Strength: strength,
		State:    u.State,
		Stats: StatsView{
			Def:      s.Def,
			Move:     s.Move,
			MinRange: s.MinRange,
			MaxRange: s.MaxRange,
			RngDmg:   s.RngDmg,
			Attack:   s.Attack,
			AttDmg:   s.AttDmg,
		},
	}
}

// GameView is a saved game as seen by the web viewer.
type GameView struct {
	ScenarioID     string              `json:"scenarioId"`
	Turn           int                 `json:"turn"`
	TieBreakHolder string              `json:"tieBreakHolder"`
	Columns        int                 `json:"columns"`
	Rows           int                 `json:"rows"`
	Units          []UnitView          `json:"units"`
	Reserves       map[string][]string `json:"reserves"`
}

func newGameView(f save.File, core rules.CoreRules) GameView {
	units := make([]UnitView, len(f.State.Board.Units))
	for i, u := range f.State.Board.Units {
		units[i] = newUnitView(u, core)
	}
	reserves := make(map[string][]string, len(f.State.Reserves))
	for side, us := range f.State.Reserves {
		ids := make([]string, len(us))
		for i, u := range us {
			ids[i] = u.ID
		}
		reserves[side] = ids
	}
	return GameView{
		ScenarioID:     f.ScenarioID,
		Turn:           f.Turn,
		TieBreakHolder: f.TieBreak.Holder,
		Columns:        f.State.Board.Columns,
		Rows:           f.State.Board.Rows,
		Units:          units,
		Reserves:       reserves,
	}
}

// EventView is one turn-execution event as seen by the web viewer: Kind
// and Unit for programmatic use (e.g. highlighting Unit on the map),
// Summary for display.
type EventView struct {
	Kind    string `json:"kind"`
	Unit    string `json:"unit"`
	Summary string `json:"summary"`
}

func newEventView(e game.Event) EventView {
	return EventView{Kind: e.Kind, Unit: e.Unit, Summary: e.Summary()}
}

// OutcomeView is a win-condition check as seen by the web viewer.
type OutcomeView struct {
	Over   bool   `json:"over"`
	Winner string `json:"winner,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func newOutcomeView(o game.Outcome) OutcomeView {
	return OutcomeView{Over: o.Over, Winner: o.Winner, Reason: o.Reason}
}
