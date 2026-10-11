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

// CheckAnnihilation reports whether either side has no units on the
// map, no reserves, and less in the bank than its cheapest allowed unit
// (so it could never field another one), in which case the other side
// wins immediately (docs/starter-battle.md "Winning", rule 1; docs/two-
// towns.md "Winning", rule 2; dev-plan.md section 7.10 added the bank
// check — a scenario with no bank concept always has Bank 0, which is
// less than any unit's Cost, so nothing changes for it).
func CheckAnnihilation(state GameState, units map[string]rules.Unit, scenario rules.Scenario) Outcome {
	northAlive := sideAlive(state, units, scenario, "north")
	southAlive := sideAlive(state, units, scenario, "south")
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

func sideAlive(state GameState, units map[string]rules.Unit, scenario rules.Scenario, side string) bool {
	for _, u := range state.Board.Units {
		if u.Side == side {
			return true
		}
	}
	if len(state.Reserves[side]) > 0 {
		return true
	}
	return state.Bank[side] >= cheapestUnitCost(units, scenario)
}

// cheapestUnitCost is the lowest Cost among the scenario's allowed unit
// types, for the annihilation check: 0 if units_allowed is somehow
// empty or names nothing in units (shouldn't happen — rules.LoadScenario
// already validates units_allowed against the unit roster).
func cheapestUnitCost(units map[string]rules.Unit, scenario rules.Scenario) int {
	cheapest := -1
	for _, id := range scenario.UnitsAllowed {
		u, ok := units[id]
		if !ok {
			continue
		}
		if cheapest == -1 || u.Cost < cheapest {
			cheapest = u.Cost
		}
	}
	if cheapest == -1 {
		return 0
	}
	return cheapest
}

// otherSide returns the opponent of side ("north"/"south").
func otherSide(side string) string {
	if side == "north" {
		return "south"
	}
	return "north"
}

// CheckCapture reports whether a side has just won by capture
// (docs/two-towns.md "Winning", rule 1): a town is captured at the end
// of a turn if one side has a unit in the other's own town and the
// other has none there; capturing it again in the same way at the end
// of the following turn wins. state.PendingCapture (as it stood before
// this call) is what was captured, by whom, at the end of the previous
// turn; the returned map is this turn's capture state, which the
// caller must store back as the new PendingCapture for the check to
// mean anything next time (see cmd/wartable/turn.go, cmd/wartable-
// web/handlers.go). Both are a no-op (never over, an empty map) for a
// scenario with no terrain, or whose deployment zones don't name an
// own town (e.g. the Starter Battle).
func CheckCapture(state GameState, scenario rules.Scenario) (Outcome, map[string]string) {
	pending := map[string]string{}
	if scenario.Terrain == nil {
		return Outcome{}, pending
	}
	var outcome Outcome
	for _, side := range []string{"north", "south"} {
		enemy := otherSide(side)
		enemyTown := scenario.DeploymentZones[enemy].NearTown
		if enemyTown == "" {
			continue
		}
		townHexes := scenario.Terrain.Towns[enemyTown]
		if !anyUnitIn(state.Board, side, townHexes) || anyUnitIn(state.Board, enemy, townHexes) {
			continue
		}
		pending[enemyTown] = side
		if !outcome.Over && state.PendingCapture[enemyTown] == side {
			outcome = Outcome{Over: true, Winner: side, Reason: fmt.Sprintf("captured %s and held it", enemyTown)}
		}
	}
	return outcome, pending
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
