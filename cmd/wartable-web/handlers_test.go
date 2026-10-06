package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

// chdirToRepoRoot changes the working directory to the repository root
// (two levels up from this package) for the test, since save.LoadRulesData
// reads data/ relative to the server's intended working directory.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(filepath.Join(wd, "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
}

func TestHandleGameMissingPath(t *testing.T) {
	rec := httptest.NewRecorder()
	handleGame(rec, httptest.NewRequest(http.MethodGet, "/api/game", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGameNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	handleGame(rec, httptest.NewRequest(http.MethodGet, "/api/game?path=/nonexistent/state.json", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGameSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	err := save.Write(path, save.File{
		ScenarioID: "starter-battle",
		Turn:       2,
		TieBreak:   game.TieBreak{Holder: "north"},
		State: game.GameState{
			Board: game.Board{Columns: 12, Rows: 10, Units: []game.UnitInstance{
				{ID: "A", Side: "north", Template: rules.Unit{ID: "infantry"}},
			}},
		},
	})
	if err != nil {
		t.Fatalf("save.Write: %v", err)
	}

	rec := httptest.NewRecorder()
	handleGame(rec, httptest.NewRequest(http.MethodGet, "/api/game?path="+path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got GameView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ScenarioID != "starter-battle" || got.Turn != 2 || len(got.Units) != 1 {
		t.Errorf("GameView = %+v", got)
	}
}

func TestHandleTurnDeploy(t *testing.T) {
	chdirToRepoRoot(t)

	units, _, scenario, err := save.LoadRulesData()
	if err != nil {
		t.Fatalf("LoadRulesData: %v", err)
	}
	reserves := map[string][]game.UnitInstance{
		"north": {{ID: "North 1st Infantry", Side: "north", Template: units["infantry"], Strength: game.Full}},
		"south": {{ID: "South 1st Infantry", Side: "south", Template: units["infantry"], Strength: game.Full}},
	}
	path := filepath.Join(t.TempDir(), "state.json")
	err = save.Write(path, save.File{
		ScenarioID: scenario.ID,
		Turn:       1,
		TieBreak:   game.TieBreak{Holder: scenario.TieBreakHolder},
		State:      game.GameState{Board: game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows}, Reserves: reserves},
	})
	if err != nil {
		t.Fatalf("save.Write: %v", err)
	}

	body, _ := json.Marshal(turnRequest{
		Path:  path,
		North: "North 1st Infantry | Deploy | B2",
		South: "South 1st Infantry | Deploy | B9",
	})
	rec := httptest.NewRecorder()
	handleTurn(rec, httptest.NewRequest(http.MethodPost, "/api/turn", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got turnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Events) != 2 {
		t.Fatalf("Events = %+v, want 2 deploy events", got.Events)
	}
	if len(got.Game.Units) != 2 {
		t.Fatalf("Game.Units = %+v, want 2 deployed units", got.Game.Units)
	}
	if got.Game.Turn != 2 {
		t.Errorf("Game.Turn = %d, want 2", got.Game.Turn)
	}

	// The save file on disk should reflect the executed turn too.
	reloaded, err := save.Load(path)
	if err != nil {
		t.Fatalf("save.Load: %v", err)
	}
	if reloaded.Turn != 2 || len(reloaded.State.Board.Units) != 2 {
		t.Errorf("reloaded save = %+v, want turn 2 with 2 units", reloaded)
	}
}

func TestHandleTurnBadOrders(t *testing.T) {
	chdirToRepoRoot(t)
	path := filepath.Join(t.TempDir(), "state.json")
	if err := save.Write(path, save.File{ScenarioID: "starter-battle", Turn: 1}); err != nil {
		t.Fatalf("save.Write: %v", err)
	}

	body, _ := json.Marshal(turnRequest{Path: path, North: "A | Charge | B2", South: ""})
	rec := httptest.NewRecorder()
	handleTurn(rec, httptest.NewRequest(http.MethodPost, "/api/turn", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s, want %d", rec.Code, rec.Body.String(), http.StatusBadRequest)
	}
}

func TestHandleTurnMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	handleTurn(rec, httptest.NewRequest(http.MethodGet, "/api/turn", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
