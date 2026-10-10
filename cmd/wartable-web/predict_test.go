package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

// TestHandlePredictMoveIgnoresUnits checks docs/dev-plan.md section 7.9:
// a Move prediction's path is a terrain-only planning estimate that
// ignores every unit, friendly or enemy — unlike actually running the
// order (which takes the safe route, EX-8's geometry — see
// internal/game's TestSafeRouteEX8, unaffected by this change), the
// preview goes straight down the column even past an enemy at G6, since
// other units may move before the order actually runs.
func TestHandlePredictMoveIgnoresUnits(t *testing.T) {
	chdirToRepoRoot(t)
	units, _, _, err := save.LoadRulesData(save.DefaultScenarioID)
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}
	mover := game.UnitInstance{ID: "2nd Foot", Side: "south", Template: units["infantry"], Pos: hex.Offset{Col: 5, Row: 8}, Facing: hex.N, Strength: game.Full}
	enemy := game.UnitInstance{ID: "1st Foot", Side: "north", Template: units["infantry"], Pos: hex.Offset{Col: 6, Row: 5}, Facing: hex.S, Strength: game.Full}
	path := writeTestSave(t, save.File{
		State: game.GameState{Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{mover, enemy}}},
	})

	var got PredictionView
	rec := httptest.NewRecorder()
	handlePredict(rec, httptest.NewRequest(http.MethodGet, "/api/predict?path="+path+"&side=south&unit="+url("2nd Foot")+"&type=Move&col=5&row=3", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got.Path) == 0 {
		t.Fatalf("Path is empty, want the plain terrain route")
	}
	last := got.Path[len(got.Path)-1]
	if last.Col != 5 || last.Row != 4 {
		t.Errorf("Path ends at %+v, want F5 (col 5, row 4): straight down the column, stopped by Move 4, ignoring 1st Foot", last)
	}
}

func TestHandlePredictMoveMissingColRow(t *testing.T) {
	chdirToRepoRoot(t)
	path := writeTestSave(t, save.File{State: game.GameState{Board: game.Board{Columns: 12, Rows: 10}}})
	rec := httptest.NewRecorder()
	handlePredict(rec, httptest.NewRequest(http.MethodGet, "/api/predict?path="+path+"&side=north&unit=A&type=Move", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePredictUnsupportedType(t *testing.T) {
	rec := httptest.NewRecorder()
	handlePredict(rec, httptest.NewRequest(http.MethodGet, "/api/predict?path=x&side=north&unit=A&type=Ready&target=B", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func url(s string) string {
	// Minimal query-escape for the space in a unit ID like "2nd Foot";
	// avoids importing net/url just for this one call.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			out = append(out, '%', '2', '0')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
