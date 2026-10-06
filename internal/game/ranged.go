package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// RangedHitCheck is the result of comparing the shooter's and target's
// totals (docs/core-rules.md section 9).
type RangedHitCheck struct {
	ShooterTotal, TargetTotal int
	Hit                       bool
}

// RangedDamageCheck is the result of comparing the shooter's RngDmg against
// the target's Def (docs/core-rules.md section 9).
type RangedDamageCheck struct {
	RngDmg, Def int
	Hit         bool
}

// RangedResult is the full, auditable record of one ranged attack
// (docs/core-rules.md section 9).
type RangedResult struct {
	ShooterID, TargetID string
	InRange, InArc      bool
	ShooterSupport      int
	TargetSupport       int

	// HitCheck and Damage are the zero value if the target was out of
	// range or outside the firing arc: nothing happens in that case
	// (section 6.4).
	HitCheck RangedHitCheck
	Damage   RangedDamageCheck
}

// ResolveRanged resolves one ranged attack from shooterID at targetID
// (docs/core-rules.md section 9). It does not check whether the shooter is
// otherwise able to fire (e.g. artillery must be Ready); callers checking
// an order's preconditions should do that first.
func ResolveRanged(board Board, core rules.CoreRules, shooterID, targetID string) (RangedResult, error) {
	shooter, ok := board.Unit(shooterID)
	if !ok {
		return RangedResult{}, fmt.Errorf("game: unknown unit %q", shooterID)
	}
	target, ok := board.Unit(targetID)
	if !ok {
		return RangedResult{}, fmt.Errorf("game: unknown unit %q", targetID)
	}

	shooterStats := shooter.Stats(core)
	dist := hex.Distance(shooter.Pos, target.Pos)
	result := RangedResult{
		ShooterID: shooterID,
		TargetID:  targetID,
		InRange:   dist >= shooterStats.MinRange && dist <= shooterStats.MaxRange,
		InArc:     hex.InFiringArc(shooter.Pos, shooter.Facing, target.Pos),
	}
	if !result.InRange || !result.InArc {
		return result, nil
	}

	targetStats := target.Stats(core)
	result.ShooterSupport = board.Support(shooter.Side, shooter.Pos)
	result.TargetSupport = board.Support(target.Side, target.Pos)

	shooterTotal := shooterStats.Attack + result.ShooterSupport
	targetTotal := targetStats.Def + result.TargetSupport
	hit := shooterTotal > targetTotal
	result.HitCheck = RangedHitCheck{ShooterTotal: shooterTotal, TargetTotal: targetTotal, Hit: hit}
	if !hit {
		return result, nil
	}

	damageHit := shooterStats.RngDmg > targetStats.Def
	result.Damage = RangedDamageCheck{RngDmg: shooterStats.RngDmg, Def: targetStats.Def, Hit: damageHit}
	return result, nil
}
