package hex

// PassableFunc reports whether a hex may be entered. Callers should treat a
// moving unit's own current hex as passable: it blocks nothing for the unit
// that is already standing on it.
type PassableFunc func(Offset) bool

// EdgeBlockedFunc reports whether moving directly between two adjacent
// hexes is blocked, independently of whether either hex is itself
// passable (docs/core-rules.md section 2.3: a river edge with no
// bridge). nil means no edge is ever blocked.
type EdgeBlockedFunc func(from, to Offset) bool

func blocked(f EdgeBlockedFunc, from, to Offset) bool {
	return f != nil && f(from, to)
}

// FloodFill returns the step-distance from origin to every hex reachable
// through passable hexes and unblocked edges, including origin itself
// (distance 0).
func FloodFill(origin Offset, passable PassableFunc, edgeBlocked EdgeBlockedFunc) map[Offset]int {
	dist := map[Offset]int{origin: 0}
	queue := []Offset{origin}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range Directions {
			n := cur.Neighbor(d)
			if _, seen := dist[n]; seen {
				continue
			}
			if !passable(n) {
				continue
			}
			if blocked(edgeBlocked, cur, n) {
				continue
			}
			dist[n] = dist[cur] + 1
			queue = append(queue, n)
		}
	}
	return dist
}

// NextStep returns the neighbour of cur that is one step closer to the
// origin of field (a distance field produced by FloodFill with the same
// edgeBlocked function), breaking ties by taking the first available
// direction in clockwise order from N (docs/dev-plan.md section 5). ok
// is false if cur is the origin, or isn't in field.
func NextStep(cur Offset, field map[Offset]int, edgeBlocked EdgeBlockedFunc) (next Offset, ok bool) {
	curDist, inField := field[cur]
	if !inField || curDist == 0 {
		return Offset{}, false
	}
	for _, d := range Directions {
		n := cur.Neighbor(d)
		if nd, inField := field[n]; inField && nd == curDist-1 && !blocked(edgeBlocked, cur, n) {
			return n, true
		}
	}
	return Offset{}, false
}

// ShortestPath finds the steps from start to dest through passable hexes
// and unblocked edges (docs/core-rules.md section 7.2), taking the first
// available direction in clockwise order at every tie. dest itself must
// be passable to be reached; an impassable dest (e.g. occupied) always
// falls through to the heading-for behaviour below. If dest can't be
// reached, the path instead heads for the reachable hex closest to dest,
// breaking ties by fewest steps from start and then by reading order
// (lowest column, then lowest row). The returned path excludes start and
// includes the hex actually headed for; reached reports whether that hex
// is dest.
func ShortestPath(start, dest Offset, passable PassableFunc, edgeBlocked EdgeBlockedFunc) (path []Offset, reached bool) {
	distToDest := FloodFill(dest, passable, edgeBlocked)
	if passable(dest) {
		if _, ok := distToDest[start]; ok {
			return walk(start, distToDest, edgeBlocked), true
		}
	}

	distFromStart := FloodFill(start, passable, edgeBlocked)
	target := closestReachable(distFromStart, distToDest)
	return walk(start, FloodFill(target, passable, edgeBlocked), edgeBlocked), false
}

func walk(start Offset, field map[Offset]int, edgeBlocked EdgeBlockedFunc) []Offset {
	var path []Offset
	cur := start
	for {
		next, ok := NextStep(cur, field, edgeBlocked)
		if !ok {
			return path
		}
		path = append(path, next)
		cur = next
	}
}

func closestReachable(distFromStart, distToDest map[Offset]int) Offset {
	var best Offset
	var bestToDest, bestFromStart int
	found := false
	for h, fromStart := range distFromStart {
		toDest, ok := distToDest[h]
		if !ok {
			continue
		}
		better := !found ||
			toDest < bestToDest ||
			(toDest == bestToDest && fromStart < bestFromStart) ||
			(toDest == bestToDest && fromStart == bestFromStart && readingOrderLess(h, best))
		if better {
			best, bestToDest, bestFromStart, found = h, toDest, fromStart, true
		}
	}
	return best
}

func readingOrderLess(a, b Offset) bool {
	if a.Col != b.Col {
		return a.Col < b.Col
	}
	return a.Row < b.Row
}
