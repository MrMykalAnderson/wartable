package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/MrMykalAnderson/wartable/internal/save"
)

// SaveSummaryView is one save file's headline info, for the web viewer's
// save-management list.
type SaveSummaryView struct {
	Path       string `json:"path"`
	ScenarioID string `json:"scenarioId"`
	Turn       int    `json:"turn"`
	Units      int    `json:"units"`
}

var savePathPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*\.json$`)

// validSavePath reports whether path is a plain filename ending in
// .json, with no directory components: the web API only ever reads,
// writes or deletes save files directly inside the server's working
// directory, never anywhere else on disk.
func validSavePath(path string) bool {
	return savePathPattern.MatchString(path) && filepath.Base(path) == path
}

// handleSaves serves GET/POST/DELETE /api/saves: listing, creating and
// deleting save files, for the web viewer's save-management UI
// (docs/dev-plan.md web TODO: a "New game" button plus load/delete).
func handleSaves(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listSaves(w, r)
	case http.MethodPost:
		handleNewGame(w, r)
	case http.MethodDelete:
		handleDeleteSave(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("GET, POST or DELETE only"))
	}
}

// listSaves returns every *.json file in the working directory that
// actually loads as a save, sorted by path. Anything else (a stray JSON
// file, a corrupt save) is silently skipped rather than erroring the
// whole list.
func listSaves(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(".")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	saves := []SaveSummaryView{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		f, err := save.Load(e.Name())
		if err != nil {
			continue
		}
		saves = append(saves, SaveSummaryView{
			Path:       e.Name(),
			ScenarioID: f.ScenarioID,
			Turn:       f.Turn,
			Units:      len(f.State.Board.Units),
		})
	}
	sort.Slice(saves, func(i, j int) bool { return saves[i].Path < saves[j].Path })
	writeJSON(w, saves)
}

type newGameRequest struct {
	Path       string `json:"path"`
	ScenarioID string `json:"scenarioId"`
}

// handleNewGame creates a fresh save (save.NewGame: both sides seeded
// with the suggested army, turn 1) at a new path. It refuses to
// overwrite an existing file — delete it first if that's really what's
// wanted.
func handleNewGame(w http.ResponseWriter, r *http.Request) {
	var req newGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !validSavePath(req.Path) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid save name %q: use a plain filename ending in .json", req.Path))
		return
	}
	if _, err := os.Stat(req.Path); err == nil {
		writeError(w, http.StatusConflict, fmt.Errorf("%s already exists", req.Path))
		return
	}
	scenarioID := req.ScenarioID
	if scenarioID == "" {
		scenarioID = save.DefaultScenarioID
	}

	units, core, scenario, err := save.LoadRulesData(scenarioID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	f := save.NewGame(units, scenario)
	if err := save.Write(req.Path, f); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, newGameView(f, core, scenario))
}

// handleDeleteSave removes a save file. path must be a plain filename
// (validSavePath), so this can never reach outside the working
// directory.
func handleDeleteSave(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !validSavePath(path) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid save name %q", path))
		return
	}
	if err := os.Remove(path); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
