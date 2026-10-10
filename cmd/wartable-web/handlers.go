package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

// handleGame serves GET /api/game?path=<state.json>: the current state of
// a saved game, for drawing the map (docs/dev-plan.md section 7, M6).
func handleGame(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing path"))
		return
	}
	f, err := save.Load(path)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	_, core, _, err := save.LoadRulesData()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, newGameView(f, core))
}

type turnRequest struct {
	Path  string `json:"path"`
	North string `json:"north"`
	South string `json:"south"`
}

type turnResponse struct {
	Game    GameView    `json:"game"`
	Events  []EventView `json:"events"`
	Outcome OutcomeView `json:"outcome"`
}

// handleTurn serves POST /api/turn: runs one turn (loading, executing,
// and saving, exactly like "wartable turn") and returns the resulting
// state plus the event log, for the viewer to step through one event at
// a time (docs/dev-plan.md section 7, M6).
func handleTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	var req turnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing path"))
		return
	}

	f, err := save.Load(req.Path)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	_, core, scenario, err := save.LoadRulesData()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	northOrders, err := orders.Parse(req.North)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("north orders: %w", err))
		return
	}
	southOrders, err := orders.Parse(req.South)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("south orders: %w", err))
		return
	}

	boardBefore := f.State.Board
	newState, events, err := game.ExecuteTurn(f.State, core, scenario, &f.TieBreak, northOrders, southOrders)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	f.State = newState
	f.RecordTurn(boardBefore, req.North, req.South, events)

	outcome := game.CheckAnnihilation(f.State)
	if !outcome.Over && f.Turn >= scenario.TurnLimit {
		_, _, limitOutcome := game.ScoreAtTurnLimit(f.State)
		outcome = limitOutcome
	}
	f.Turn++

	if err := save.Write(req.Path, f); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	eventViews := make([]EventView, len(events))
	for i, e := range events {
		eventViews[i] = newEventView(e, core)
	}
	writeJSON(w, turnResponse{
		Game:    newGameView(f, core),
		Events:  eventViews,
		Outcome: newOutcomeView(outcome),
	})
}

// PredictionView is a predicted result for an order not yet confirmed
// (docs/dev-plan.md section 7.5, interface item 2; section 7.7): the
// engine's own resolution, run against the current board but never
// saved, labelled as a prediction since other units may move first.
// Melee/Ranged are nil if the order wouldn't make contact this turn
// (e.g. a Close order that can't reach its target within half its
// Move). Path is set for a Move prediction: the safe route it would
// actually take this turn (docs/core-rules.md section 7.2), already
// truncated to the unit's Move.
type PredictionView struct {
	Melee  *MeleeView  `json:"melee,omitempty"`
	Ranged *RangedView `json:"ranged,omitempty"`
	Path   []HexView   `json:"path,omitempty"`
}

var predictableOrderTypes = map[string]orders.Type{
	"Move":             orders.Move,
	"Close and Attack": orders.CloseAndAttack,
	"Fire":             orders.Fire,
	"Close and Fire":   orders.CloseAndFire,
}

// handlePredict serves GET /api/predict?path=&side=&unit=&type=&target=
// (a unit target, for an attack/Fire order) or
// GET /api/predict?path=&side=&unit=&type=Move&col=&row= (a hex target,
// for a Move order): what would happen if unit's order were carried out
// this turn, assuming other units don't move first. It runs the real
// engine resolution (game.ExecuteOrder) against the saved state but
// never writes it back, so trying a prediction has no effect on the
// game.
func handlePredict(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	side := r.URL.Query().Get("side")
	unitID := r.URL.Query().Get("unit")
	orderType := r.URL.Query().Get("type")
	if path == "" || side == "" || unitID == "" || orderType == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing path, side, unit or type"))
		return
	}
	ot, ok := predictableOrderTypes[orderType]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported predict type %q", orderType))
		return
	}

	o := orders.Order{Unit: unitID, Type: ot}
	if ot == orders.Move {
		col, colErr := strconv.Atoi(r.URL.Query().Get("col"))
		row, rowErr := strconv.Atoi(r.URL.Query().Get("row"))
		if colErr != nil || rowErr != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("missing or invalid col/row for a Move prediction"))
			return
		}
		o.HasTargetHex = true
		o.TargetHex = hex.Offset{Col: col, Row: row}
	} else {
		targetID := r.URL.Query().Get("target")
		if targetID == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("missing target"))
			return
		}
		o.TargetUnit = targetID
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

	_, events := game.ExecuteOrder(f.State, core, scenario, side, o)

	var pred PredictionView
	for _, e := range events {
		if e.Melee != nil {
			pred.Melee = newMeleeView(e.Melee)
		}
		if e.Ranged != nil {
			pred.Ranged = newRangedView(e.Ranged)
		}
		if e.Kind == "moved" && pred.Path == nil {
			pred.Path = make([]HexView, len(e.Path))
			for i, h := range e.Path {
				pred.Path[i] = HexView{Col: h.Col, Row: h.Row}
			}
		}
	}
	writeJSON(w, pred)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
