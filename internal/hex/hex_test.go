package hex

import "testing"

func mustParse(t *testing.T, s string) Offset {
	t.Helper()
	o, err := ParseOffset(s)
	if err != nil {
		t.Fatalf("ParseOffset(%q): %v", s, err)
	}
	return o
}

func TestParseAndString(t *testing.T) {
	cases := []struct {
		name string
		want Offset
	}{
		{"A1", Offset{Col: 0, Row: 0}},
		{"B1", Offset{Col: 1, Row: 0}},
		{"F6", Offset{Col: 5, Row: 5}},
		{"Z1", Offset{Col: 25, Row: 0}},
		{"AA1", Offset{Col: 26, Row: 0}},
	}
	for _, c := range cases {
		got := mustParse(t, c.name)
		if got != c.want {
			t.Errorf("ParseOffset(%q) = %+v, want %+v", c.name, got, c.want)
		}
		if s := got.String(); s != c.name {
			t.Errorf("%+v.String() = %q, want %q", got, s, c.name)
		}
	}
}

func TestParseOffsetInvalid(t *testing.T) {
	for _, s := range []string{"", "1", "A", "A0", "1A"} {
		if _, err := ParseOffset(s); err == nil {
			t.Errorf("ParseOffset(%q): want error, got none", s)
		}
	}
}

// TestNeighbors checks the offset-direction table in docs/core-rules.md 2.1,
// including its worked examples: SE of A1 is B1, and NW of B1 is A1.
func TestNeighbors(t *testing.T) {
	cases := []struct {
		from string
		dir  Direction
		want string
	}{
		// Worked examples from core-rules.md 2.1.
		{"A1", SE, "B1"},
		{"B1", NW, "A1"},

		// From an odd column (A, C, E ...). Uses C2, not A2, so the
		// SW/NW neighbours stay on the map and are nameable.
		{"C2", N, "C1"},
		{"C2", NE, "D1"},
		{"C2", SE, "D2"},
		{"C2", S, "C3"},
		{"C2", SW, "B2"},
		{"C2", NW, "B1"},

		// From an even column (B, D, F ...).
		{"B2", N, "B1"},
		{"B2", NE, "C2"},
		{"B2", SE, "C3"},
		{"B2", S, "B3"},
		{"B2", SW, "A3"},
		{"B2", NW, "A2"},
	}
	for _, c := range cases {
		from := mustParse(t, c.from)
		got := from.Neighbor(c.dir)
		want := mustParse(t, c.want)
		if got != want {
			t.Errorf("%s.Neighbor(%s) = %s, want %s", c.from, c.dir, got, c.want)
		}
	}
}

func TestOffsetCubeRoundTrip(t *testing.T) {
	for col := 0; col < 10; col++ {
		for row := 0; row < 10; row++ {
			o := Offset{Col: col, Row: row}
			got := o.ToCube().ToOffset()
			if got != o {
				t.Errorf("%+v.ToCube().ToOffset() = %+v, want %+v", o, got, o)
			}
		}
	}
}

func TestCubeCoordinatesSumToZero(t *testing.T) {
	for col := 0; col < 10; col++ {
		for row := 0; row < 10; row++ {
			c := Offset{Col: col, Row: row}.ToCube()
			if c.Q+c.R+c.S != 0 {
				t.Errorf("Offset{%d,%d}.ToCube() = %+v, Q+R+S != 0", col, row, c)
			}
		}
	}
}

func TestDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"A1", "A1", 0},
		{"A1", "B1", 1}, // adjacent (SE of A1).
		{"F6", "E7", 1}, // adjacent (knockback target in EX-3).
		{"F6", "F7", 1}, // adjacent (EX-4).
		{"A1", "A5", 4},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := Distance(a, b); got != c.want {
			t.Errorf("Distance(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
		if got := Distance(b, a); got != c.want {
			t.Errorf("Distance(%s, %s) = %d, want %d", c.b, c.a, got, c.want)
		}
	}
}

func TestAdjacentDirection(t *testing.T) {
	a, b := mustParse(t, "F6"), mustParse(t, "G6")
	dir, ok := AdjacentDirection(a, b)
	if !ok || dir != NE {
		t.Errorf("AdjacentDirection(F6, G6) = %v, %v, want NE, true", dir, ok)
	}

	nonAdjacent := mustParse(t, "A1")
	if _, ok := AdjacentDirection(a, nonAdjacent); ok {
		t.Errorf("AdjacentDirection(F6, A1): want false, got true")
	}
}

func TestRotateCWAndOpposite(t *testing.T) {
	if got := N.RotateCW(1); got != NE {
		t.Errorf("N.RotateCW(1) = %v, want NE", got)
	}
	if got := N.RotateCW(-1); got != NW {
		t.Errorf("N.RotateCW(-1) = %v, want NW", got)
	}
	if got := N.RotateCW(6); got != N {
		t.Errorf("N.RotateCW(6) = %v, want N", got)
	}
	for _, d := range Directions {
		if got := d.Opposite().Opposite(); got != d {
			t.Errorf("%v.Opposite().Opposite() = %v, want %v", d, got, d)
		}
	}
}
