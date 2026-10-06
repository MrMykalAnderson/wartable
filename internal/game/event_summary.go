package game

import "fmt"

// Summary renders an event as a human-readable line (docs/dev-plan.md
// section 4), including the numbers behind a melee or ranged attack. It's
// shared by the CLI and the web viewer.
func (e Event) Summary() string {
	switch {
	case e.Melee != nil:
		return e.Melee.Summary()
	case e.Ranged != nil:
		return e.Ranged.Summary()
	default:
		return fmt.Sprintf("[%s] %s: %s", e.Kind, e.Unit, e.Detail)
	}
}

// Summary renders a melee result as a human-readable line.
func (m MeleeResult) Summary() string {
	return fmt.Sprintf(
		"[melee] %s attacks %s's %s: hit check %d/%d (attacker wins=%v); damage %d/%d (hit=%v)%s",
		m.AttackerID, m.DefenderID, m.Edge,
		m.Hit.AttackerTotal, m.Hit.DefenderTotal, m.Hit.AttackerWins,
		m.Damage.Damage, m.Damage.Def, m.Damage.Hit,
		m.knockbackSuffix(),
	)
}

func (m MeleeResult) knockbackSuffix() string {
	switch {
	case m.LoserDestroyed:
		return fmt.Sprintf("; %s destroyed (already half strength)", m.LoserID)
	case m.Knockback == nil:
		return ""
	case m.Knockback.Destroyed:
		return fmt.Sprintf("; %s destroyed (couldn't retreat to %s)", m.LoserID, m.Knockback.To)
	default:
		return fmt.Sprintf("; %s knocked back to %s", m.LoserID, m.Knockback.To)
	}
}

// Summary renders a ranged result as a human-readable line.
func (r RangedResult) Summary() string {
	if !r.InRange || !r.InArc {
		return fmt.Sprintf("[ranged] %s fires at %s: out of range or arc (in range=%v, in arc=%v)", r.ShooterID, r.TargetID, r.InRange, r.InArc)
	}
	return fmt.Sprintf(
		"[ranged] %s fires at %s: hit check %d/%d (hit=%v); damage %d/%d (hit=%v)",
		r.ShooterID, r.TargetID,
		r.HitCheck.ShooterTotal, r.HitCheck.TargetTotal, r.HitCheck.Hit,
		r.Damage.RngDmg, r.Damage.Def, r.Damage.Hit,
	)
}
