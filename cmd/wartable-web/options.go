package main

import (
	"fmt"
	"net/http"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

// HexView is a hex, for the frontend to highlight.
type HexView struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

// OptionsView is what click-to-order needs to know about one unit's
// available orders this turn, plus its range visualiser (docs/dev-plan.md
// sections 7.1 and 7.3): computed by the engine, never re-derived in
// JavaScript.
//
// MoveHexes/CloseMoveHexes are a reach *guide*, not a restriction
// (docs/dev-plan.md section 7.6): pure distance from the unit's current
// position, ignoring every other unit on the board. A Move order's path
// is only worked out at execution, by which time other units (friendly
// and enemy) will have moved, so planning can't know what's actually in
// the way — any hex, and any enemy for an attack/Fire order, is a valid
// order target regardless of what these fields show.
type OptionsView struct {
	IsReserve   bool      `json:"isReserve"`
	MoveHexes   []HexView `json:"moveHexes"`
	DeployHexes []HexView `json:"deployHexes"`
	FireHexes   []HexView `json:"fireHexes"`
	// CloseMoveHexes is the shorter, half-move reach a Close and Attack
	// or Close and Fire order can close across (docs/core-rules.md
	// section 6.3/6.5), shown distinctly from MoveHexes' full reach
	// (docs/dev-plan.md section 7.5, interface item 1). Empty if the
	// unit can't make either order (e.g. artillery, which can't Close).
	CloseMoveHexes   []HexView `json:"closeMoveHexes"`
	CloseMove        int       `json:"closeMove"` // the half-move distance itself, for the "Close: N hexes" label.
	MeleeTargets     []string  `json:"meleeTargets"`
	FireTargets      []string  `json:"fireTargets"`
	CloseFireTargets []string  `json:"closeFireTargets"`
	CanReady         bool      `json:"canReady"`
	CanMobilise      bool      `json:"canMobilise"`
	// HasBarrage is true if this unit, in its current state, fires a
	// Barrage at the start of every Execution Phase (docs/core-rules.md
	// section 3.3): FireHexes doubles as its barrage zone.
	HasBarrage bool `json:"hasBarrage"`
}

// handleOptions serves GET /api/options?path=&side=&unit=: the hexes and
// enemies a unit's order could legally target this turn, for the
// click-to-order UI to highlight. It never changes the saved game.
func handleOptions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	side := r.URL.Query().Get("side")
	unitID := r.URL.Query().Get("unit")
	if path == "" || side == "" || unitID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing path, side or unit"))
		return
	}

	f, err := save.Load(path)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	_, core, scenario, err := save.LoadRulesData()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if unit, ok := f.State.Board.Unit(unitID); ok {
		writeJSON(w, deployedUnitOptions(f.State.Board, core, unit, scenario))
		return
	}
	for _, u := range f.State.Reserves[side] {
		if u.ID == unitID {
			writeJSON(w, reserveUnitOptions(f.State.Board, scenario, side))
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Errorf("unit %q not found on the board or in %s's reserves", unitID, side))
}

func deployedUnitOptions(board game.Board, core rules.CoreRules, unit game.UnitInstance, scenario rules.Scenario) OptionsView {
	view := OptionsView{
		CanReady:    canEnterState(unit, "ready"),
		CanMobilise: canEnterState(unit, "mobilised"),
		HasBarrage:  len(unit.Template.States) > 0 && unit.CanFire(),
	}

	stats := unit.Stats(core)
	if unit.CanMove() {
		view.MoveHexes = hexesWithin(unit.Pos, stats.Move, scenario)
		if unit.Template.Melee || len(unit.Template.States) == 0 {
			view.CloseMove = stats.Move / 2
			view.CloseMoveHexes = hexesWithin(unit.Pos, view.CloseMove, scenario)
		}
	}

	if unit.CanFire() && stats.MaxRange > 0 {
		for col := 0; col < scenario.Map.Columns; col++ {
			for row := 0; row < scenario.Map.Rows; row++ {
				h := hex.Offset{Col: col, Row: row}
				dist := hex.Distance(unit.Pos, h)
				if dist < stats.MinRange || dist > stats.MaxRange {
					continue
				}
				if !hex.InFiringArc(unit.Pos, unit.Facing, h) {
					continue
				}
				view.FireHexes = append(view.FireHexes, HexView{Col: col, Row: row})
			}
		}
	}

	for _, other := range board.Units {
		if other.Side == unit.Side {
			continue
		}
		if unit.Template.Melee {
			view.MeleeTargets = append(view.MeleeTargets, other.ID)
		}
		if len(unit.Template.States) == 0 {
			view.CloseFireTargets = append(view.CloseFireTargets, other.ID)
		}
		if unit.CanFire() {
			// Any enemy is a valid Fire target regardless of current
			// range/arc (docs/dev-plan.md section 7.6): the prediction
			// (GET /api/predict) says "out of range or arc" if it
			// actually is, once the order is being confirmed.
			view.FireTargets = append(view.FireTargets, other.ID)
		}
	}
	return view
}

// hexesWithin returns every hex on scenario's map within dist of pos
// (exclusive of pos itself), ignoring every unit on the board
// (docs/dev-plan.md section 7.6: a reach guide, not a pathfinding
// result — the map edge is the only limit).
func hexesWithin(pos hex.Offset, dist int, scenario rules.Scenario) []HexView {
	var hexes []HexView
	if dist <= 0 {
		return hexes
	}
	for col := 0; col < scenario.Map.Columns; col++ {
		for row := 0; row < scenario.Map.Rows; row++ {
			h := hex.Offset{Col: col, Row: row}
			d := hex.Distance(pos, h)
			if d > 0 && d <= dist {
				hexes = append(hexes, HexView{Col: col, Row: row})
			}
		}
	}
	return hexes
}

func canEnterState(unit game.UnitInstance, state string) bool {
	_, ok := unit.Template.States[state]
	return ok
}

func reserveUnitOptions(board game.Board, scenario rules.Scenario, side string) OptionsView {
	view := OptionsView{IsReserve: true}
	dz, ok := scenario.DeploymentZones[side]
	if !ok {
		return view
	}
	for row := dz.Rows.From - 1; row <= dz.Rows.To-1; row++ {
		for col := 0; col < scenario.Map.Columns; col++ {
			h := hex.Offset{Col: col, Row: row}
			if _, occupied := board.UnitAt(h); occupied {
				continue
			}
			if board.AdjacentEnemy(side, h) {
				continue
			}
			view.DeployHexes = append(view.DeployHexes, HexView{Col: h.Col, Row: h.Row})
		}
	}
	return view
}
