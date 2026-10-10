package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

const (
	unitsPath        = "../../data/units/standard.yaml"
	coreRulesPath    = "../../data/rules/core.yaml"
	terrainRulesPath = "../../data/rules/terrain.yaml"
	scenarioPath     = "../../data/scenarios/starter-battle.yaml"
)

func loadTestScenario(t *testing.T, templates map[string]rules.Unit) rules.Scenario {
	t.Helper()
	s, err := rules.LoadScenario(scenarioPath, templates)
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	return s
}

// loadTestTwoTowns loads the real Two Towns scenario and map
// (data/scenarios/two-towns.yaml, data/maps/two-towns.yaml), for tests
// exercising terrain rules against the actual data (e.g. EX-10, EX-11).
// map_file inside the scenario is repo-root-relative, so this chdirs
// there for the call, same as internal/rules' own tests.
func loadTestTwoTowns(t *testing.T, templates map[string]rules.Unit) rules.Scenario {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(filepath.Join(wd, "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(wd)

	s, err := rules.LoadScenario("data/scenarios/two-towns.yaml", templates)
	if err != nil {
		t.Fatalf("LoadScenario(two-towns): %v", err)
	}
	return s
}

// reserveUnit builds a full-strength UnitInstance not yet on the map, for
// GameState.Reserves.
func reserveUnit(t *testing.T, templates map[string]rules.Unit, id, side, templateID string) UnitInstance {
	t.Helper()
	tmpl, ok := templates[templateID]
	if !ok {
		t.Fatalf("reserveUnit: unknown template %q", templateID)
	}
	return UnitInstance{ID: id, Side: side, Template: tmpl, Strength: Full, State: tmpl.DeployState}
}

func loadTestRules(t *testing.T) (map[string]rules.Unit, rules.CoreRules) {
	t.Helper()
	units, err := rules.LoadUnits(unitsPath)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	core, err := rules.LoadCoreRules(coreRulesPath)
	if err != nil {
		t.Fatalf("LoadCoreRules: %v", err)
	}
	core.Terrain, err = rules.LoadTerrainEffects(terrainRulesPath)
	if err != nil {
		t.Fatalf("LoadTerrainEffects: %v", err)
	}
	return units, core
}

// newUnit builds a full-strength UnitInstance at the hex named pos (e.g.
// "F6"), facing dir. Artillery starts in its deploy_state; override
// u.State for a ready gun.
func newUnit(t *testing.T, templates map[string]rules.Unit, id, side, templateID, pos string, facing hex.Direction) UnitInstance {
	t.Helper()
	tmpl, ok := templates[templateID]
	if !ok {
		t.Fatalf("newUnit: unknown template %q", templateID)
	}
	p, err := hex.ParseOffset(pos)
	if err != nil {
		t.Fatalf("newUnit: %v", err)
	}
	return UnitInstance{
		ID:       id,
		Side:     side,
		Template: tmpl,
		Pos:      p,
		Facing:   facing,
		Strength: Full,
		State:    tmpl.DeployState,
	}
}

// starterBoard builds a Board the size of the Starter Battle map
// (docs/starter-battle.md) containing the given units.
func starterBoard(units ...UnitInstance) Board {
	return Board{Columns: 12, Rows: 10, Units: units}
}

// boardFor builds a Board the size of scenario's map, with its terrain
// (if any) attached, containing the given units.
func boardFor(scenario rules.Scenario, units ...UnitInstance) Board {
	return Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows, Units: units, Terrain: scenario.Terrain}
}
