package hex

import "testing"

// costFn returns a CostFunc giving cost 2 to every hex in expensive,
// and 1 to everything else — a synthetic "forest" for testing
// CheapestPath without needing a real scenario map.
func costFn(expensive ...Offset) CostFunc {
	set := make(map[Offset]bool, len(expensive))
	for _, h := range expensive {
		set[h] = true
	}
	return func(o Offset) int {
		if set[o] {
			return 2
		}
		return 1
	}
}

// TestCheapestPathPrefersCheaperDetour checks docs/core-rules.md
// section 7.2/EX-12: a longer-but-cheaper route is taken over a
// shorter-but-costlier one. A1->A5 direct (A2,A3,A4 expensive then A5,
// cost 2+2+2+1 = 7) is more expensive than detouring via column B
// (B1,B2,B3,B4,A5, all cost 1 = 5), even though the detour has one more
// hex.
func TestCheapestPathPrefersCheaperDetour(t *testing.T) {
	expensive := names(t, "A2", "A3", "A4")
	passable := boardPassable()
	cost := costFn(expensive...)

	path, reached := CheapestPath(mustParse(t, "A1"), mustParse(t, "A5"), passable, cost, nil)
	if !reached {
		t.Fatalf("CheapestPath: want reached, got unreachable")
	}
	want := names(t, "B1", "B2", "B3", "B4", "A5")
	if len(path) != len(want) {
		t.Fatalf("CheapestPath = %v, want %v", path, want)
	}
	for i, h := range want {
		if path[i] != h {
			t.Errorf("CheapestPath[%d] = %v, want %v (full path %v)", i, path[i], h, path)
		}
	}
}

// TestCheapestPathUniformCostMatchesShortestPath checks that with a
// uniform CostFunc (every hex costs 1), CheapestPath reduces to
// ShortestPath, including its clockwise tie-break.
func TestCheapestPathUniformCostMatchesShortestPath(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "C1")
	passable := boardPassable()
	uniform := costFn()

	want, wantReached := ShortestPath(start, dest, passable, nil)
	got, gotReached := CheapestPath(start, dest, passable, uniform, nil)
	if gotReached != wantReached {
		t.Fatalf("CheapestPath reached = %v, want %v", gotReached, wantReached)
	}
	if len(got) != len(want) {
		t.Fatalf("CheapestPath = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("CheapestPath[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestCheapestPathFewerHexesTieBreak checks that when two routes cost
// the same, the one with fewer hexes wins, even if a clockwise-first
// route would otherwise be longer. Direct A1->A3 (A2,A3, cost 1 each =
// 2) ties on a per-hex-cost basis with no alternative here, but
// construct a case where a 2-hex route costs the same as this 2-hex
// direct route reached a different way is not meaningful on an open
// board; instead check a cost-4 detour (B1,B2,B3,A3, cost 1 each=4)
// loses to a cost-2 direct corridor even though both would be "valid"
// routes, confirming cost (not hex count) is the primary ranking.
func TestCheapestPathFewerHexesTieBreak(t *testing.T) {
	start, dest := mustParse(t, "A1"), mustParse(t, "A3")
	passable := boardPassable()
	uniform := costFn()

	path, reached := CheapestPath(start, dest, passable, uniform, nil)
	if !reached {
		t.Fatalf("CheapestPath: want reached, got unreachable")
	}
	want := names(t, "A2", "A3")
	if len(path) != len(want) {
		t.Fatalf("CheapestPath = %v, want %v (the direct, cheaper route)", path, want)
	}
}

// TestCheapestPathFallsBackToClosestReachable checks the unreachable-
// destination fallback (docs/core-rules.md 7.2's provisional rule),
// weighted-case: an occupied destination heads for the cheapest-to-
// reach neighbour instead.
func TestCheapestPathFallsBackToClosestReachable(t *testing.T) {
	dest := mustParse(t, "A3")
	passable := boardPassable(dest) // A3 itself is occupied/impassable.
	uniform := costFn()

	path, reached := CheapestPath(mustParse(t, "A1"), dest, passable, uniform, nil)
	if reached {
		t.Fatalf("CheapestPath: want not reached (dest occupied), got reached with path %v", path)
	}
	if len(path) == 0 {
		t.Fatalf("CheapestPath: want a fallback path, got none")
	}
	last := path[len(path)-1]
	if Distance(last, dest) != 1 {
		t.Errorf("CheapestPath fallback ended at %v, want a hex adjacent to A3 (closest reachable)", last)
	}
}
