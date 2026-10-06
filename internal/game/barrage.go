package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// BarrageShot is one unit's passive Barrage attack against one enemy
// inside its range band and firing arc (docs/core-rules.md section 3.3).
// Board is the board exactly as it stood immediately after this shot's
// hit (if any) was applied, for step-by-step replay.
type BarrageShot struct {
	Result RangedResult
	Board  Board
}

// ResolveBarragePhase resolves the Execution Phase's passive Barrage step
// (docs/core-rules.md section 5.3): every unit currently able to fire
// from a state (so far, only Ready artillery) fires at every enemy in
// its range band and firing arc, using the positions and stats of board
// as given — the start of the phase. Every shot is computed first, from
// that unmutated board; only then are all the hits applied together, so
// a unit hit by two guns takes two hits, and a unit destroyed by the
// barrage still counts as support for other shots in the same barrage.
func ResolveBarragePhase(board Board, core rules.CoreRules) (Board, []BarrageShot) {
	var shots []BarrageShot
	for _, shooter := range board.Units {
		if len(shooter.Template.States) == 0 || !shooter.CanFire() {
			continue
		}
		stats := shooter.Stats(core)
		if stats.MaxRange == 0 {
			continue
		}
		for _, target := range board.Units {
			if target.Side == shooter.Side {
				continue
			}
			dist := hex.Distance(shooter.Pos, target.Pos)
			if dist < stats.MinRange || dist > stats.MaxRange {
				continue
			}
			if !hex.InFiringArc(shooter.Pos, shooter.Facing, target.Pos) {
				continue
			}
			result, err := ResolveRanged(board, core, shooter.ID, target.ID)
			if err != nil {
				continue
			}
			shots = append(shots, BarrageShot{Result: result})
		}
	}

	for i := range shots {
		board = applyRanged(board, shots[i].Result)
		shots[i].Board = board
	}
	return board, shots
}
