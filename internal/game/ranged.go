package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// RangedResult is the full, auditable record of one ranged attack
// (docs/core-rules.md section 9): one comparison, whose margin decides
// the result. There is no position bonus and no knockback.
type RangedResult struct {
	ShooterID, TargetID string
	InRange, InArc      bool
	ShooterSupport      int
	TargetSupport       int

	// ShooterBase/TargetBase/ShooterTotal/TargetTotal/Margin/Hits/
	// Destroyed are all the zero value if the target was out of range or
	// outside the firing arc: nothing happens in that case (section 6.4).
	ShooterBase, TargetBase   int
	ShooterTotal, TargetTotal int
	Margin                    int

	// Hits is how many hits the target takes: 0 (miss), 1 (margin 1-2),
	// or 2 (margin 3+, which destroys any unit outright).
	Hits int
	// Destroyed is true if Hits is 2, or Hits is 1 and the target was
	// already at half strength (docs/core-rules.md section 3.2).
	Destroyed bool
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
	result.ShooterBase = shooterStats.Attack
	result.TargetBase = targetStats.Def

	shooterTotal := shooterStats.Attack + result.ShooterSupport
	targetTotal := targetStats.Def + result.TargetSupport
	result.ShooterTotal, result.TargetTotal = shooterTotal, targetTotal
	result.Margin = shooterTotal - targetTotal

	if result.Margin <= 0 {
		return result, nil
	}
	result.Hits = 1
	if result.Margin >= 3 {
		result.Hits = 2
	}
	if result.Hits == 2 || target.Strength == Half {
		result.Destroyed = true
	}
	return result, nil
}
