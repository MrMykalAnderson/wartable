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
	templates, core := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	full := newUnit(t, templates, "A", "north", "infantry", "A1", hex.N) // Cost 10.
	half := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	half.Strength = Half // Cost 10 / 2 = 5.
	state := GameState{Board: starterBoard(full, half)}

	north, south, outcome := ScoreAtTurnLimit(state, core, scenario)
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
	templates, core := loadTestRules(t)
	scenario := loadTestScenario(t, templates)
	a := newUnit(t, templates, "A", "north", "infantry", "A1", hex.N)
	b := newUnit(t, templates, "B", "south", "infantry", "L10", hex.N)
	state := GameState{Board: starterBoard(a, b)}

	_, _, outcome := ScoreAtTurnLimit(state, core, scenario)
	if !outcome.Over || outcome.Winner != "" {
		t.Errorf("outcome = %+v, want a draw", outcome)
	}
}

// TestCheckCapture checks docs/two-towns.md "Winning" rule 1: a unit in
// the enemy's own town, with none of theirs there, wins by capture. The
// Starter Battle (no terrain) never triggers it.
func TestCheckCapture(t *testing.T) {
	templates, _ := loadTestRules(t)
	twoTowns := loadTestTwoTowns(t, templates)
	starter := loadTestScenario(t, templates)

	raider := newUnit(t, templates, "Raider", "south", "cavalry", "B15", hex.N) // inside North town.
	notCapture := GameState{Board: boardFor(twoTowns, raider,
		newUnit(t, templates, "Defender", "north", "infantry", "C15", hex.N), // also in North town.
	)}
	if got := CheckCapture(notCapture, twoTowns); got.Over {
		t.Errorf("CheckCapture (defender still present) = %+v, want not over", got)
	}

	captured := GameState{Board: boardFor(twoTowns, raider)}
	got := CheckCapture(captured, twoTowns)
	if !got.Over || got.Winner != "south" {
		t.Errorf("CheckCapture (North town undefended, South inside) = %+v, want South wins", got)
	}

	noTerrain := GameState{Board: starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N))}
	if got := CheckCapture(noTerrain, starter); got.Over {
		t.Errorf("CheckCapture on the Starter Battle (no terrain) = %+v, want never over", got)
	}
}

// TestScoreAtTurnLimitObjectivePoints checks docs/two-towns.md
// "Winning" rule 3: Cost plus 20 per owned town and 10 per owned
// hamlet.
func TestScoreAtTurnLimitObjectivePoints(t *testing.T) {
	templates, core := loadTestRules(t)
	scenario := loadTestTwoTowns(t, templates)
	a := newUnit(t, templates, "A", "north", "infantry", "A1", hex.N) // Cost 10.
	state := GameState{Board: boardFor(scenario, a), Objectives: map[string]string{}}
	for name := range scenario.Terrain.AllObjectives() {
		state.Objectives[name] = "north" // north owns every objective on the map.
	}

	north, _, _ := ScoreAtTurnLimit(state, core, scenario)
	wantTownPoints := len(scenario.Terrain.Towns) * core.Terrain.TownPoints
	wantHamletPoints := len(scenario.Terrain.Hamlets) * core.Terrain.HamletPoints
	want := 10 + wantTownPoints + wantHamletPoints
	if north != want {
		t.Errorf("northScore = %d, want %d (Cost 10 + %d town points + %d hamlet points)", north, want, wantTownPoints, wantHamletPoints)
	}
}
