package hex

import (
	"reflect"
	"testing"
)

func names(t *testing.T, ss ...string) []Offset {
	t.Helper()
	out := make([]Offset, len(ss))
	for i, s := range ss {
		out[i] = mustParse(t, s)
	}
	return out
}

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

func TestShortestPathStraightLine(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "A5")
	path, reached := ShortestPath(start, dest, boardPassable(), nil)
	if !reached {
		t.Fatalf("ShortestPath(A1, A5): want reached, got unreachable")
	}
	want := names(t, "A2", "A3", "A4", "A5")
	if !reflect.DeepEqual(path, want) {
		t.Errorf("ShortestPath(A1, A5) = %v, want %v", path, want)
	}
}

// TestShortestPathTieBreak checks that when several shortest paths exist,
// the first available direction in clockwise order from N is taken
// (docs/core-rules.md section 7.2). A1 is the top-left hex, so its N, NE
// and NW neighbours are off the board; the first real tie is between SE
// (toward B1) and the other directions, all of which are off-board here.
func TestShortestPathTieBreak(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "C1")
	path, reached := ShortestPath(start, dest, boardPassable(), nil)
	if !reached {
		t.Fatalf("ShortestPath(A1, C1): want reached, got unreachable")
	}
	if len(path) == 0 || path[0] != mustParse(t, "B1") {
		t.Errorf("ShortestPath(A1, C1) first step = %v, want B1", path)
	}
}

func TestShortestPathAroundObstacle(t *testing.T) {
	start, dest := mustParse(t, "A3"), mustParse(t, "C3")
	passable := boardPassable(mustParse(t, "B3"))

	path, reached := ShortestPath(start, dest, passable, nil)
	if !reached {
		t.Fatalf("ShortestPath around obstacle: want reached, got unreachable")
	}
	blocked := mustParse(t, "B3")
	for _, step := range path {
		if step == blocked {
			t.Fatalf("ShortestPath around obstacle: path %v steps through blocked hex %s", path, step)
		}
	}
	if got := path[len(path)-1]; got != dest {
		t.Errorf("ShortestPath around obstacle: last step = %s, want dest %s", got, dest)
	}
}

// TestShortestPathUnreachable checks the core-rules.md 7.2 fallback: when
// dest is occupied (impassable) and so can't be reached, head for the
// reachable hex closest to dest instead.
func TestShortestPathUnreachable(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "C1")
	passable := boardPassable(dest) // dest occupied by another unit.

	path, reached := ShortestPath(start, dest, passable, nil)
	if reached {
		t.Fatalf("ShortestPath to occupied dest: want unreachable, got reached with path %v", path)
	}
	// The closest reachable hex to C1 is B1: it's adjacent to C1 and is the
	// nearest such hex to start among C1's other neighbours (C2, D1).
	want := names(t, "B1")
	if !reflect.DeepEqual(path, want) {
		t.Errorf("ShortestPath to occupied dest: path = %v, want %v", path, want)
	}
}

func TestShortestPathAlreadyThere(t *testing.T) {
	start := mustParse(t, "A1")
	path, reached := ShortestPath(start, start, boardPassable(), nil)
	if !reached {
		t.Fatalf("ShortestPath(A1, A1): want reached")
	}
	if len(path) != 0 {
		t.Errorf("ShortestPath(A1, A1) = %v, want empty path", path)
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

// TestShortestPathRoutesAroundBlockedEdge checks that a blocked edge
// (docs/core-rules.md section 2.3: a river with no bridge) is avoided
// even though both hexes it joins are individually passable.
func TestShortestPathRoutesAroundBlockedEdge(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "A3")
	// A1-A2 is the only direct step toward A3; block it and the path
	// must detour, even though A2 is passable from elsewhere.
	edgeBlocked := boardEdgeBlocked(NewEdge(mustParse(t, "A1"), mustParse(t, "A2")))
	path, reached := ShortestPath(start, dest, boardPassable(), edgeBlocked)
	if !reached {
		t.Fatalf("ShortestPath(A1, A3) with A1-A2 blocked: want reached, got unreachable: %v", path)
	}
	if len(path) == 0 || path[0] == mustParse(t, "A2") {
		t.Errorf("ShortestPath(A1, A3) first step = %v, want anything but a direct A1->A2 step", path)
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
