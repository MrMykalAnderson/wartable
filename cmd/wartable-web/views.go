package main

import (
	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
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
	Attack   int `json:"attack"`
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
			Attack:   s.Attack,
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

// EdgeView is one edge between two adjacent hexes (docs/dev-plan.md
// section 7.8: a river or bridge edge, drawn along the hex boundary
// rather than filling a hex).
type EdgeView struct {
	A HexView `json:"a"`
	B HexView `json:"b"`
}

func newEdgeView(e hex.Edge) EdgeView {
	return EdgeView{A: HexView{Col: e.A.Col, Row: e.A.Row}, B: HexView{Col: e.B.Col, Row: e.B.Row}}
}

// TerrainView is a scenario's map terrain as seen by the web viewer
// (docs/dev-plan.md section 7.8, interface item 2): the server's own
// terrain data, laid out for drawing directly — hex fills for forest,
// towns and hamlets, road chains as polylines through hex centres,
// river and bridge edges along hex boundaries. Never recomputed from
// raw rules in JavaScript.
type TerrainView struct {
	Forest  []HexView            `json:"forest"`
	Roads   [][]HexView          `json:"roads"`
	River   []EdgeView           `json:"river"`
	Bridges []EdgeView           `json:"bridges"`
	Towns   map[string][]HexView `json:"towns"`
	Hamlets map[string][]HexView `json:"hamlets"`
}

func hexViews(hexes []hex.Offset) []HexView {
	views := make([]HexView, len(hexes))
	for i, h := range hexes {
		views[i] = HexView{Col: h.Col, Row: h.Row}
	}
	return views
}

// newTerrainView returns nil for a scenario with no terrain (e.g. the
// Starter Battle), so the frontend draws nothing extra for it.
func newTerrainView(m *rules.TerrainMap) *TerrainView {
	if m == nil {
		return nil
	}
	v := &TerrainView{
		Towns:   map[string][]HexView{},
		Hamlets: map[string][]HexView{},
	}
	for h := range m.Forest {
		v.Forest = append(v.Forest, HexView{Col: h.Col, Row: h.Row})
	}
	for _, hexes := range m.Roads {
		v.Roads = append(v.Roads, hexViews(hexes))
	}
	for e := range m.River {
		v.River = append(v.River, newEdgeView(e))
	}
	for e := range m.Bridge {
		v.Bridges = append(v.Bridges, newEdgeView(e))
	}
	for name, hexes := range m.Towns {
		v.Towns[name] = hexViews(hexes)
	}
	for name, hexes := range m.Hamlets {
		v.Hamlets[name] = hexViews(hexes)
	}
	return v
}

// GameView is a saved game as seen by the web viewer. History is every
// turn played so far (docs/dev-plan.md section 7.7), so the viewer can
// step through any past turn right after loading, not just ones played
// in the current browser session. Terrain is nil for a scenario with
// none; Objectives maps each of its towns/hamlets to its owning side,
// omitted entirely for an objective with no owner yet (docs/dev-plan.md
// section 7.8).
type GameView struct {
	ScenarioID     string              `json:"scenarioId"`
	Turn           int                 `json:"turn"`
	TieBreakHolder string              `json:"tieBreakHolder"`
	Columns        int                 `json:"columns"`
	Rows           int                 `json:"rows"`
	Units          []UnitView          `json:"units"`
	Reserves       map[string][]string `json:"reserves"`
	CoreRules      CoreRulesView       `json:"coreRules"`
	History        []TurnRecordView    `json:"history"`
	Terrain        *TerrainView        `json:"terrain,omitempty"`
	Objectives     map[string]string   `json:"objectives,omitempty"`
	// DefaultFacing is each side's facing on Deploy if an order doesn't
	// set one explicitly (docs/dev-plan.md section 7.9: facing is
	// optional), for the frontend to show what a pending Deploy order
	// will end up facing without having to ask the server.
	DefaultFacing map[string]string `json:"defaultFacing"`
}

func newGameView(f save.File, core rules.CoreRules, scenario rules.Scenario) GameView {
	board := newBoardView(f.State.Board, core)
	reserves := make(map[string][]string, len(f.State.Reserves))
	for side, us := range f.State.Reserves {
		ids := make([]string, len(us))
		for i, u := range us {
			ids[i] = u.ID
		}
		reserves[side] = ids
	}
	history := make([]TurnRecordView, len(f.History))
	for i, t := range f.History {
		history[i] = newTurnRecordView(t, core)
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
		History:        history,
		Terrain:        newTerrainView(scenario.Terrain),
		Objectives:     f.State.Objectives,
		DefaultFacing:  scenario.DefaultFacing,
	}
}

// TurnRecordView is one played turn's full record as seen by the web
// viewer: the order sheets submitted and the resulting event log, with
// the board exactly as it stood before the turn ran.
type TurnRecordView struct {
	Turn        int         `json:"turn"`
	NorthOrders string      `json:"northOrders"`
	SouthOrders string      `json:"southOrders"`
	BoardBefore BoardView   `json:"boardBefore"`
	Events      []EventView `json:"events"`
}

func newTurnRecordView(t save.TurnRecord, core rules.CoreRules) TurnRecordView {
	events := make([]EventView, len(t.Events))
	for i, e := range t.Events {
		events[i] = newEventView(e, core)
	}
	return TurnRecordView{
		Turn:        t.Turn,
		NorthOrders: t.NorthOrders,
		SouthOrders: t.SouthOrders,
		BoardBefore: newBoardView(t.BoardBefore, core),
		Events:      events,
	}
}

// MeleeView is a melee combat's full, auditable record (docs/core-rules.md
// section 8.3), broken into every number used, for a detailed step-by-
// step view (docs/dev-plan.md section 7.3) rather than just the final
// totals. Combat is one comparison: Margin decides Hits (0 repelled/1
// hit/2 destroyed outright).
type MeleeView struct {
	AttackerID      string `json:"attackerId"`
	DefenderID      string `json:"defenderId"`
	Edge            string `json:"edge"`
	PositionBonus   int    `json:"positionBonus"`
	AttackerSupport int    `json:"attackerSupport"`
	DefenderSupport int    `json:"defenderSupport"`
	Ambushed        bool   `json:"ambushed"`
	AmbushPenalty   int    `json:"ambushPenalty"`

	AttackerBase  int `json:"attackerBase"`
	DefenderBase  int `json:"defenderBase"`
	AttackerTotal int `json:"attackerTotal"`
	DefenderTotal int `json:"defenderTotal"`
	Margin        int `json:"margin"`

	WinnerID string `json:"winnerId"`
	LoserID  string `json:"loserId"`
	Hits     int    `json:"hits"`

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
		AmbushPenalty:   m.AmbushPenalty,
		AttackerBase:    m.AttackerBase,
		DefenderBase:    m.DefenderBase,
		AttackerTotal:   m.AttackerTotal,
		DefenderTotal:   m.DefenderTotal,
		Margin:          m.Margin,
		WinnerID:        m.WinnerID,
		LoserID:         m.LoserID,
		Hits:            m.Hits,
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

	ShooterSupport int `json:"shooterSupport"`
	TargetSupport  int `json:"targetSupport"`
	ShooterBase    int `json:"shooterBase"`
	TargetBase     int `json:"targetBase"`
	ShooterTotal   int `json:"shooterTotal"`
	TargetTotal    int `json:"targetTotal"`
	Margin         int `json:"margin"`

	Hits      int  `json:"hits"`
	Destroyed bool `json:"destroyed"`
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
		ShooterBase:    r.ShooterBase,
		TargetBase:     r.TargetBase,
		ShooterTotal:   r.ShooterTotal,
		TargetTotal:    r.TargetTotal,
		Margin:         r.Margin,
		Hits:           r.Hits,
		Destroyed:      r.Destroyed,
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
