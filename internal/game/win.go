package game

// Outcome is the result of checking a scenario's win conditions
// (docs/starter-battle.md "Winning").
type Outcome struct {
	Over   bool
	Winner string // "" if the game isn't over, or ended in a draw.
	Reason string
}

// CheckAnnihilation reports whether either side has no units on the map
// and none left to deploy, in which case the other side wins immediately
// (docs/starter-battle.md "Winning", rule 1).
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

// ScoreAtTurnLimit scores each side's surviving units (a half-strength
// unit counts for half its Cost, rounded down) and reports who has the
// higher score (docs/starter-battle.md "Winning", rule 2).
func ScoreAtTurnLimit(state GameState) (northScore, southScore int, outcome Outcome) {
	northScore = score(state, "north")
	southScore = score(state, "south")
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

func score(state GameState, side string) int {
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
	return total
}
