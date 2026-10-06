package game

import "github.com/MrMykalAnderson/wartable/internal/rules"

// applyMelee applies a resolved melee combat's outcome to the board: the
// loser takes a hit (if any), and is then knocked back, destroyed, or
// (if it was already half strength) destroyed outright
// (docs/core-rules.md section 8.3).
func applyMelee(board Board, r MeleeResult) Board {
	if r.LoserDestroyed {
		return board.WithoutUnit(r.LoserID)
	}
	loser, ok := board.Unit(r.LoserID)
	if !ok {
		return board
	}
	if r.Damage.Hit {
		loser, _ = loser.TakeHit()
	}
	if r.Knockback != nil {
		if r.Knockback.Destroyed {
			return board.WithoutUnit(loser.ID)
		}
		loser.Pos = r.Knockback.To
	}
	return board.WithUnit(loser)
}

// applyRanged applies a resolved ranged attack's outcome to the board: the
// target takes a hit, if any (docs/core-rules.md section 9; there is no
// knockback from ranged attacks).
func applyRanged(board Board, r RangedResult) Board {
	if !r.Damage.Hit {
		return board
	}
	target, ok := board.Unit(r.TargetID)
	if !ok {
		return board
	}
	next, destroyed := target.TakeHit()
	if destroyed {
		return board.WithoutUnit(target.ID)
	}
	return board.WithUnit(next)
}

// resolveAmbush resolves an ambush (docs/core-rules.md section 7.4): each
// enemy currently adjacent to the ambushed unit attacks it in melee, one
// at a time in clockwise order from N, rechecking adjacency after each
// combat (an attacker knocked out of adjacency doesn't get its attack),
// and stopping once the ambushed unit is destroyed.
func resolveAmbush(board Board, core rules.CoreRules, ambushedID string) (Board, []MeleeResult) {
	var results []MeleeResult
	for {
		ambushed, ok := board.Unit(ambushedID)
		if !ok {
			return board, results
		}
		enemies := board.AdjacentEnemies(ambushed.Side, ambushed.Pos)
		if len(enemies) == 0 {
			return board, results
		}
		result, err := ResolveMelee(board, core, enemies[0].ID, ambushedID, true)
		if err != nil {
			return board, results
		}
		results = append(results, result)
		board = applyMelee(board, result)
	}
}
