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

func mustParse(t *testing.T, s string) hex.Offset {
	t.Helper()
	h, err := hex.ParseOffset(s)
	if err != nil {
		t.Fatalf("hex.ParseOffset(%q): %v", s, err)
	}
	return h
}

func writeTestSave(t *testing.T, f save.File) string {
	t.Helper()
	if f.ScenarioID == "" {
		f.ScenarioID = save.DefaultScenarioID
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := save.Write(path, f); err != nil {
		t.Fatalf("save.Write: %v", err)
	}
	return path
}

func TestHandleOptionsDeployedUnit(t *testing.T) {
	chdirToRepoRoot(t)
	units, core, _, err := save.LoadRulesData(save.DefaultScenarioID)
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
	units, core, _, err := save.LoadRulesData(save.DefaultScenarioID)
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
	units, _, _, err := save.LoadRulesData(save.DefaultScenarioID)
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
	units, _, scenario, err := save.LoadRulesData(save.DefaultScenarioID)
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

// TestHandleOptionsBuyFromBank checks docs/core-rules.md section 10
// (dev-plan.md section 7.10): requesting options for a unit type
// (rather than an existing unit's ID) returns the same deploy options a
// reserve unit would, once it's turn 2 or later and the bank can
// afford it — and a 404 otherwise.
func TestHandleOptionsBuyFromBank(t *testing.T) {
	chdirToRepoRoot(t)
	_, _, scenario, err := save.LoadRulesData(save.DefaultScenarioID)
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}

	affordable := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		Turn:       2,
		State: game.GameState{
			Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows},
			Bank:  map[string]int{"north": 25},
		},
	})
	var got OptionsView
	getJSON(t, "/api/options?path="+affordable+"&side=north&unit=infantry", &got)
	if !got.IsReserve || len(got.DeployHexes) == 0 {
		t.Errorf("options for buyable infantry = %+v, want IsReserve with deploy hexes", got)
	}

	tooPoor := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		Turn:       2,
		State: game.GameState{
			Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows},
			Bank:  map[string]int{"north": 5}, // less than infantry's Cost 10.
		},
	})
	rec := httptest.NewRecorder()
	handleOptions(rec, httptest.NewRequest(http.MethodGet, "/api/options?path="+tooPoor+"&side=north&unit=infantry", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s, want %d (bank too low to buy)", rec.Code, rec.Body.String(), http.StatusNotFound)
	}

	beforeTurn2 := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		Turn:       1,
		State: game.GameState{
			Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows},
			Bank:  map[string]int{"north": 25},
		},
	})
	rec = httptest.NewRecorder()
	handleOptions(rec, httptest.NewRequest(http.MethodGet, "/api/options?path="+beforeTurn2+"&side=north&unit=infantry", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s, want %d (can't buy before turn 2)", rec.Code, rec.Body.String(), http.StatusNotFound)
	}
}

// TestHandleOptionsTwoTownsRoadMarchAndFord checks docs/dev-plan.md
// section 7.8, interface item 3, against the real Two Towns map: a
// unit on the road gets a road-march reach guide (Move+2, road hexes
// only), and a unit next to the unbridged river gets a ford hex.
func TestHandleOptionsTwoTownsRoadMarchAndFord(t *testing.T) {
	chdirToRepoRoot(t)
	units, _, scenario, err := save.LoadRulesData("two-towns")
	if err != nil {
		t.Fatalf("LoadRulesData(two-towns): %v", err)
	}
	onRoad := game.UnitInstance{ID: "A", Side: "north", Template: units["infantry"], Pos: mustParse(t, "K7"), Facing: hex.N, Strength: game.Full}
	onRiverBank := game.UnitInstance{ID: "B", Side: "north", Template: units["infantry"], Pos: mustParse(t, "K6"), Facing: hex.N, Strength: game.Full}
	path := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		State: game.GameState{
			Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows, Units: []game.UnitInstance{onRoad, onRiverBank}},
		},
	})

	var roadOpts OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=A", &roadOpts)
	foundQ5 := false
	for _, h := range roadOpts.RoadMarchHexes {
		if h.Col == mustParse(t, "Q5").Col && h.Row == mustParse(t, "Q5").Row {
			foundQ5 = true
		}
	}
	if !foundQ5 {
		t.Errorf("RoadMarchHexes = %+v, want Q5 included (within Move 4+2=6 along the road)", roadOpts.RoadMarchHexes)
	}

	var bankOpts OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=B", &bankOpts)
	foundL5 := false
	for _, h := range bankOpts.FordHexes {
		if h.Col == mustParse(t, "L5").Col && h.Row == mustParse(t, "L5").Row {
			foundL5 = true
		}
	}
	if !foundL5 {
		t.Errorf("FordHexes = %+v, want L5 included (directly across the unbridged K6|L5 river edge)", bankOpts.FordHexes)
	}
}

// TestHandleOptionsMoveHexesUsesForestCost checks docs/dev-plan.md
// section 7.10's "reach preview uses forest costs" against the real
// Two Towns map: a forest hex two steps in through forest (cost 2 each
// = 4) is excluded from infantry's Move-4 guide, even though it's only
// 2 hexes away, while an open-ground hex 4 steps away (cost 1 each) is
// included.
func TestHandleOptionsMoveHexesUsesForestCost(t *testing.T) {
	chdirToRepoRoot(t)
	units, _, scenario, err := save.LoadRulesData("two-towns")
	if err != nil {
		t.Fatalf("LoadRulesData(two-towns): %v", err)
	}
	mover := game.UnitInstance{ID: "A", Side: "north", Template: units["infantry"], Pos: mustParse(t, "K8"), Facing: hex.N, Strength: game.Full}
	path := writeTestSave(t, save.File{
		ScenarioID: scenario.ID,
		State:      game.GameState{Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows, Units: []game.UnitInstance{mover}}},
	})

	var got OptionsView
	getJSON(t, "/api/options?path="+path+"&side=north&unit=A", &got)

	has := func(name string) bool {
		want := mustParse(t, name)
		for _, h := range got.MoveHexes {
			if h.Col == want.Col && h.Row == want.Row {
				return true
			}
		}
		return false
	}
	// N9 is 3 forest hexes in (L8, M9, N9: cost 2+2+2=6 > Move 4):
	// unreachable despite being only distance 3.
	if has("N9") {
		t.Errorf("MoveHexes includes N9, want it excluded (costs 6 through forest, more than Move 4)")
	}
	// H5 is 4 open-ground hexes away (cost 1 each = 4): reachable.
	if !has("H5") {
		t.Errorf("MoveHexes excludes H5, want it included (4 open-ground hexes, cost 4, within Move 4)")
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
