package hex

import "container/heap"

// CostFunc returns the movement cost to enter hex o (docs/core-rules.md
// section 2.3): at least 1. Unlike PassableFunc, cost is a property of
// the hex being entered, not of the hex being left — so it's generally
// cheaper or more expensive depending on direction of travel only
// insofar as the two endpoints' own costs differ, never the edge
// itself (see EdgeBlockedFunc for edge-level effects like a river).
type CostFunc func(Offset) int

// WeightedDist is a hex's distance from some origin under a CostFunc:
// Cost is the primary ranking (docs/core-rules.md section 7.2: the
// cheapest route), Hexes (the number of steps) is the tie-break before
// clockwise direction order.
type WeightedDist struct {
	Cost  int
	Hexes int
}

func (a WeightedDist) less(b WeightedDist) bool {
	if a.Cost != b.Cost {
		return a.Cost < b.Cost
	}
	return a.Hexes < b.Hexes
}

// weightedDistTo returns, for every passable hex reachable from dest,
// the cost of the cheapest route from that hex TO dest (docs/core-
// rules.md section 7.2) — i.e. cost(h, dest) for each h, found by a
// Dijkstra flood fill starting AT dest but spreading via the reverse
// relaxation dist[n] = dist[cur] + cost(cur), not cost(n).
//
// This reverse relaxation is deliberate, not a mistake: cost is a
// property of the hex being ENTERED, so a real route h -> ... -> dest
// pays cost(p) for every hex p on it except h itself (including dest).
// Flood-filling outward from dest with dist[n] = dist[cur] + cost(cur)
// accumulates exactly that sum by the time it reaches h, by symmetry of
// the search (walking dest -> ... -> h backward pays, at each step
// cur -> n, the cost of the hex "cur" would be entering if that step
// were instead taken forward, n -> cur). dist[n] = dist[cur] + cost(n)
// (paying to enter n, the forward direction) would instead compute
// cost(dest, h), which is generally different when costs are
// asymmetric (forest costs more to enter than to leave).
func weightedDistTo(dest Offset, passable PassableFunc, cost CostFunc, edgeBlocked EdgeBlockedFunc) map[Offset]WeightedDist {
	dist := map[Offset]WeightedDist{dest: {}}
	pq := &weightedQueue{{dest, WeightedDist{}}}
	for pq.Len() > 0 {
		item := heap.Pop(pq).(weightedItem)
		cur := item.hex
		if item.dist != dist[cur] {
			continue // stale queue entry, already improved on.
		}
		stepCost := cost(cur)
		for _, d := range Directions {
			n := cur.Neighbor(d)
			if !passable(n) {
				continue
			}
			if blocked(edgeBlocked, cur, n) {
				continue
			}
			nd := WeightedDist{Cost: item.dist.Cost + stepCost, Hexes: item.dist.Hexes + 1}
			if existing, ok := dist[n]; !ok || nd.less(existing) {
				dist[n] = nd
				heap.Push(pq, weightedItem{n, nd})
			}
		}
	}
	return dist
}

// ReachCost returns, for every passable hex reachable from origin, the
// cost of the cheapest route FROM origin to that hex — i.e.
// cost(origin, h) for each h, the natural forward direction
// (dist[n] = dist[cur] + cost(n): paying to enter n). Exported for a
// reach guide: every hex reachable for cost at most some budget
// (docs/core-rules.md section 2.3; dev-plan.md section 7.10).
func ReachCost(origin Offset, passable PassableFunc, cost CostFunc, edgeBlocked EdgeBlockedFunc) map[Offset]WeightedDist {
	dist := map[Offset]WeightedDist{origin: {}}
	pq := &weightedQueue{{origin, WeightedDist{}}}
	for pq.Len() > 0 {
		item := heap.Pop(pq).(weightedItem)
		cur := item.hex
		if item.dist != dist[cur] {
			continue
		}
		for _, d := range Directions {
			n := cur.Neighbor(d)
			if !passable(n) {
				continue
			}
			if blocked(edgeBlocked, cur, n) {
				continue
			}
			nd := WeightedDist{Cost: item.dist.Cost + cost(n), Hexes: item.dist.Hexes + 1}
			if existing, ok := dist[n]; !ok || nd.less(existing) {
				dist[n] = nd
				heap.Push(pq, weightedItem{n, nd})
			}
		}
	}
	return dist
}

