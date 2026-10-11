package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

func TestCheckAnnihilation(t *testing.T) {
	templates, _ := loadTestRules(t)
	scenario := loadTestScenario(t, templates)

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
		{
			// docs/dev-plan.md section 7.10: a side with bank funds at
			// least equal to its cheapest allowed unit (infantry, Cost
			// 10) isn't annihilated even with nothing on the map or in
			// reserves.
			name: "south has enough bank for another unit, not annihilated",
			state: GameState{
				Board: starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N)),
				Bank:  map[string]int{"south": 10},
			},
			wantOver: false,
		},
		{
			name: "south's bank is short of even the cheapest unit, annihilated",
			state: GameState{
				Board: starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N)),
				Bank:  map[string]int{"south": 9},
			},
			wantOver:   true,
			wantWinner: "north",
		},
	}
	for _, c := range cases {
		got := CheckAnnihilation(c.state, templates, scenario)
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
// the enemy's own town, with none of theirs there, captures it — but
// only wins if the same side still holds it the same way at the end of
// the following turn (dev-plan.md section 7.10). The Starter Battle (no
// terrain) never triggers it.
func TestCheckCapture(t *testing.T) {
	templates, _ := loadTestRules(t)
	twoTowns := loadTestTwoTowns(t, templates)
	starter := loadTestScenario(t, templates)

	raider := newUnit(t, templates, "Raider", "south", "cavalry", "B15", hex.N) // inside North town.
	defended := GameState{Board: boardFor(twoTowns, raider,
		newUnit(t, templates, "Defender", "north", "infantry", "C15", hex.N), // also in North town.
	)}
	if got, pending := CheckCapture(defended, twoTowns); got.Over || len(pending) != 0 {
		t.Errorf("CheckCapture (defender still present) = %+v, %v, want not over, nothing pending", got, pending)
	}

	captured := GameState{Board: boardFor(twoTowns, raider)}
	firstTurn, pending := CheckCapture(captured, twoTowns)
	if firstTurn.Over {
		t.Errorf("CheckCapture (first turn captured) = %+v, want not over yet (must hold it one more turn)", firstTurn)
	}
	if pending["North town"] != "south" {
		t.Errorf("pending = %v, want North town pending for south", pending)
	}

	// Held the same way at the end of the following turn: wins.
	captured.PendingCapture = pending
	secondTurn, _ := CheckCapture(captured, twoTowns)
	if !secondTurn.Over || secondTurn.Winner != "south" {
		t.Errorf("CheckCapture (held a second turn) = %+v, want south to win", secondTurn)
	}

	// The defender retakes it (south's raider is gone, north has a unit
	// there again): pending resets, no win.
	retaken := GameState{
		Board:          boardFor(twoTowns, newUnit(t, templates, "Defender", "north", "infantry", "B15", hex.N)),
		PendingCapture: pending,
	}
	resetOutcome, resetPending := CheckCapture(retaken, twoTowns)
	if resetOutcome.Over {
		t.Errorf("CheckCapture (retaken by the defender) = %+v, want not over", resetOutcome)
	}
	if len(resetPending) != 0 {
		t.Errorf("pending after retaking = %v, want empty", resetPending)
	}

	noTerrain := GameState{Board: starterBoard(newUnit(t, templates, "A", "north", "infantry", "A1", hex.N))}
	if got, pending := CheckCapture(noTerrain, starter); got.Over || len(pending) != 0 {
		t.Errorf("CheckCapture on the Starter Battle (no terrain) = %+v, %v, want never over, nothing pending", got, pending)
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
