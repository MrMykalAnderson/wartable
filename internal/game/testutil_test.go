package game

import (
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

const (
	unitsPath     = "../../data/units/standard.yaml"
	coreRulesPath = "../../data/rules/core.yaml"
)

func loadTestRules(t *testing.T) (map[string]rules.Unit, rules.CoreRules) {
	t.Helper()
	units, err := rules.LoadUnits(unitsPath)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	core, err := rules.LoadCoreRules(coreRulesPath)
	if err != nil {
		t.Fatalf("LoadCoreRules: %v", err)
	}
	return units, core
}

// newUnit builds a full-strength UnitInstance at the hex named pos (e.g.
// "F6"), facing dir. Artillery starts in its deploy_state; override
// u.State for a ready gun.
func newUnit(t *testing.T, templates map[string]rules.Unit, id, side, templateID, pos string, facing hex.Direction) UnitInstance {
	t.Helper()
	tmpl, ok := templates[templateID]
	if !ok {
		t.Fatalf("newUnit: unknown template %q", templateID)
	}
	p, err := hex.ParseOffset(pos)
	if err != nil {
		t.Fatalf("newUnit: %v", err)
	}
	return UnitInstance{
		ID:       id,
		Side:     side,
		Template: tmpl,
		Pos:      p,
		Facing:   facing,
		Strength: Full,
		State:    tmpl.DeployState,
	}
}

// starterBoard builds a Board the size of the Starter Battle map
// (docs/starter-battle.md) containing the given units.
func starterBoard(units ...UnitInstance) Board {
	return Board{Columns: 12, Rows: 10, Units: units}
}
