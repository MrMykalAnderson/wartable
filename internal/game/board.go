package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// Board is the units on a scenario's map (docs/core-rules.md section 2).
// Terrain is the scenario's map data (section 2.3), or nil for a
// scenario with no terrain (e.g. the Starter Battle): every terrain-
// aware check then behaves as open ground everywhere. It's never
// serialized (every event's board snapshot would otherwise repeat it);
// callers re-attach it from the loaded scenario after reading a board
// back from a save.
type Board struct {
	Columns, Rows int
	Units         []UnitInstance
	Terrain       *rules.TerrainMap `json:"-"`
}

// InBounds reports whether a hex is on the map (docs/core-rules.md section
// 7.1).
func (b Board) InBounds(o hex.Offset) bool {
	return o.Col >= 0 && o.Col < b.Columns && o.Row >= 0 && o.Row < b.Rows
}

// UnitAt returns the unit standing on pos, if any.
func (b Board) UnitAt(pos hex.Offset) (UnitInstance, bool) {
	for _, u := range b.Units {
		if u.Pos == pos {
			return u, true
		}
	}
	return UnitInstance{}, false
}

// Unit returns the unit with the given ID.
func (b Board) Unit(id string) (UnitInstance, bool) {
	for _, u := range b.Units {
		if u.ID == id {
			return u, true
		}
	}
	return UnitInstance{}, false
}

// adjacentHexes returns pos's neighbours that count as adjacent for game
// rules: a river edge with no bridge excludes a neighbour, even though
// it's still geometrically next door (docs/core-rules.md section 2.3:
// "across a river" — contact, support, ambush, overrun and no man's
// land all ignore such pairs).
func (b Board) adjacentHexes(pos hex.Offset) []hex.Offset {
	var out []hex.Offset
	for _, nb := range pos.Neighbors() {
		if b.Terrain.RiverBlocks(pos, nb) {
			continue
		}
		out = append(out, nb)
	}
	return out
}

// Support counts the units of side adjacent to pos (docs/core-rules.md
// section 8.1).
func (b Board) Support(side string, pos hex.Offset) int {
	n := 0
	for _, nb := range b.adjacentHexes(pos) {
		if u, ok := b.UnitAt(nb); ok && u.Side == side {
			n++
		}
	}
	return n
}

// AdjacentEnemy reports whether any unit not of side is adjacent to pos.
func (b Board) AdjacentEnemy(side string, pos hex.Offset) bool {
	for _, nb := range b.adjacentHexes(pos) {
		if u, ok := b.UnitAt(nb); ok && u.Side != side {
			return true
		}
	}
	return false
}

// AdjacentEnemies returns the units not of side adjacent to pos, in
// clockwise order from pos's N edge (docs/core-rules.md section 7.4).
func (b Board) AdjacentEnemies(side string, pos hex.Offset) []UnitInstance {
	var out []UnitInstance
	for _, d := range hex.Directions {
		nb := pos.Neighbor(d)
		if b.Terrain.RiverBlocks(pos, nb) {
			continue
		}
		if u, ok := b.UnitAt(nb); ok && u.Side != side {
			out = append(out, u)
		}
	}
	return out
}

// WithUnit returns a copy of the board with u added, or replacing the
// existing unit with the same ID.
func (b Board) WithUnit(u UnitInstance) Board {
	units := make([]UnitInstance, len(b.Units))
	copy(units, b.Units)
	for i := range units {
		if units[i].ID == u.ID {
			units[i] = u
			return Board{Columns: b.Columns, Rows: b.Rows, Units: units, Terrain: b.Terrain}
		}
	}
	units = append(units, u)
	return Board{Columns: b.Columns, Rows: b.Rows, Units: units, Terrain: b.Terrain}
}

// WithoutUnit returns a copy of the board with the unit of the given ID
// removed (docs/core-rules.md section 3.2: destruction).
func (b Board) WithoutUnit(id string) Board {
	units := make([]UnitInstance, 0, len(b.Units))
	for _, u := range b.Units {
		if u.ID != id {
			units = append(units, u)
		}
	}
	return Board{Columns: b.Columns, Rows: b.Rows, Units: units, Terrain: b.Terrain}
}

// CanRetreatTo reports whether a unit of side can be knocked back from
// "from" onto "to" (docs/core-rules.md section 8.3): the hex must be on
// the map, empty, not adjacent to any enemy, and not across a river
// edge from "from".
func (b Board) CanRetreatTo(side string, from, to hex.Offset) bool {
	if !b.InBounds(to) {
		return false
	}
	if _, occupied := b.UnitAt(to); occupied {
		return false
	}
	if b.Terrain.RiverBlocks(from, to) {
		return false
	}
	return !b.AdjacentEnemy(side, to)
}