// weightedNextStep returns the neighbour of cur that is one real
// (forward) step closer to the dest that field (from weightedDistTo)
// was computed for, breaking ties by clockwise direction order. ok is
// false if cur is dest itself or isn't in field.
func weightedNextStep(cur Offset, field map[Offset]WeightedDist, cost CostFunc, edgeBlocked EdgeBlockedFunc) (next Offset, ok bool) {
	curDist, inField := field[cur]
	if !inField || curDist == (WeightedDist{}) {
		return Offset{}, false
	}
	for _, d := range Directions {
		n := cur.Neighbor(d)
		nDist, inField := field[n]
		if !inField || blocked(edgeBlocked, cur, n) {
			continue
		}
		if nDist.Cost+cost(n) == curDist.Cost && nDist.Hexes+1 == curDist.Hexes {
			return n, true
		}
	}
	return Offset{}, false
}

func weightedWalk(start Offset, field map[Offset]WeightedDist, cost CostFunc, edgeBlocked EdgeBlockedFunc) []Offset {
	var path []Offset
	cur := start
	for {
		next, ok := weightedNextStep(cur, field, cost, edgeBlocked)
		if !ok {
			return path
		}
		path = append(path, next)
		cur = next
	}
}

// weightedClosestReachable mirrors closestReachable (path.go) for the
// weighted case: the reachable hex (from distFromStart) cheapest to
// reach dest (from distToDest), tied first by the fewest-steps rule
// for the moving unit, then reading order (docs/core-rules.md 7.2's
// provisional unreachable-destination rule).
func weightedClosestReachable(distFromStart, distToDest map[Offset]WeightedDist) Offset {
	var best Offset
	var bestToDest, bestFromStart WeightedDist
	found := false
	for h, fromStart := range distFromStart {
		toDest, ok := distToDest[h]
		if !ok {
			continue
		}
		better := !found ||
			toDest.less(bestToDest) ||
			(toDest == bestToDest && fromStart.less(bestFromStart)) ||
			(toDest == bestToDest && fromStart == bestFromStart && readingOrderLess(h, best))
		if better {
			best, bestToDest, bestFromStart, found = h, toDest, fromStart, true
		}
	}
	return best
}

// CheapestPath finds the cheapest route from start to dest by total
// movement cost (docs/core-rules.md section 7.2: Dijkstra), tied first
// by fewer hexes and then by clockwise direction order at each step —
// the weighted equivalent of ShortestPath, which it reduces to when
// cost is uniformly 1. dest itself must be passable to be reached; an
// impassable dest (e.g. occupied) always falls through to the heading-
// for behaviour below. If dest can't be reached, the path instead heads
// for the reachable hex cheapest to reach from dest, breaking ties by
// cost from start and then by reading order. The returned path excludes
// start and includes the hex actually headed for; reached reports
// whether that hex is dest.
func CheapestPath(start, dest Offset, passable PassableFunc, cost CostFunc, edgeBlocked EdgeBlockedFunc) (path []Offset, reached bool) {
	distToDest := weightedDistTo(dest, passable, cost, edgeBlocked)
	if passable(dest) {
		if _, ok := distToDest[start]; ok {
			return weightedWalk(start, distToDest, cost, edgeBlocked), true
		}
	}

	distFromStart := ReachCost(start, passable, cost, edgeBlocked)
	target := weightedClosestReachable(distFromStart, distToDest)
	return weightedWalk(start, weightedDistTo(target, passable, cost, edgeBlocked), cost, edgeBlocked), false
}

// weightedItem/weightedQueue implement container/heap for Dijkstra,
// ordered by WeightedDist.less.
type weightedItem struct {
	hex  Offset
	dist WeightedDist
}

type weightedQueue []weightedItem

func (q weightedQueue) Len() int            { return len(q) }
func (q weightedQueue) Less(i, j int) bool  { return q[i].dist.less(q[j].dist) }
func (q weightedQueue) Swap(i, j int)       { q[i], q[j] = q[j], q[i] }
func (q *weightedQueue) Push(x interface{}) { *q = append(*q, x.(weightedItem)) }
func (q *weightedQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	*q = old[:n-1]
	return item
}
