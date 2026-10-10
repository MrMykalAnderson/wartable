package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenAnnihilation and TestGoldenTurnLimit replay a full Starter
// Battle game (docs/dev-plan.md section 7, M5) turn by turn through the
// CLI's own runNew/runTurn and check the combined output byte-for-byte
// against a recorded transcript. Each game was played by hand to exercise
// specific rules: see testdata/<scenario>/README.md.
// turnCount is how many of each scenario's turn order files to replay:
// annihilation ends early (the whole point of the scenario), so it has
// fewer turn files than the 12-turn limit.
func TestGoldenAnnihilation(t *testing.T) { testGolden(t, "annihilation", 11) }
func TestGoldenTurnLimit(t *testing.T)    { testGolden(t, "turnlimit", 12) }

func testGolden(t *testing.T, scenario string, turns int) {
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	testdataDir := filepath.Join(pkgDir, "testdata", scenario)
	goldenPath := filepath.Join(testdataDir, "golden.txt")
	repoRoot := filepath.Join(pkgDir, "..", "..")
	statePath := filepath.Join(t.TempDir(), "state.json")

	restore := chdir(t, repoRoot)
	defer restore()

	// runNew's own confirmation line echoes statePath, which is a fresh
	// temp path every run; it carries no game content, so it's excluded
	// from the golden comparison below (discarded here, not even printed).
	if err := runNew([]string{statePath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runNew: %v", err)
	}

	var out bytes.Buffer
	for n := 1; n <= turns; n++ {
		north := filepath.Join(testdataDir, fmt.Sprintf("north-%02d.txt", n))
		south := filepath.Join(testdataDir, fmt.Sprintf("south-%02d.txt", n))
		if err := runTurn([]string{statePath, north, south}, &out); err != nil {
			t.Fatalf("runTurn(turn %d): %v", n, err)
		}
	}

	// No man's land (docs/dev-plan.md section 7.7): a played-out game
	// should never trip the invariant that two enemy units are left
	// adjacent after an order. A golden replay tripping it would mean
	// the warning text got silently baked into golden.txt as if it were
	// expected, so check for it directly rather than relying on the
	// byte-for-byte comparison below to catch it.
	if strings.Contains(out.String(), "no man's land violation") {
		t.Errorf("golden replay for %s tripped the no man's land invariant: a [warning] event appears in its output", scenario)
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", goldenPath, err)
	}
	if out.String() != string(want) {
		gotPath := filepath.Join(os.TempDir(), "wartable-golden-got-"+scenario+".txt")
		os.WriteFile(gotPath, out.Bytes(), 0o644)
		t.Errorf("golden mismatch for %s: replayed output doesn't match %s (got written to %s)", scenario, goldenPath, gotPath)
	}
}

// chdir changes the working directory to dir, returning a function that
// restores the original. loadRulesData reads data/ relative to the CLI's
// intended working directory (the repository root), so tests that
// exercise it must chdir there first.
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%s): %v", dir, err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("Chdir(%s): %v", wd, err)
		}
	}
}
