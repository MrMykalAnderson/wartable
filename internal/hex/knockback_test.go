package hex

import "testing"

// TestKnockback checks EX-3 and EX-4 from docs/core-rules.md section 11.
func TestKnockback(t *testing.T) {
	cases := []struct {
		name          string
		winner, loser string
		want          string
	}{
		{"EX-3", "G6", "F6", "E7"}, // the line from G6 through F6 runs SW.
		{"EX-4", "F7", "F6", "F5"}, // A at F7 (S of B); B pushed N.
	}
	for _, c := range cases {
		winner, loser := mustParse(t, c.winner), mustParse(t, c.loser)
		got := Knockback(winner, loser)
		want := mustParse(t, c.want)
		if got != want {
			t.Errorf("%s: Knockback(%s, %s) = %s, want %s", c.name, c.winner, c.loser, got, c.want)
		}
	}
}
