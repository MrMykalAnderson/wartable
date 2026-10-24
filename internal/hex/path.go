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
