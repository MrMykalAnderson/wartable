// Package hex implements the coordinate geometry for Wartable's map:
// offset/cube coordinates, directions, distance, firing arcs, knockback
// and pathfinding. See docs/core-rules.md section 2 and docs/dev-plan.md
// section 5.
package hex

import (
	"fmt"
	"math"
	"strconv"
)

// Offset is a hex identified by 0-based column (A = 0) and 0-based row
// (row 1 = 0), matching the column-letter/row-number names used in the
// rules and on order sheets.
type Offset struct {
	Col, Row int
}

// ParseOffset parses a hex name such as "A1" or "AA12".
func ParseOffset(s string) (Offset, error) {
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(s) {
		return Offset{}, fmt.Errorf("hex: invalid hex name %q", s)
	}
	col := 0
	for _, c := range s[:i] {
		col = col*26 + int(c-'A'+1)
	}
	row, err := strconv.Atoi(s[i:])
	if err != nil || row < 1 {
		return Offset{}, fmt.Errorf("hex: invalid hex name %q", s)
	}
	return Offset{Col: col - 1, Row: row - 1}, nil
}

// String renders the hex as a column-letter/row-number name, e.g. "A1".
func (o Offset) String() string {
	n := o.Col + 1
	var letters []byte
	for n > 0 {
		n--
		letters = append([]byte{byte('A' + n%26)}, letters...)
		n /= 26
	}
	return fmt.Sprintf("%s%d", letters, o.Row+1)
}

// Cube is a cube coordinate, where Q+R+S always equals 0.
type Cube struct {
	Q, R, S int
}

// ToCube converts an offset coordinate to a cube coordinate (docs/dev-plan.md
// section 5).
func (o Offset) ToCube() Cube {
	q := o.Col
	r := o.Row - (o.Col-(o.Col&1))/2
	return Cube{Q: q, R: r, S: -q - r}
}

// ToOffset converts a cube coordinate back to an offset coordinate.
func (c Cube) ToOffset() Offset {
	col := c.Q
	row := c.R + (col-(col&1))/2
	return Offset{Col: col, Row: row}
}

// Direction is one of the six hex edge directions.
type Direction int

const (
	N Direction = iota
	NE
	SE
	S
	SW
	NW
)

// Directions lists all six directions in clockwise order starting from N.
// Tie-breaks in pathfinding and ambush order use this order.
var Directions = [6]Direction{N, NE, SE, S, SW, NW}

func (d Direction) String() string {
	return [6]string{"N", "NE", "SE", "S", "SW", "NW"}[((int(d)%6)+6)%6]
}

// RotateCW rotates the direction clockwise by n steps (n may be negative).
func (d Direction) RotateCW(n int) Direction {
	return Direction((((int(d) + n) % 6) + 6) % 6)
}

// Opposite returns the direction pointing the opposite way.
func (d Direction) Opposite() Direction {
	return d.RotateCW(3)
}

var cubeStep = map[Direction]Cube{
	N:  {Q: 0, R: -1, S: 1},
	NE: {Q: 1, R: -1, S: 0},
	SE: {Q: 1, R: 0, S: -1},
	S:  {Q: 0, R: 1, S: -1},
	SW: {Q: -1, R: 1, S: 0},
	NW: {Q: -1, R: 0, S: 1},
}

// Add returns the cube reached by moving one step in direction d.
func (c Cube) Add(d Direction) Cube {
	s := cubeStep[d]
	return Cube{Q: c.Q + s.Q, R: c.R + s.R, S: c.S + s.S}
}

func (c Cube) sub(o Cube) Cube {
	return Cube{Q: c.Q - o.Q, R: c.R - o.R, S: c.S - o.S}
}

func (c Cube) add(o Cube) Cube {
	return Cube{Q: c.Q + o.Q, R: c.R + o.R, S: c.S + o.S}
}

// Neighbor returns the hex one step away in direction d.
func (o Offset) Neighbor(d Direction) Offset {
	return o.ToCube().Add(d).ToOffset()
}

// Neighbors returns all six neighbouring hexes, in clockwise order from N.
func (o Offset) Neighbors() [6]Offset {
	var out [6]Offset
	for i, d := range Directions {
		out[i] = o.Neighbor(d)
	}
	return out
}

// AdjacentDirection returns the direction from "from" to "to" if the two
// hexes are adjacent (distance 1), and false otherwise.
func AdjacentDirection(from, to Offset) (Direction, bool) {
	for _, d := range Directions {
		if from.Neighbor(d) == to {
			return d, true
		}
	}
	return 0, false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// CubeDistance returns the number of steps between two cube coordinates.
func CubeDistance(a, b Cube) int {
	d := a.sub(b)
	return (abs(d.Q) + abs(d.R) + abs(d.S)) / 2
}

// Distance returns the number of steps between two hexes (docs/core-rules.md
// section 2.2).
func Distance(a, b Offset) int {
	return CubeDistance(a.ToCube(), b.ToCube())
}

// DirectionToward returns the direction whose step is most aligned with
// the vector from "from" to "to": the edge pointing most directly at a
// hex that may be distant and not on any single direction's exact line
// (docs/core-rules.md section 4.1, turning to face a target without
// moving). If from == to, N is returned arbitrarily.
func DirectionToward(from, to Offset) Direction {
	delta := to.ToCube().sub(from.ToCube())
	best, bestDot := N, math.MinInt
	for _, d := range Directions {
		step := cubeStep[d]
		dot := step.Q*delta.Q + step.R*delta.R + step.S*delta.S
		if dot > bestDot {
			best, bestDot = d, dot
		}
	}
	return best
}
