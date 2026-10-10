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

// TestHandlePredictMoveFollowsSafeRoute checks docs/dev-plan.md section
// 7.7: a Move prediction's path is the real safe route the engine would
// take (EX-8's geometry), not a straight line.
func TestHandlePredictMoveFollowsSafeRoute(t *testing.T) {
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
		t.Fatalf("Path is empty, want the safe route")
	}
	last := got.Path[len(got.Path)-1]
	if last.Col != 4 || last.Row != 5 {
		t.Errorf("Path ends at %+v, want E6 (col 4, row 5): the safe route, stopped by Move 4", last)
	}
	for _, h := range got.Path {
		if h.Col == 5 && (h.Row == 4 || h.Row == 5) {
			t.Errorf("Path = %+v, includes F5/F6, both next to 1st Foot: want the safe route, not the plain shortest path", got.Path)
		}
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
