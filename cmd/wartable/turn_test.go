package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/save"
)

// TestRunTurnRecordsHistory checks docs/dev-plan.md section 7.7: running
// a turn through the CLI persists its order sheets and event log to the
// save file, not just the resulting position.
func TestRunTurnRecordsHistory(t *testing.T) {
	restore := chdir(t, filepath.Join(mustGetwd(t), "..", ".."))
	defer restore()

	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := runNew([]string{statePath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runNew: %v", err)
	}

	northPath := filepath.Join(t.TempDir(), "north.txt")
	southPath := filepath.Join(t.TempDir(), "south.txt")
	if err := os.WriteFile(northPath, []byte("North 1st Infantry | Deploy | B2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(southPath, []byte("South 1st Infantry | Deploy | B9\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := runTurn([]string{statePath, northPath, southPath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runTurn: %v", err)
	}

	f, err := save.Load(statePath)
	if err != nil {
		t.Fatalf("save.Load: %v", err)
	}
	if len(f.History) != 1 {
		t.Fatalf("History = %+v, want 1 record", f.History)
	}
	if f.History[0].Turn != 1 {
		t.Errorf("History[0].Turn = %d, want 1", f.History[0].Turn)
	}
	if f.History[0].NorthOrders != "North 1st Infantry | Deploy | B2\n" {
		t.Errorf("History[0].NorthOrders = %q", f.History[0].NorthOrders)
	}
	if len(f.History[0].Events) != 2 {
		t.Errorf("History[0].Events = %+v, want 2 deploy events", f.History[0].Events)
	}
	if len(f.History[0].BoardBefore.Units) != 0 {
		t.Errorf("History[0].BoardBefore = %+v, want no units yet", f.History[0].BoardBefore)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return wd
}
