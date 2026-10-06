// Package game implements Wartable's combat and (eventually) turn
// execution: a pure engine over the rules data loaded by internal/rules
// and the coordinate geometry in internal/hex.
package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// Strength is whether a unit is at full or half strength
// (docs/core-rules.md section 3.2).
type Strength int

const (
	Full Strength = iota
	Half
)

// UnitInstance is a unit on the map: a reference to its template plus the
// state that changes during play.
type UnitInstance struct {
	ID       string
	Side     string
	Template rules.Unit
	Pos      hex.Offset
	Facing   hex.Direction
	Strength Strength
	// State is the unit's current state (e.g. artillery's "ready" or
	// "mobilised"). Empty for units without states.
	State string
}

// Stats is a unit's effective stat block: its template stats, resolved for
// its current state and strength (docs/core-rules.md sections 3.1-3.2).
// MinRange and MaxRange are the ranged-attack band: a unit can only fire
// at a target between the two, inclusive.
type Stats struct {
	Def, Move, MinRange, MaxRange, RngDmg, Attack, AttDmg int
}

// Stats computes the unit's effective stats. A half-strength unit has -1
// to every stat (floored at 0), except MinRange, which is unaffected
// (docs/core-rules.md section 3.2); Def is resolved for the unit's state
// first.
func (u UnitInstance) Stats(core rules.CoreRules) Stats {
	s := Stats{
		Def:      u.Template.Def.Value(u.State),
		Move:     u.Template.Move,
		MinRange: u.Template.MinRange,
		MaxRange: u.Template.Range,
		RngDmg:   u.Template.RngDmg,
		Attack:   u.Template.Attack,
		AttDmg:   u.Template.AttDmg,
	}
	if u.Strength == Half {
		p := core.HalfStrengthPenalty
		s.Def = max(0, s.Def-p)
		s.Move = max(0, s.Move-p)
		s.MaxRange = max(0, s.MaxRange-p)
		s.RngDmg = max(0, s.RngDmg-p)
		s.Attack = max(0, s.Attack-p)
		s.AttDmg = max(0, s.AttDmg-p)
	}
	return s
}

// TakeHit applies one hit (docs/core-rules.md section 3.2): a full-strength
// unit drops to half strength; a half-strength unit is destroyed.
func (u UnitInstance) TakeHit() (next UnitInstance, destroyed bool) {
	if u.Strength == Half {
		return u, true
	}
	u.Strength = Half
	return u, false
}

// CanMove, CanFire and CanTurn report what a unit can currently do, given
// its state (docs/core-rules.md section 3.3). Units without states (e.g.
// infantry, cavalry) can always do all three. Exported for the web
// viewer's order-preview API (cmd/wartable-web), which must reuse these
// rather than re-deriving them in JavaScript.
func (u UnitInstance) CanMove() bool {
	s, ok := u.Template.States[u.State]
	return !ok || s.CanMove
}

func (u UnitInstance) CanFire() bool {
	s, ok := u.Template.States[u.State]
	return !ok || s.CanFire
}

func (u UnitInstance) CanTurn() bool {
	s, ok := u.Template.States[u.State]
	return !ok || s.CanTurn
}
