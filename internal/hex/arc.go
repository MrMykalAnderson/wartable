package hex

// InFiringArc reports whether target lies in the firing arc of a unit at
// unitHex facing in direction facing (docs/core-rules.md section 4): the
// 120-degree wedge bounded by the lines of hexes running through the unit's
// two flank edges, including those lines.
func InFiringArc(unitHex Offset, facing Direction, target Offset) bool {
	delta := target.ToCube().sub(unitHex.ToCube())
	d1 := cubeStep[facing.RotateCW(-1)]
	d2 := cubeStep[facing.RotateCW(1)]

	// d1 and d2 form a unimodular basis of the hex lattice (det = ±1), so
	// this division is always exact.
	det := d1.Q*d2.R - d2.Q*d1.R
	a := (delta.Q*d2.R - delta.R*d2.Q) / det
	b := (delta.R*d1.Q - delta.Q*d1.R) / det
	return a >= 0 && b >= 0
}

// Position is where an attack lands relative to the defender's facing.
type Position int

const (
	Front Position = iota
	Flank
	Rear
)

func (p Position) String() string {
	switch p {
	case Front:
		return "front"
	case Flank:
		return "flank"
	case Rear:
		return "rear"
	default:
		return "unknown"
	}
}

// EdgeAttacked reports which of the defender's edges the attacker is
// attacking across, given the defender's facing (docs/core-rules.md section
// 4). It returns false if defender and attacker aren't adjacent.
func EdgeAttacked(defenderFacing Direction, defender, attacker Offset) (Position, bool) {
	dir, ok := AdjacentDirection(defender, attacker)
	if !ok {
		return 0, false
	}
	switch diff := dir.RotateCW(-int(defenderFacing)); diff {
	case N:
		return Front, true
	case NE, NW:
		return Flank, true
	default:
		return Rear, true
	}
}
