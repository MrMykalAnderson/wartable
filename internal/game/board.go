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
