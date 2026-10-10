package rules

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const starterBattlePath = "../../data/scenarios/starter-battle.yaml"

// TestLoadScenarioMatchesDocs checks the loaded Starter Battle settings
// against docs/starter-battle.md "At a glance".
func TestLoadScenarioMatchesDocs(t *testing.T) {
	units, err := LoadUnits(unitsPath)
	if err != nil {
		t.Fatalf("LoadUnits(%q): %v", unitsPath, err)
	}
	s, err := LoadScenario(starterBattlePath, units)
	if err != nil {
		t.Fatalf("LoadScenario(%q): %v", starterBattlePath, err)
	}

	if s.Map != (MapSize{Columns: 12, Rows: 10}) {
		t.Errorf("Map = %+v, want {Columns:12 Rows:10}", s.Map)
	}
	if s.Capacity != 60 {
		t.Errorf("Capacity = %d, want 60", s.Capacity)
	}
	if s.TurnLimit != 12 {
		t.Errorf("TurnLimit = %d, want 12", s.TurnLimit)
	}
	if s.TieBreakHolder != "north" {
		t.Errorf("TieBreakHolder = %q, want %q", s.TieBreakHolder, "north")
	}

	north, ok := s.DeploymentZones["north"]
	if !ok || north.Rows == nil || *north.Rows != (RowRange{From: 1, To: 2}) {
		t.Errorf("DeploymentZones[north] = %+v, %v, want {Rows:{1 2}}, true", north, ok)
	}
	south, ok := s.DeploymentZones["south"]
	if !ok || south.Rows == nil || *south.Rows != (RowRange{From: 9, To: 10}) {
		t.Errorf("DeploymentZones[south] = %+v, %v, want {Rows:{9 10}}, true", south, ok)
	}

	if s.DefaultFacing["north"] != "S" || s.DefaultFacing["south"] != "N" {
		t.Errorf("DefaultFacing = %+v, want {north:S south:N}", s.DefaultFacing)
	}

	wantUnits := []string{"infantry", "cavalry", "artillery"}
	gotUnits := append([]string(nil), s.UnitsAllowed...)
	if !reflect.DeepEqual(gotUnits, wantUnits) {
		t.Errorf("UnitsAllowed = %v, want %v", gotUnits, wantUnits)
	}
}

// TestLoadScenarioTwoTowns checks the Two Towns scenario loads its
// map_file and resolves its radius-from-town deployment zones, matching
// docs/two-towns.md "At a glance" (44 hexes per side's zone).
func TestLoadScenarioTwoTowns(t *testing.T) {
	// map_file inside a scenario file is repo-root-relative (matching
	// how the CLI and web server are run), so this test must chdir
	// there, same as internal/save's TestLoadRulesData.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(filepath.Join(wd, "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(wd)

	units, err := LoadUnits("data/units/standard.yaml")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	s, err := LoadScenario("data/scenarios/two-towns.yaml", units)
	if err != nil {
		t.Fatalf("LoadScenario(data/scenarios/two-towns.yaml): %v", err)
	}
	if s.Terrain == nil {
		t.Fatalf("Terrain is nil, want it loaded from map_file")
	}
	if s.Map != (MapSize{Columns: 26, Rows: 18}) {
		t.Errorf("Map = %+v, want 26x18 (from the map file)", s.Map)
	}
	if s.Capacity != 120 || s.TurnLimit != 16 {
		t.Errorf("Capacity/TurnLimit = %d/%d, want 120/16", s.Capacity, s.TurnLimit)
	}

	north := s.DeploymentZones["north"]
	hexes := north.Hexes(s.Map, s.Terrain)
	if len(hexes) != 44 {
		t.Errorf("north deployment zone has %d hexes, want 44", len(hexes))
	}
	if !north.Contains(mustParseHex(t, "B15"), s.Terrain) {
		t.Errorf("north deployment zone should contain B15 (in North town itself)")
	}
}

func TestLoadScenarioRejectsUnknownUnit(t *testing.T) {
	units := map[string]Unit{} // no units defined
	path := writeTempFile(t, "scenario-*.yaml", `
id: test
name: Test
map: { columns: 12, rows: 10 }
deployment_zones:
  north: { rows: [1, 2] }
default_facing: { north: S }
capacity: 60
units_allowed: [infantry]
turn_limit: 12
tie_break_holder: north
`)
	if _, err := LoadScenario(path, units); err == nil {
		t.Fatalf("LoadScenario: want error for unknown unit, got none")
	}
}

func TestLoadScenarioRejectsZoneOutsideMap(t *testing.T) {
	units := map[string]Unit{"infantry": {ID: "infantry", Name: "Infantry"}}
	path := writeTempFile(t, "scenario-*.yaml", `
id: test
name: Test
map: { columns: 12, rows: 10 }
deployment_zones:
  north: { rows: [1, 11] }
default_facing: { north: S }
capacity: 60
units_allowed: [infantry]
turn_limit: 12
tie_break_holder: north
`)
	if _, err := LoadScenario(path, units); err == nil {
		t.Fatalf("LoadScenario: want error for deployment zone outside map, got none")
	}
}
