package rules

import "testing"

const terrainEffectsPath = "../../data/rules/terrain.yaml"

// TestLoadTerrainEffectsMatchesDocs checks the loaded numbers against
// docs/core-rules.md section 2.3.
func TestLoadTerrainEffectsMatchesDocs(t *testing.T) {
	e, err := LoadTerrainEffects(terrainEffectsPath)
	if err != nil {
		t.Fatalf("LoadTerrainEffects(%q): %v", terrainEffectsPath, err)
	}
	want := TerrainEffects{
		ForestDefBonus: 1, TownDefBonus: 1, HamletDefBonus: 1,
		CavalryForestPenalty: 1, BridgeAttackPenalty: 1, RoadMarchMoveBonus: 2,
		TownPoints: 20, HamletPoints: 10,
	}
	if e != want {
		t.Errorf("LoadTerrainEffects = %+v, want %+v", e, want)
	}
}

func TestLoadTerrainMapRejectsNonAdjacentRiverEdge(t *testing.T) {
	path := writeTempFile(t, "map-*.yaml", `
id: bad
name: Bad
columns: 5
rows: 5
terrain_key: { ".": open }
terrain:
- '.....'
- '.....'
- '.....'
- '.....'
- '.....'
river:
- A1|C3
`)
	if _, err := LoadTerrainMap(path); err == nil {
		t.Fatalf("LoadTerrainMap: want error for a non-adjacent river edge, got none")
	}
}

func TestLoadTerrainMapRejectsBridgeWithoutRoad(t *testing.T) {
	path := writeTempFile(t, "map-*.yaml", `
id: bad
name: Bad
columns: 5
rows: 5
terrain_key: { ".": open }
terrain:
- '.....'
- '.....'
- '.....'
- '.....'
- '.....'
river:
- A1|A2
bridges:
- A1|A2
`)
	if _, err := LoadTerrainMap(path); err == nil {
		t.Fatalf("LoadTerrainMap: want error for a bridge with no road, got none")
	}
}

func TestLoadTerrainMapRejectsBridgeNotOnRiver(t *testing.T) {
	path := writeTempFile(t, "map-*.yaml", `
id: bad
name: Bad
columns: 5
rows: 5
terrain_key: { ".": open }
terrain:
- '.....'
- '.....'
- '.....'
- '.....'
- '.....'
roads:
  R1: [A1, A2]
bridges:
- A1|A2
`)
	if _, err := LoadTerrainMap(path); err == nil {
		t.Fatalf("LoadTerrainMap: want error for a bridge that isn't a river edge, got none")
	}
}

func TestLoadTerrainMapRejectsDiscontiguousRoad(t *testing.T) {
	path := writeTempFile(t, "map-*.yaml", `
id: bad
name: Bad
columns: 5
rows: 5
terrain_key: { ".": open }
terrain:
- '.....'
- '.....'
- '.....'
- '.....'
- '.....'
roads:
  R1: [A1, C3]
`)
	if _, err := LoadTerrainMap(path); err == nil {
		t.Fatalf("LoadTerrainMap: want error for a discontiguous road, got none")
	}
}

// TestLoadTerrainMapRoadClearsForest checks docs/core-rules.md section
// 2.3: "a road hex is never forest: roads are cleared through it."
func TestLoadTerrainMapRoadClearsForest(t *testing.T) {
	path := writeTempFile(t, "map-*.yaml", `
id: ok
name: OK
columns: 5
rows: 5
terrain_key: { ".": open, "F": forest }
terrain:
- '.....'
- '.FFF.'
- '.....'
- '.....'
- '.....'
roads:
  R1: [B2, C2, D2]
`)
	m, err := LoadTerrainMap(path)
	if err != nil {
		t.Fatalf("LoadTerrainMap: %v", err)
	}
	if m.IsForest(mustParseHex(t, "C2")) {
		t.Errorf("C2 is on R1 and marked forest in the grid: want IsForest false (roads clear forest)")
	}
}
