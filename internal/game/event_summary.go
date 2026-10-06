package game

import (
	"fmt"
	"strings"
)

// Summary renders an event as a human-readable line (docs/dev-plan.md
// section 4), including the numbers behind a melee or ranged attack. It's
// shared by the CLI and the web viewer.
func (e Event) Summary() string {
	switch {
	case e.Kind == "barrage" && e.Ranged != nil:
		return "[barrage] " + strings.TrimPrefix(e.Ranged.Summary(), "[ranged] ")
	case e.Melee != nil:
		return e.Melee.Summary()
	case e.Ranged != nil:
		return e.Ranged.Summary()
	default:
		return fmt.Sprintf("[%s] %s: %s", e.Kind, e.Unit, e.Detail)
	}
}

// Summary renders a melee result as a human-readable line, e.g. "1st
// Horse (3 +3 rear) vs 2nd Foot (2): margin 4, destroyed" (docs/dev-
// plan.md section 7.5).
func (m MeleeResult) Summary() string {
	return fmt.Sprintf(
		"[melee] %s (%s) vs %s (%s): margin %d, %s",
		m.AttackerID, m.attackerBreakdown(), m.DefenderID, m.defenderBreakdown(), m.Margin, m.outcome(),
	)
}

func (m MeleeResult) attackerBreakdown() string {
	s := fmt.Sprintf("%d", m.AttackerBase)
	if m.AttackerSupport > 0 {
		s += fmt.Sprintf(" +%d support", m.AttackerSupport)
	}
	if m.PositionBonus > 0 {
		s += fmt.Sprintf(" +%d %s", m.PositionBonus, strings.ToLower(m.Edge.String()))
	}
	return s
}

func (m MeleeResult) defenderBreakdown() string {
	s := fmt.Sprintf("%d", m.DefenderBase)
	if m.DefenderSupport > 0 {
		s += fmt.Sprintf(" +%d support", m.DefenderSupport)
	}
	if m.AmbushPenalty > 0 {
		s += fmt.Sprintf(" -%d ambush", m.AmbushPenalty)
	}
	return s
}

func (m MeleeResult) outcome() string {
	switch {
	case m.LoserDestroyed:
		return "destroyed"
	case m.Hits == 0:
		return "repelled" + m.knockbackSuffix()
	default:
		return "hit" + m.knockbackSuffix()
	}
}

func (m MeleeResult) knockbackSuffix() string {
	switch {
	case m.Knockback == nil:
		return ""
	case m.Knockback.Destroyed:
		return fmt.Sprintf(" (couldn't retreat to %s: destroyed)", m.Knockback.To)
	default:
		return fmt.Sprintf(", knocked back to %s", m.Knockback.To)
	}
}

// Summary renders a ranged result as a human-readable line.
func (r RangedResult) Summary() string {
	if !r.InRange || !r.InArc {
		return fmt.Sprintf("[ranged] %s fires at %s: out of range or arc (in range=%v, in arc=%v)", r.ShooterID, r.TargetID, r.InRange, r.InArc)
	}
	return fmt.Sprintf(
		"[ranged] %s (%s) vs %s (%s): margin %d, %s",
		r.ShooterID, r.shooterBreakdown(), r.TargetID, r.targetBreakdown(), r.Margin, r.outcome(),
	)
}

func (r RangedResult) shooterBreakdown() string {
	s := fmt.Sprintf("%d", r.ShooterBase)
	if r.ShooterSupport > 0 {
		s += fmt.Sprintf(" +%d support", r.ShooterSupport)
	}
	return s
}

func (r RangedResult) targetBreakdown() string {
	s := fmt.Sprintf("%d", r.TargetBase)
	if r.TargetSupport > 0 {
		s += fmt.Sprintf(" +%d support", r.TargetSupport)
	}
	return s
}

func (r RangedResult) outcome() string {
	switch {
	case r.Hits == 0:
		return "miss"
	case r.Destroyed:
		return "destroyed"
	default:
		return "hit"
	}
}
