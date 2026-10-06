package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

func TestCheckAnnihilation(t *testing.T) {
	templates, _ := loadTestRules(t)

	cases := []struct {
		name       string
		state      GameState
		wantOver   bool
		wantWinner string
	}{
		{
			name:       "south annihilated (no units, no reserves)",
			state:      GameState{Board: starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N))},
			wantOver:   true,
			wantWinner: "north",
		},
		{
			name: "south has reserves, not annihilated",
			state: GameState{
				Board:    starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N)),
				Reserves: map[string][]UnitInstance{"south": {reserveUnit(t, templates, "B", "south", "infantry")}},
			},
			wantOver: false,
		},
		{
			name: "both sides alive",
			state: GameState{Board: starterBoard(
				newUnit(t, templates, "A", "north", "infantry", "A1", hex.N),
				newUnit(t, templates, "B", "south", "infantry", "L10", hex.N),
			)},
			wantOver: false,
		},
	}
	for _, c := range cases {
		got := CheckAnnihilation(c.state)
		if got.Over != c.wantOver {
			t.Errorf("%s: Over = %v, want %v", c.name, got.Over, c.wantOver)
		}
		if got.Winner != c.wantWinner {
			t.Errorf("%s: Winner = %q, want %q", c.name, got.Winner, c.wantWinner)
		}
	}
}

func TestScoreAtTurnLimit(t *testing.T) {
	templates, _ := loadTestRules(t)
	full := newUnit(t, templates, "A", "north", "infantry", "A1", hex.N) // Cost 10.
	half := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	half.Strength = Half // Cost 10 / 2 = 5.
	state := GameState{Board: starterBoard(full, half)}

	north, south, outcome := ScoreAtTurnLimit(state)
	if north != 10 {
		t.Errorf("northScore = %d, want 10", north)
	}
	if south != 5 {
		t.Errorf("southScore = %d, want 5", south)
	}
	if !outcome.Over || outcome.Winner != "north" {
		t.Errorf("outcome = %+v, want north to win on points", outcome)
	}
}

func TestScoreAtTurnLimitDraw(t *testing.T) {
	templates, _ := loadTestRules(t)
	a := newUnit(t, templates, "A", "north", "infantry", "A1", hex.N)
	b := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	state := GameState{Board: starterBoard(a, b)}

	_, _, outcome := ScoreAtTurnLimit(state)
	if !outcome.Over || outcome.Winner != "" {
		t.Errorf("outcome = %+v, want a draw", outcome)
	}
}
