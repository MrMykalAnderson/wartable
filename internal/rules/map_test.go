package rules

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

const twoTownsMapPath = "../../data/maps/two-towns.yaml"

// TestLoadTerrainMapTwoTowns checks the real Two Towns map data loads
// and validates cleanly, matching docs/two-towns.md's description.
func TestLoadTerrainMapTwoTowns(t *testing.T) {
	m, err := LoadTerrainMap(twoTownsMapPath)
	if err != nil {
		t.Fatalf("LoadTerrainMap(%q): %v", twoTownsMapPath, err)
	}
	if m.ID != "two-towns" || m.Columns != 26 || m.Rows != 18 {
		t.Errorf("map header = %+v", m)
	}
	if len(m.Towns) != 2 {
		t.Errorf("Towns = %v, want 2 (North town, South town)", m.Towns)
	}
	if len(m.Towns["North town"]) != 6 {
		t.Errorf("North town = %v, want 6 hexes", m.Towns["North town"])
	}
	if len(m.Hamlets) != 2 {
		t.Errorf("Hamlets = %v, want 2 (West hamlet, East hamlet)", m.Hamlets)
	}
	if len(m.Bridge) != 2 {
		t.Errorf("Bridge = %v, want 2 bridges", m.Bridge)
	}
	if len(m.Forest) == 0 {
		t.Errorf("Forest is empty, want the forest block between the bridges")
	}
	if !m.RoadAt[mustParseHex(t, "K7")] {
		t.Errorf("K7 should be a road hex (EX-10 starts there)")
	}
	if !m.IsBridge(mustParseHex(t, "M7"), mustParseHex(t, "N6")) {
		t.Errorf("M7|N6 should be a bridge")
	}
	if !m.RiverBlocks(mustParseHex(t, "K6"), mustParseHex(t, "L5")) {
		t.Errorf("K6|L5 should be a blocked (non-bridge) river edge (EX-11)")
	}
}

func mustParseHex(t *testing.T, s string) hex.Offset {
	t.Helper()
	h, err := hex.ParseOffset(s)
	if err != nil {
		t.Fatalf("ParseOffset(%q): %v", s, err)
	}
	return h
}
