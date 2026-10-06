package game

import "github.com/MrMykalAnderson/wartable/internal/hex"

// Board is the units on a scenario's map (docs/core-rules.md section 2).
type Board struct {
	Columns, Rows int
	Units         []UnitInstance
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

// Support counts the units of side adjacent to pos (docs/core-rules.md
// section 8.1).
func (b Board) Support(side string, pos hex.Offset) int {
	n := 0
	for _, nb := range pos.Neighbors() {
		if u, ok := b.UnitAt(nb); ok && u.Side == side {
			n++
		}
	}
	return n
}

// AdjacentEnemy reports whether any unit not of side is adjacent to pos.
func (b Board) AdjacentEnemy(side string, pos hex.Offset) bool {
	for _, nb := range pos.Neighbors() {
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
		if u, ok := b.UnitAt(pos.Neighbor(d)); ok && u.Side != side {
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
			return Board{Columns: b.Columns, Rows: b.Rows, Units: units}
		}
	}
	units = append(units, u)
	return Board{Columns: b.Columns, Rows: b.Rows, Units: units}
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
	return Board{Columns: b.Columns, Rows: b.Rows, Units: units}
}

// CanRetreatTo reports whether a unit of side can be knocked back onto pos
// (docs/core-rules.md section 8.3): the hex must be on the map, empty, and
// not adjacent to any enemy.
func (b Board) CanRetreatTo(side string, pos hex.Offset) bool {
	if !b.InBounds(pos) {
		return false
	}
	if _, occupied := b.UnitAt(pos); occupied {
		return false
	}
	return !b.AdjacentEnemy(side, pos)
}
