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

// BoardView is a board's units at one point in time: the live board (as
// part of GameView) or the board immediately after one event (as part of
// EventView), for step-by-step map replay (docs/dev-plan.md section
// 7.3).
type BoardView struct {
	Columns int        `json:"columns"`
	Rows    int        `json:"rows"`
	Units   []UnitView `json:"units"`
}

func newBoardView(b game.Board, core rules.CoreRules) BoardView {
	units := make([]UnitView, len(b.Units))
	for i, u := range b.Units {
		units[i] = newUnitView(u, core)
	}
	return BoardView{Columns: b.Columns, Rows: b.Rows, Units: units}
}

// CoreRulesView is the subset of core rule numbers the frontend needs to
// label a calculation breakdown (docs/dev-plan.md section 7.3) without
// ever deciding a rule itself.
type CoreRulesView struct {
	PositionBonus struct {
		Front int `json:"front"`
		Flank int `json:"flank"`
		Rear  int `json:"rear"`
	} `json:"positionBonus"`
	AmbushDefPenalty    int `json:"ambushDefPenalty"`
	HalfStrengthPenalty int `json:"halfStrengthPenalty"`
	SupportPerAlly      int `json:"supportPerAlly"`
}

func newCoreRulesView(core rules.CoreRules) CoreRulesView {
	v := CoreRulesView{
		AmbushDefPenalty:    core.AmbushDefPenalty,
		HalfStrengthPenalty: core.HalfStrengthPenalty,
		SupportPerAlly:      core.SupportPerAlly,
	}
	v.PositionBonus.Front = core.PositionBonus.Front
	v.PositionBonus.Flank = core.PositionBonus.Flank
	v.PositionBonus.Rear = core.PositionBonus.Rear
	return v
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
	CoreRules      CoreRulesView       `json:"coreRules"`
}

func newGameView(f save.File, core rules.CoreRules) GameView {
	board := newBoardView(f.State.Board, core)
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
		Columns:        board.Columns,
		Rows:           board.Rows,
		Units:          board.Units,
		Reserves:       reserves,
		CoreRules:      newCoreRulesView(core),
	}
}

// MeleeView is a melee combat's full, auditable record (docs/core-rules.md
// section 8), broken into every number used, for a detailed step-by-step
// view (docs/dev-plan.md section 7.3) rather than just the final totals.
type MeleeView struct {
	AttackerID      string `json:"attackerId"`
	DefenderID      string `json:"defenderId"`
	Edge            string `json:"edge"`
	PositionBonus   int    `json:"positionBonus"`
	AttackerSupport int    `json:"attackerSupport"`
	DefenderSupport int    `json:"defenderSupport"`
	Ambushed        bool   `json:"ambushed"`

	AttackerTotal int  `json:"attackerTotal"`
	DefenderTotal int  `json:"defenderTotal"`
	AttackerWins  bool `json:"attackerWins"`

	WinnerID  string `json:"winnerId"`
	LoserID   string `json:"loserId"`
	Damage    int    `json:"damage"`
	DamageDef int    `json:"damageDef"`
	DamageHit bool   `json:"damageHit"`

	LoserDestroyed   bool     `json:"loserDestroyed"`
	KnockbackTo      *HexView `json:"knockbackTo,omitempty"`
	KnockbackBlocked bool     `json:"knockbackBlocked,omitempty"`
}

func newMeleeView(m *game.MeleeResult) *MeleeView {
	if m == nil {
		return nil
	}
	v := &MeleeView{
		AttackerID:      m.AttackerID,
		DefenderID:      m.DefenderID,
		Edge:            m.Edge.String(),
		PositionBonus:   m.PositionBonus,
		AttackerSupport: m.AttackerSupport,
		DefenderSupport: m.DefenderSupport,
		Ambushed:        m.Ambushed,
		AttackerTotal:   m.Hit.AttackerTotal,
		DefenderTotal:   m.Hit.DefenderTotal,
		AttackerWins:    m.Hit.AttackerWins,
		WinnerID:        m.WinnerID,
		LoserID:         m.LoserID,
		Damage:          m.Damage.Damage,
		DamageDef:       m.Damage.Def,
		DamageHit:       m.Damage.Hit,
		LoserDestroyed:  m.LoserDestroyed,
	}
	if m.Knockback != nil {
		v.KnockbackTo = &HexView{Col: m.Knockback.To.Col, Row: m.Knockback.To.Row}
		v.KnockbackBlocked = m.Knockback.Destroyed
	}
	return v
}

// RangedView is a ranged attack's full, auditable record (docs/core-
// rules.md section 9), broken into every number used.
type RangedView struct {
	ShooterID string `json:"shooterId"`
	TargetID  string `json:"targetId"`
	InRange   bool   `json:"inRange"`
	InArc     bool   `json:"inArc"`

	ShooterSupport int  `json:"shooterSupport"`
	TargetSupport  int  `json:"targetSupport"`
	ShooterTotal   int  `json:"shooterTotal"`
	TargetTotal    int  `json:"targetTotal"`
	Hit            bool `json:"hit"`

	RngDmg    int  `json:"rngDmg"`
	DamageDef int  `json:"damageDef"`
	DamageHit bool `json:"damageHit"`
}

func newRangedView(r *game.RangedResult) *RangedView {
	if r == nil {
		return nil
	}
	return &RangedView{
		ShooterID:      r.ShooterID,
		TargetID:       r.TargetID,
		InRange:        r.InRange,
		InArc:          r.InArc,
		ShooterSupport: r.ShooterSupport,
		TargetSupport:  r.TargetSupport,
		ShooterTotal:   r.HitCheck.ShooterTotal,
		TargetTotal:    r.HitCheck.TargetTotal,
		Hit:            r.HitCheck.Hit,
		RngDmg:         r.Damage.RngDmg,
		DamageDef:      r.Damage.Def,
		DamageHit:      r.Damage.Hit,
	}
}

// EventView is one turn-execution event as seen by the web viewer: Kind
// and Unit for programmatic use (e.g. highlighting Unit on the map),
// Summary for a one-line display, Board for replaying the map at this
// exact step, Path for the route a "moved" event's unit actually took,
// and Melee/Ranged for a full calculation breakdown.
type EventView struct {
	Kind    string      `json:"kind"`
	Unit    string      `json:"unit"`
	Summary string      `json:"summary"`
	Board   BoardView   `json:"board"`
	Path    []HexView   `json:"path,omitempty"`
	Melee   *MeleeView  `json:"melee,omitempty"`
	Ranged  *RangedView `json:"ranged,omitempty"`
}

func newEventView(e game.Event, core rules.CoreRules) EventView {
	var path []HexView
	if len(e.Path) > 0 {
		path = make([]HexView, len(e.Path))
		for i, h := range e.Path {
			path[i] = HexView{Col: h.Col, Row: h.Row}
		}
	}
	return EventView{
		Kind:    e.Kind,
		Unit:    e.Unit,
		Summary: e.Summary(),
		Board:   newBoardView(e.Board, core),
		Path:    path,
		Melee:   newMeleeView(e.Melee),
		Ranged:  newRangedView(e.Ranged),
	}
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
