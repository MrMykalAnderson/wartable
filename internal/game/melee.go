package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// Knockback is the result of pushing a melee combat's loser back
// (docs/core-rules.md section 8.3).
type Knockback struct {
	To        hex.Offset
	Destroyed bool // the loser couldn't retreat and was destroyed instead.
}

// MeleeResult is the full, auditable record of one melee combat
// (docs/core-rules.md section 8.3): one comparison, whose margin decides
// the result.
type MeleeResult struct {
	AttackerID, DefenderID string
	Edge                   hex.Position // the defender's edge attacked.
	PositionBonus          int
	AttackerSupport        int
	DefenderSupport        int
	Ambushed               bool
	AmbushPenalty          int // the penalty actually applied (0 unless Ambushed).

	AttackerBase, DefenderBase   int // raw Attack/Def, before support/bonus/penalty.
	AttackerTotal, DefenderTotal int
	Margin                       int

	// WinnerID/LoserID: the repelled attacker (margin <= 0) or the hit
	// defender (margin >= 1).
	WinnerID, LoserID string

	// Hits is how many hits the loser takes: 0 (repelled, no damage), 1
	// (margin 1-2), or 2 (margin 3+, which destroys any unit outright).
	Hits int

	// LoserDestroyed is true if the loser ends up destroyed without a
	// knockback being attempted: Hits is 2, the loser was already at
	// half strength (docs/core-rules.md section 3.2), or the loser is
	// dug in (Ready artillery, section 3.4: never knocked back, any
	// margin of loss destroys it).
	LoserDestroyed bool
	Knockback      *Knockback // nil if LoserDestroyed is true.
}

func positionBonusFor(core rules.CoreRules, edge hex.Position) int {
	switch edge {
	case hex.Flank:
		return core.PositionBonus.Flank
	case hex.Rear:
		return core.PositionBonus.Rear
	default:
		return core.PositionBonus.Front
	}
}

// ResolveMelee resolves one melee combat between attackerID and defenderID
// (docs/core-rules.md section 8.3). ambushed marks the defender as
// ambushed (section 7.4): the ambushed unit always takes the defender's
// role.
func ResolveMelee(board Board, core rules.CoreRules, attackerID, defenderID string, ambushed bool) (MeleeResult, error) {
	attacker, ok := board.Unit(attackerID)
	if !ok {
		return MeleeResult{}, fmt.Errorf("game: unknown unit %q", attackerID)
	}
	defender, ok := board.Unit(defenderID)
	if !ok {
		return MeleeResult{}, fmt.Errorf("game: unknown unit %q", defenderID)
	}
	edge, ok := hex.EdgeAttacked(defender.Facing, defender.Pos, attacker.Pos)
	if !ok {
		return MeleeResult{}, fmt.Errorf("game: %q is not adjacent to %q", attackerID, defenderID)
	}

	attackerStats := attacker.Stats(core)
	defenderStats := defender.Stats(core)
	attackerSupport := board.Support(attacker.Side, attacker.Pos)
	defenderSupport := board.Support(defender.Side, defender.Pos)
	positionBonus := positionBonusFor(core, edge)
	ambushPenalty := 0
	if ambushed {
		ambushPenalty = core.AmbushDefPenalty
	}

	attackerTotal := attackerStats.Attack + attackerSupport + positionBonus
	defenderTotal := defenderStats.Def + defenderSupport - ambushPenalty
	margin := attackerTotal - defenderTotal

	result := MeleeResult{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		Edge:            edge,
		PositionBonus:   positionBonus,
		AttackerSupport: attackerSupport,
		DefenderSupport: defenderSupport,
		Ambushed:        ambushed,
		AmbushPenalty:   ambushPenalty,
		AttackerBase:    attackerStats.Attack,
		DefenderBase:    defenderStats.Def,
		AttackerTotal:   attackerTotal,
		DefenderTotal:   defenderTotal,
		Margin:          margin,
	}

	if margin <= 0 {
		// Repelled: ties go to the defender. No damage; the attacker is
		// knocked back.
		result.WinnerID, result.LoserID = defenderID, attackerID
		result.Knockback = resolveKnockback(board, defender, attacker)
		return result, nil
	}

	result.WinnerID, result.LoserID = attackerID, defenderID
	result.Hits = 1
	if margin >= 3 {
		result.Hits = 2
	}
	if result.Hits == 2 || defender.Strength == Half || defender.DugIn() {
		result.LoserDestroyed = true
		return result, nil
	}
	result.Knockback = resolveKnockback(board, attacker, defender)
	return result, nil
}

func resolveKnockback(board Board, winner, loser UnitInstance) *Knockback {
	to := hex.Knockback(winner.Pos, loser.Pos)
	if board.CanRetreatTo(loser.Side, to) {
		return &Knockback{To: to}
	}
	return &Knockback{To: to, Destroyed: true}
}
