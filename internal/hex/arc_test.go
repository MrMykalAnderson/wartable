package hex

import "testing"

// TestInFiringArc checks the arc for a unit at F6: facing N, the arc is
// bounded by (and includes) the lines of hexes running NE and NW
// (docs/core-rules.md section 4).
func TestInFiringArc(t *testing.T) {
	unit := mustParse(t, "F6")
	cases := []struct {
		target string
		facing Direction
		want   bool
	}{
		{"F5", N, true},  // straight ahead.
		{"F4", N, true},  // further ahead, same line.
		{"G6", N, true},  // on the NE boundary line.
		{"E6", N, true},  // on the NW boundary line.
		{"F7", N, false}, // directly behind.
		{"E7", N, false}, // rear (SW).
		{"G7", N, false}, // rear (SE).
		{"F6", N, true},  // the unit's own hex (degenerate case).

		// Rotating the facing rotates the arc with it.
		{"F7", S, true},
		{"F5", S, false},
	}
	for _, c := range cases {
		target := mustParse(t, c.target)
		if got := InFiringArc(unit, c.facing, target); got != c.want {
			t.Errorf("InFiringArc(%s, %s, %s) = %v, want %v", unit, c.facing, c.target, got, c.want)
		}
	}
}

// TestEdgeAttacked checks EX-2 (front), EX-3 (flank) and EX-4 (rear) from
// docs/core-rules.md section 11.
func TestEdgeAttacked(t *testing.T) {
	cases := []struct {
		name           string
		defenderFacing Direction
		defender       string
		attacker       string
		want           Position
	}{
		{"EX-3 flank", N, "F6", "G6", Flank},
		{"EX-4 rear", N, "F6", "F7", Rear},
		{"front", N, "F6", "F5", Front},
	}
	for _, c := range cases {
		defender, attacker := mustParse(t, c.defender), mustParse(t, c.attacker)
		got, ok := EdgeAttacked(c.defenderFacing, defender, attacker)
		if !ok {
			t.Errorf("%s: EdgeAttacked(%s, %s, %s): not adjacent", c.name, c.defenderFacing, c.defender, c.attacker)
			continue
		}
		if got != c.want {
			t.Errorf("%s: EdgeAttacked(%s, %s, %s) = %s, want %s", c.name, c.defenderFacing, c.defender, c.attacker, got, c.want)
		}
	}
}

func TestEdgeAttackedNotAdjacent(t *testing.T) {
	defender, attacker := mustParse(t, "F6"), mustParse(t, "A1")
	if _, ok := EdgeAttacked(N, defender, attacker); ok {
		t.Errorf("EdgeAttacked(F6, A1): want not-adjacent, got a result")
	}
}
