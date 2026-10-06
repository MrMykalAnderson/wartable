package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

func writeTestSave(t *testing.T, f save.File) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	if err := save.Write(path, f); err != nil {
		t.Fatalf("save.Write: %v", err)
	}
	return path
}

func TestHandleOptionsDeployedUnit(t *testing.T) {
	chdirToRepoRoot(t)
	units, core, _, err := save.LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}

	mover := game.UnitInstance{ID: "A", Side: "north", Template: units["cavalry"], Pos: hex.Offset{Col: 5, Row: 5}, Facing: hex.N, Strength: game.Full}
	enemy := game.UnitInstance{ID: "B", Side: "south", Template: units["infantry"], Pos: hex.Offset{Col: 5, Row: 1}, Facing: hex.S, Strength: game.Full}
	path := writeTestSave(t, save.File{
		State: game.GameState{Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{mover, enemy}}},
	})

	var got OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=A", &got)

	if got.IsReserve {
		t.Errorf("IsReserve = true, want false")
	}
	wantMove := mover.Stats(core).Move
	if len(got.MoveHexes) == 0 {
		t.Errorf("MoveHexes is empty, want reachable hexes within Move %d", wantMove)
	}
	for _, h := range got.MoveHexes {
		if d := hex.Distance(mover.Pos, hex.Offset{Col: h.Col, Row: h.Row}); d > wantMove {
			t.Errorf("MoveHexes contains %+v at distance %d, want <= %d", h, d, wantMove)
		}
	}
	if len(got.MeleeTargets) != 1 || got.MeleeTargets[0] != "B" {
		t.Errorf("MeleeTargets = %v, want [B] (cavalry can melee)", got.MeleeTargets)
	}
	if len(got.CloseFireTargets) != 1 || got.CloseFireTargets[0] != "B" {
		t.Errorf("CloseFireTargets = %v, want [B] (non-artillery)", got.CloseFireTargets)
	}
	// B is out of cavalry's Range (2) at distance 4, but any enemy is a
	// valid Fire target regardless of current range (docs/dev-plan.md
	// section 7.6): the prediction says "out of range" instead.
	if len(got.FireTargets) != 1 || got.FireTargets[0] != "B" {
		t.Errorf("FireTargets = %v, want [B] (any enemy is a valid Fire target)", got.FireTargets)
	}
}

// TestHandleOptionsMoveHexesIgnoreObstacles checks docs/dev-plan.md
// section 7.6: the reach highlight is pure distance, ignoring every
// unit on the board, since a Move order's path is only worked out at
// execution (by which time other units will have moved).
func TestHandleOptionsMoveHexesIgnoreObstacles(t *testing.T) {
	chdirToRepoRoot(t)
	units, core, _, err := save.LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}

	mover := game.UnitInstance{ID: "A", Side: "north", Template: units["infantry"], Pos: hex.Offset{Col: 5, Row: 5}, Facing: hex.N, Strength: game.Full}
	blockerPos := hex.Offset{Col: 5, Row: 4}
	blocker := game.UnitInstance{ID: "Blocker", Side: "north", Template: units["infantry"], Pos: blockerPos, Facing: hex.N, Strength: game.Full}
	path := writeTestSave(t, save.File{
		State: game.GameState{Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{mover, blocker}}},
	})

	var got OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=A", &got)

	wantMove := mover.Stats(core).Move
	wantCount := 0
	for col := 0; col < 12; col++ {
		for row := 0; row < 10; row++ {
			d := hex.Distance(mover.Pos, hex.Offset{Col: col, Row: row})
			if d > 0 && d <= wantMove {
				wantCount++
			}
		}
	}
	if len(got.MoveHexes) != wantCount {
		t.Errorf("MoveHexes has %d hexes, want %d (every hex within Move %d, ignoring Blocker)", len(got.MoveHexes), wantCount, wantMove)
	}
	foundBeyondBlocker := false
	for _, h := range got.MoveHexes {
		if h.Col == blockerPos.Col && h.Row == blockerPos.Row-1 {
			foundBeyondBlocker = true
		}
	}
	if !foundBeyondBlocker {
		t.Errorf("MoveHexes = %+v, want it to include the hex directly behind Blocker (obstacles are ignored)", got.MoveHexes)
	}
}

