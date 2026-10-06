package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// HitCheck is the result of comparing the two sides' totals in a melee's
// hit check (docs/core-rules.md section 8.3, step 1). Ties go to the
// defender.
type HitCheck struct {
	AttackerTotal, DefenderTotal int
	AttackerWins                 bool
}

// DamageCheck is the result of comparing the winner's damage against the
// loser's Def (docs/core-rules.md section 8.3, step 2).
type DamageCheck struct {
	Damage, Def int
	Hit         bool
}

// Knockback is the result of pushing the hit check's loser back
// (docs/core-rules.md section 8.3, step 3).
type Knockback struct {
	To        hex.Offset
	Destroyed bool // the loser couldn't retreat and was destroyed instead.
}

// MeleeResult is the full, auditable record of one melee combat
// (docs/core-rules.md section 8): every number used at each step.
type MeleeResult struct {
	AttackerID, DefenderID string
	Edge                   hex.Position // the defender's edge attacked.
	PositionBonus          int
	AttackerSupport        int
	DefenderSupport        int
	Ambushed               bool

	Hit HitCheck

	// WinnerID/LoserID are the hit check's winner and loser.
	WinnerID, LoserID string
	Damage            DamageCheck

	// LoserDestroyed is true if the loser was already at half strength
	// and so was destroyed by the hit, rather than knocked back.
	LoserDestroyed bool
	Knockback      *Knockback // nil if the loser was destroyed by the hit.
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
// (docs/core-rules.md section 8). ambushed marks the defender as ambushed
// (section 7.4): the ambushed unit always takes the defender's role.
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

	result := MeleeResult{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		Edge:            edge,
		PositionBonus:   positionBonus,
		AttackerSupport: attackerSupport,
		DefenderSupport: defenderSupport,
		Ambushed:        ambushed,
	}

	attackerTotal := attackerStats.Attack + attackerSupport + positionBonus
	defenderTotal := max(0, defenderStats.Def+defenderSupport-ambushPenalty)
	attackerWins := attackerTotal > defenderTotal
	result.Hit = HitCheck{AttackerTotal: attackerTotal, DefenderTotal: defenderTotal, AttackerWins: attackerWins}

	winner, loser := defender, attacker
	result.WinnerID, result.LoserID = defenderID, attackerID
	if attackerWins {
		winner, loser = attacker, defender
		result.WinnerID, result.LoserID = attackerID, defenderID
	}

	var damage, loserDef int
	if attackerWins {
		damage = attackerStats.AttDmg + positionBonus
		loserDef = max(0, defenderStats.Def-ambushPenalty)
	} else {
		damage = defenderStats.AttDmg
		loserDef = attackerStats.Def
	}
	hit := damage > loserDef
	result.Damage = DamageCheck{Damage: damage, Def: loserDef, Hit: hit}

	if !hit {
		result.Knockback = resolveKnockback(board, winner, loser)
		return result, nil
	}

	if loser.Strength == Half {
		result.LoserDestroyed = true
		return result, nil
	}
	result.Knockback = resolveKnockback(board, winner, loser)
	return result, nil
}

func resolveKnockback(board Board, winner, loser UnitInstance) *Knockback {
	to := hex.Knockback(winner.Pos, loser.Pos)
	if board.CanRetreatTo(loser.Side, to) {
		return &Knockback{To: to}
	}
	return &Knockback{To: to, Destroyed: true}
}
