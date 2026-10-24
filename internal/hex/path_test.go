package hex

import (
	"testing"
)

// boardPassable simulates a finite board (columns/rows 0..9) with the given
// hexes blocked (e.g. occupied by a unit).
func boardPassable(blocked ...Offset) PassableFunc {
	blockedSet := make(map[Offset]bool, len(blocked))
	for _, b := range blocked {
		blockedSet[b] = true
	}
	return func(o Offset) bool {
		return o.Col >= 0 && o.Col < 10 && o.Row >= 0 && o.Row < 10 && !blockedSet[o]
	}
}

func TestFloodFillIncludesOrigin(t *testing.T) {
	origin := mustParse(t, "A1")
	field := FloodFill(origin, boardPassable(), nil)
	if d, ok := field[origin]; !ok || d != 0 {
		t.Errorf("FloodFill(A1) origin distance = %v, %v, want 0, true", d, ok)
	}
}

// boardEdgeBlocked blocks movement directly across the given edges
// (e.g. a river with no bridge), simulating docs/core-rules.md section
// 2.3, independently of whether either hex is itself passable.
func boardEdgeBlocked(edges ...Edge) EdgeBlockedFunc {
	set := make(map[Edge]bool, len(edges))
	for _, e := range edges {
		set[e] = true
	}
	return func(from, to Offset) bool {
		return set[NewEdge(from, to)]
	}
}

// TestFloodFillRespectsBlockedEdge checks the blocked hex is simply
// unreachable through that one edge when no detour exists either.
func TestFloodFillRespectsBlockedEdge(t *testing.T) {
	origin := mustParse(t, "E5")
	target := mustParse(t, "E6")
	edgeBlocked := boardEdgeBlocked(NewEdge(origin, target))
	// Also block every other neighbour of target so there's truly no detour.
	blockedHexes := target.Neighbors()
	passable := func(o Offset) bool {
		if o == target {
			return true
		}
		for _, b := range blockedHexes {
			if b != origin && o == b {
				return false
			}
		}
		return o.Col >= 0 && o.Col < 10 && o.Row >= 0 && o.Row < 10
	}
	field := FloodFill(origin, passable, edgeBlocked)
	if _, ok := field[target]; ok {
		t.Errorf("FloodFill reached %s across a blocked edge with no detour available", target)
	}
}