func TestHandleOptionsArtilleryExcludesMeleeAndCloseFire(t *testing.T) {
	chdirToRepoRoot(t)
	units, _, _, err := save.LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}
	gun := game.UnitInstance{ID: "Gun", Side: "north", Template: units["artillery"], Pos: hex.Offset{Col: 5, Row: 5}, Facing: hex.N, Strength: game.Full, State: "ready"}
	enemy := game.UnitInstance{ID: "B", Side: "south", Template: units["infantry"], Pos: hex.Offset{Col: 5, Row: 2}, Facing: hex.S, Strength: game.Full}
	path := writeTestSave(t, save.File{
		State: game.GameState{Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{gun, enemy}}},
	})

	var got OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=Gun", &got)

	if len(got.MeleeTargets) != 0 {
		t.Errorf("MeleeTargets = %v, want none (artillery can't melee)", got.MeleeTargets)
	}
	if len(got.CloseFireTargets) != 0 {
		t.Errorf("CloseFireTargets = %v, want none (artillery can't Close and Fire)", got.CloseFireTargets)
	}
	if len(got.MoveHexes) != 0 {
		t.Errorf("MoveHexes = %v, want none (Ready artillery can't move)", got.MoveHexes)
	}
	if len(got.FireTargets) != 1 || got.FireTargets[0] != "B" {
		t.Errorf("FireTargets = %v, want [B] (in range and arc)", got.FireTargets)
	}
	// Both orders are valid regardless of current state: the engine
	// allows re-issuing Ready while already ready (e.g. just to set a
	// new facing).
	if !got.CanMobilise || !got.CanReady {
		t.Errorf("CanMobilise/CanReady = %v/%v, want true/true", got.CanMobilise, got.CanReady)
	}
}

func TestHandleOptionsReserveUnit(t *testing.T) {
	chdirToRepoRoot(t)
	units, _, scenario, err := save.LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}
	blocker := game.UnitInstance{ID: "Blocker", Side: "north", Template: units["infantry"], Pos: hex.Offset{Col: 0, Row: 0}, Facing: hex.S, Strength: game.Full}
	path := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		State: game.GameState{
			Board:    game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows, Units: []game.UnitInstance{blocker}},
			Reserves: map[string][]game.UnitInstance{"north": {{ID: "Reserve", Side: "north", Template: units["infantry"]}}},
		},
	})

	var got OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=Reserve", &got)

	if !got.IsReserve {
		t.Errorf("IsReserve = false, want true")
	}
	if len(got.DeployHexes) == 0 {
		t.Errorf("DeployHexes is empty, want north's deployment zone minus A1")
	}
	for _, h := range got.DeployHexes {
		if h.Col == 0 && h.Row == 0 {
			t.Errorf("DeployHexes includes A1, which is occupied by Blocker")
		}
	}
}

func TestHandleOptionsUnknownUnit(t *testing.T) {
	chdirToRepoRoot(t)
	path := writeTestSave(t, save.File{State: game.GameState{Board: game.Board{Columns: 12, Rows: 10}}})

	rec := httptest.NewRecorder()
	handleOptions(rec, httptest.NewRequest(http.MethodGet, "/api/options?path="+path+"&side=north&unit=Nobody", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s, want %d", rec.Code, rec.Body.String(), http.StatusNotFound)
	}
}

func TestHandleOptionsMissingParams(t *testing.T) {
	rec := httptest.NewRecorder()
	handleOptions(rec, httptest.NewRequest(http.MethodGet, "/api/options", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// getJSON calls handleOptions directly and decodes a successful response.
func getJSON(t *testing.T, target string, v any) {
	t.Helper()
	rec := httptest.NewRecorder()
	handleOptions(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, body = %s", target, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
