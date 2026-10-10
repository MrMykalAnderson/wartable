package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// Outcome is the result of checking a scenario's win conditions
// (docs/starter-battle.md "Winning", docs/two-towns.md "Winning").
type Outcome struct {
	Over   bool
	Winner string // "" if the game isn't over, or ended in a draw.
	Reason string
}

// CheckAnnihilation reports whether either side has no units on the map
// and none left to deploy, in which case the other side wins immediately
// (docs/starter-battle.md "Winning", rule 1; docs/two-towns.md "Winning",
// rule 2).
func CheckAnnihilation(state GameState) Outcome {
	northAlive := sideAlive(state, "north")
	southAlive := sideAlive(state, "south")
	switch {
	case !northAlive && !southAlive:
		return Outcome{Over: true, Reason: "both sides annihilated"}
	case !northAlive:
		return Outcome{Over: true, Winner: "south", Reason: "north annihilated"}
	case !southAlive:
		return Outcome{Over: true, Winner: "north", Reason: "south annihilated"}
	default:
		return Outcome{}
	}
}

func sideAlive(state GameState, side string) bool {
	for _, u := range state.Board.Units {
		if u.Side == side {
			return true
		}
	}
	return len(state.Reserves[side]) > 0
}

// otherSide returns the opponent of side ("north"/"south").
func otherSide(side string) string {
	if side == "north" {
		return "south"
	}
	return "north"
}

// CheckCapture reports whether either side has a unit in the other
// side's own town while the other side has none there (docs/two-
// towns.md "Winning", rule 1). It's a no-op (never over) for a
// scenario with no terrain, or whose deployment zones don't name an
// own town (e.g. the Starter Battle).
func CheckCapture(state GameState, scenario rules.Scenario) Outcome {
	if scenario.Terrain == nil {
		return Outcome{}
	}
	for _, side := range []string{"north", "south"} {
		enemy := otherSide(side)
		enemyTown := scenario.DeploymentZones[enemy].NearTown
		if enemyTown == "" {
			continue
		}
		townHexes := scenario.Terrain.Towns[enemyTown]
		if anyUnitIn(state.Board, side, townHexes) && !anyUnitIn(state.Board, enemy, townHexes) {
			return Outcome{Over: true, Winner: side, Reason: fmt.Sprintf("captured %s", enemyTown)}
		}
	}
	return Outcome{}
}

func anyUnitIn(board Board, side string, hexes []hex.Offset) bool {
	for _, h := range hexes {
		if u, ok := board.UnitAt(h); ok && u.Side == side {
			return true
		}
	}
	return false
}

// ScoreAtTurnLimit scores each side's surviving units (a half-strength
// unit counts for half its Cost, rounded down), plus the objective
// points for every town and hamlet it owns (docs/two-towns.md
// "Winning", rule 3; 0 for a scenario with no terrain), and reports who
// has the higher score (docs/starter-battle.md "Winning", rule 2).
func ScoreAtTurnLimit(state GameState, core rules.CoreRules, scenario rules.Scenario) (northScore, southScore int, outcome Outcome) {
	northScore = score(state, core, scenario, "north")
	southScore = score(state, core, scenario, "south")
	switch {
	case northScore > southScore:
		outcome = Outcome{Over: true, Winner: "north", Reason: "turn limit"}
	case southScore > northScore:
		outcome = Outcome{Over: true, Winner: "south", Reason: "turn limit"}
	default:
		outcome = Outcome{Over: true, Reason: "turn limit: draw"}
	}
	return northScore, southScore, outcome
}

func score(state GameState, core rules.CoreRules, scenario rules.Scenario, side string) int {
	total := 0
	for _, u := range state.Board.Units {
		if u.Side != side {
			continue
		}
		cost := u.Template.Cost
		if u.Strength == Half {
			cost /= 2
		}
		total += cost
	}
	if scenario.Terrain == nil {
		return total
	}
	for name, owner := range state.Objectives {
		if owner != side {
			continue
		}
		if _, ok := scenario.Terrain.Towns[name]; ok {
			total += core.Terrain.TownPoints
		} else {
			total += core.Terrain.HamletPoints
		}
	}
	return total
}
