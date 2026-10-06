package game

import (
	"fmt"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// TestAppendixACombatTables checks docs/core-rules.md's Appendix A against
// the engine for every matchup it lists, so the two can never silently
// disagree (docs/dev-plan.md section 7.5). The net modifier column is
// realized purely via support (attacker support for positive modifiers,
// defender support for negative ones): half-strength and ambush
// modifiers are already covered by their own, more specific tests, and
// mixing them in here would also trigger the "a half-strength unit that
// is hit is destroyed" nuance, which Appendix A's table deliberately
// doesn't encode (it labels the margin tier only).
func TestAppendixACombatTables(t *testing.T) {
	templates, core := loadTestRules(t)

	type defenderSpec struct {
		name     string
		template string
		state    string // artillery only.
		def      int
	}
	infantryDef := defenderSpec{"Infantry", "infantry", "", 2}
	cavalryDef := defenderSpec{"Cavalry", "cavalry", "", 3}
	artilleryReady := defenderSpec{"Artillery (Ready)", "artillery", "ready", 4}
	artilleryMobilised := defenderSpec{"Artillery (Mobilised)", "artillery", "mobilised", 2}

	const (
		R = "Repelled"
		H = "Hit"
		D = "Destroyed"
	)

	// Columns are net modifiers -2, -1, 0, +1, +2, +3, matching Appendix
	// A's tables exactly.
	type row struct {
		defender defenderSpec
		edge     hex.Position
		results  [6]string
	}

	infantryRows := []row{
		{infantryDef, hex.Front, [6]string{R, R, R, H, H, D}},
		{infantryDef, hex.Flank, [6]string{R, R, H, H, D, D}},
		{infantryDef, hex.Rear, [6]string{H, H, D, D, D, D}},
		{cavalryDef, hex.Front, [6]string{R, R, R, R, H, H}},
		{cavalryDef, hex.Flank, [6]string{R, R, R, H, H, D}},
		{cavalryDef, hex.Rear, [6]string{R, H, H, D, D, D}},
		{artilleryReady, hex.Front, [6]string{R, R, R, R, R, H}},
		{artilleryReady, hex.Flank, [6]string{R, R, R, R, H, H}},
		{artilleryReady, hex.Rear, [6]string{R, R, H, H, D, D}},
		{artilleryMobilised, hex.Front, [6]string{R, R, R, H, H, D}},
		{artilleryMobilised, hex.Flank, [6]string{R, R, H, H, D, D}},
		{artilleryMobilised, hex.Rear, [6]string{H, H, D, D, D, D}},
	}
	cavalryRows := []row{
		{infantryDef, hex.Front, [6]string{R, R, H, H, D, D}},
		{infantryDef, hex.Flank, [6]string{R, H, H, D, D, D}},
		{infantryDef, hex.Rear, [6]string{H, D, D, D, D, D}},
		{cavalryDef, hex.Front, [6]string{R, R, R, H, H, D}},
		{cavalryDef, hex.Flank, [6]string{R, R, H, H, D, D}},
		{cavalryDef, hex.Rear, [6]string{H, H, D, D, D, D}},
		{artilleryReady, hex.Front, [6]string{R, R, R, R, H, H}},
		{artilleryReady, hex.Flank, [6]string{R, R, R, H, H, D}},
		{artilleryReady, hex.Rear, [6]string{R, H, H, D, D, D}},
		{artilleryMobilised, hex.Front, [6]string{R, R, H, H, D, D}},
		{artilleryMobilised, hex.Flank, [6]string{R, H, H, D, D, D}},
		{artilleryMobilised, hex.Rear, [6]string{H, D, D, D, D, D}},
	}

	check := func(t *testing.T, attackerName string, attackerTemplate string, attackerAttack int, rows []row) {
		for _, r := range rows {
			for i, modifier := range []int{-2, -1, 0, 1, 2, 3} {
				name := fmt.Sprintf("%s_vs_%s_%s_mod%+d", attackerName, r.defender.name, r.edge, modifier)
				t.Run(name, func(t *testing.T) {
					got := meleeOutcomeAtModifier(t, templates, core, attackerTemplate, r.defender.template, r.defender.state, r.edge, modifier)
					if got != r.results[i] {
						t.Errorf("%s attacking %s's %s at modifier %+d = %s, want %s (Appendix A)", attackerName, r.defender.name, r.edge, modifier, got, r.results[i])
					}
				})
			}
		}
	}

	t.Run("Infantry_attacking", func(t *testing.T) { check(t, "Infantry", "infantry", 2, infantryRows) })
	t.Run("Cavalry_attacking", func(t *testing.T) { check(t, "Cavalry", "cavalry", 3, cavalryRows) })

	// Ranged attacks: Cavalry or Artillery, both Attack 3; no position
	// bonus, facing doesn't matter.
	rangedRows := []struct {
		defender defenderSpec
		results  [6]string
	}{
		{infantryDef, [6]string{"Miss", "Miss", H, H, D, D}},
		{cavalryDef, [6]string{"Miss", "Miss", "Miss", H, H, D}},
		{artilleryReady, [6]string{"Miss", "Miss", "Miss", "Miss", H, H}},
		{artilleryMobilised, [6]string{"Miss", "Miss", H, H, D, D}},
	}
	t.Run("Ranged_attacks", func(t *testing.T) {
		for _, r := range rangedRows {
			for i, modifier := range []int{-2, -1, 0, 1, 2, 3} {
				name := fmt.Sprintf("vs_%s_mod%+d", r.defender.name, modifier)
				t.Run(name, func(t *testing.T) {
					got := rangedOutcomeAtModifier(t, templates, core, r.defender.template, r.defender.state, modifier)
					if got != r.results[i] {
						t.Errorf("ranged attack on %s at modifier %+d = %s, want %s (Appendix A)", r.defender.name, modifier, got, r.results[i])
					}
				})
			}
		}
	})
}

// meleeOutcomeAtModifier builds a defender (with the given edge attacked)
// and realizes modifier purely via support, then returns the engine's
// outcome label.
func meleeOutcomeAtModifier(t *testing.T, templates map[string]rules.Unit, core rules.CoreRules, attackerTemplate, defenderTemplate, defenderState string, edge hex.Position, modifier int) string {
	t.Helper()
	center := hex.Offset{Col: 10, Row: 10}
	defenderFacing := hex.N
	var attackerDir hex.Direction
	switch edge {
	case hex.Front:
		attackerDir = defenderFacing
	case hex.Flank:
		attackerDir = defenderFacing.RotateCW(1)
	default:
		attackerDir = defenderFacing.Opposite()
	}
	attackerPos := center.Neighbor(attackerDir)

	defender := UnitInstance{ID: "Defender", Side: "north", Template: templates[defenderTemplate], Pos: center, Facing: defenderFacing, Strength: Full, State: defenderState}
	attacker := UnitInstance{ID: "Attacker", Side: "south", Template: templates[attackerTemplate], Pos: attackerPos, Facing: defenderFacing.Opposite(), Strength: Full, State: templates[attackerTemplate].DeployState}

	units := []UnitInstance{defender, attacker}
	if modifier > 0 {
		units = append(units, supportUnits(templates, attackerPos, center, "south", modifier)...)
	} else if modifier < 0 {
		units = append(units, supportUnits(templates, center, attackerPos, "north", -modifier)...)
	}
	board := Board{Columns: 20, Rows: 20, Units: units}

	result, err := ResolveMelee(board, core, "Attacker", "Defender", false)
	if err != nil {
		t.Fatalf("ResolveMelee: %v", err)
	}
	switch {
	case result.Hits == 0:
		return "Repelled"
	case result.LoserDestroyed:
		return "Destroyed"
	default:
		return "Hit"
	}
}

func rangedOutcomeAtModifier(t *testing.T, templates map[string]rules.Unit, core rules.CoreRules, defenderTemplate, defenderState string, modifier int) string {
	t.Helper()
	center := hex.Offset{Col: 10, Row: 10}
	shooterPos := center.Neighbor(hex.N)

	defender := UnitInstance{ID: "Defender", Side: "north", Template: templates[defenderTemplate], Pos: center, Facing: hex.N, Strength: Full, State: defenderState}
	shooter := UnitInstance{ID: "Shooter", Side: "south", Template: templates["cavalry"], Pos: shooterPos, Facing: hex.S, Strength: Full, State: templates["cavalry"].DeployState}

	units := []UnitInstance{defender, shooter}
	if modifier > 0 {
		units = append(units, supportUnits(templates, shooterPos, center, "south", modifier)...)
	} else if modifier < 0 {
		units = append(units, supportUnits(templates, center, shooterPos, "north", -modifier)...)
	}
	board := Board{Columns: 20, Rows: 20, Units: units}

	result, err := ResolveRanged(board, core, "Shooter", "Defender")
	if err != nil {
		t.Fatalf("ResolveRanged: %v", err)
	}
	switch {
	case result.Hits == 0:
		return "Miss"
	case result.Destroyed:
		return "Destroyed"
	default:
		return "Hit"
	}
}

// supportUnits places up to n full-strength infantry of side on around's
// other neighbours (excluding exclude), for realizing a support count.
func supportUnits(templates map[string]rules.Unit, around, exclude hex.Offset, side string, n int) []UnitInstance {
	var units []UnitInstance
	for _, nb := range around.Neighbors() {
		if len(units) >= n {
			break
		}
		if nb == exclude {
			continue
		}
		units = append(units, UnitInstance{
			ID:       fmt.Sprintf("%s-support-%d", side, len(units)),
			Side:     side,
			Template: templates["infantry"],
			Pos:      nb,
			Facing:   hex.N,
			Strength: Full,
			State:    templates["infantry"].DeployState,
		})
	}
	return units
}
