package game

import "github.com/MrMykalAnderson/wartable/internal/rules"

// applyMelee applies a resolved melee combat's outcome to the board
// (docs/core-rules.md section 8.3): the loser takes its hits (if any),
// and is then knocked back, destroyed, or (if LoserDestroyed) removed
// outright without a knockback being attempted.
func applyMelee(board Board, r MeleeResult) Board {
	if r.LoserDestroyed {
		return board.WithoutUnit(r.LoserID)
	}
	loser, ok := board.Unit(r.LoserID)
	if !ok {
		return board
	}
	for i := 0; i < r.Hits; i++ {
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
// target takes its hits, if any (docs/core-rules.md section 9; there is
// no knockback from ranged attacks). Barrage shots are all resolved
// against the same start-of-phase board (docs/core-rules.md section 5.3),
// so r.Destroyed only reflects what that one shot's own margin did; a
// unit already knocked to half strength by an earlier shot in the same
// barrage is still destroyed here by any further hit.
func applyRanged(board Board, r RangedResult) Board {
	if r.Hits == 0 {
		return board
	}
	target, ok := board.Unit(r.TargetID)
	if !ok {
		return board
	}
	destroyed := r.Destroyed
	for i := 0; i < r.Hits && !destroyed; i++ {
		target, destroyed = target.TakeHit()
	}
	if destroyed {
		return board.WithoutUnit(target.ID)
	}
	return board.WithUnit(target)
}

// resolveAmbush resolves an ambush (docs/core-rules.md section 7.4): each
// enemy currently adjacent to the ambushed unit attacks it in melee, one
// at a time in clockwise order from N, rechecking adjacency after each
// combat (an attacker knocked out of adjacency doesn't get its attack),
// and stopping once the ambushed unit is destroyed. Each returned event's
// Board is the board exactly as it stood after that one combat, for
// step-by-step replay.
func resolveAmbush(board Board, core rules.CoreRules, ambushedID string) (Board, []Event) {
	var events []Event
	for {
		ambushed, ok := board.Unit(ambushedID)
		if !ok {
			return board, events
		}
		enemies := board.AdjacentEnemies(ambushed.Side, ambushed.Pos)
		if len(enemies) == 0 {
			return board, events
		}
		result, err := ResolveMelee(board, core, enemies[0].ID, ambushedID, true)
		if err != nil {
			return board, events
		}
		board = applyMelee(board, result)
		events = append(events, Event{Kind: "melee", Unit: result.AttackerID, Detail: "ambush attack", Board: board, Melee: &result})
	}
}
