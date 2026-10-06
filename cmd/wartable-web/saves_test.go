package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidSavePath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"state.json", true},
		{"my-game_2.json", true},
		{"", false},
		{"../state.json", false},
		{"sub/state.json", false},
		{"/etc/passwd.json", false},
		{"state.txt", false},
		{".json", false},
	}
	for _, c := range cases {
		if got := validSavePath(c.path); got != c.want {
			t.Errorf("validSavePath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// chdirToTempWithData copies data/ into a fresh temp directory and
// chdirs there for the test: handleNewGame needs data/ (relative to the
// working directory, like the real server), and the save-management
// handlers operate on *.json files in the working directory, which must
// be a safe scratch space, not the repository root itself.
func chdirToTempWithData(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Join(wd, "..", "..")
	dir := t.TempDir()
	if err := copyDir(filepath.Join(repoRoot, "data"), filepath.Join(dir, "data")); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func TestListSavesEmpty(t *testing.T) {
	chdirToTempWithData(t)
	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodGet, "/api/saves", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got []SaveSummaryView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("listSaves = %+v, want empty", got)
	}
}

func TestHandleNewGameThenList(t *testing.T) {
	chdirToTempWithData(t)

	body, _ := json.Marshal(newGameRequest{Path: "my-game.json"})
	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodPost, "/api/saves", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var created GameView
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if created.ScenarioID != "starter-battle" || created.Turn != 1 {
		t.Errorf("created game = %+v", created)
	}
	if len(created.Reserves["north"]) != 4 || len(created.Reserves["south"]) != 4 {
		t.Errorf("created game reserves = %+v, want 4 units per side", created.Reserves)
	}

	rec = httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodGet, "/api/saves", nil))
	var list []SaveSummaryView
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(list) != 1 || list[0].Path != "my-game.json" || list[0].Turn != 1 {
		t.Errorf("listSaves after create = %+v", list)
	}
}

func TestHandleNewGameRefusesOverwrite(t *testing.T) {
	chdirToTempWithData(t)
	if err := os.WriteFile("existing.json", []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	body, _ := json.Marshal(newGameRequest{Path: "existing.json"})
	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodPost, "/api/saves", bytes.NewReader(body)))
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (should refuse to overwrite)", rec.Code, http.StatusConflict)
	}
}

func TestHandleNewGameRejectsBadPath(t *testing.T) {
	chdirToTempWithData(t)
	body, _ := json.Marshal(newGameRequest{Path: "../escape.json"})
	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodPost, "/api/saves", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteSave(t *testing.T) {
	chdirToTempWithData(t)
	if err := os.WriteFile("gone.json", []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodDelete, "/api/saves?path=gone.json", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat("gone.json"); !os.IsNotExist(err) {
		t.Errorf("gone.json still exists after delete")
	}
}

func TestHandleDeleteSaveRejectsBadPath(t *testing.T) {
	chdirToTempWithData(t)
	rec := httptest.NewRecorder()
	handleSaves(rec, httptest.NewRequest(http.MethodDelete, "/api/saves?path="+"../../etc/passwd.json", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
