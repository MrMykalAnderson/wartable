package rules

import "testing"

const coreRulesPath = "../../data/rules/core.yaml"

// TestLoadCoreRulesMatchesDocs checks the loaded numbers against
// docs/core-rules.md sections 3.2, 7.2, 7.4 and 8.1-8.2.
func TestLoadCoreRulesMatchesDocs(t *testing.T) {
	r, err := LoadCoreRules(coreRulesPath)
	if err != nil {
		t.Fatalf("LoadCoreRules(%q): %v", coreRulesPath, err)
	}
	want := CoreRules{
		PositionBonus:       PositionBonus{Front: 0, Flank: 1, Rear: 3},
		AmbushDefPenalty:    1,
		HalfStrengthPenalty: 1,
		SupportPerAlly:      1,
		DirectionOrder:      []string{"N", "NE", "SE", "S", "SW", "NW"},
	}
	if r.PositionBonus != want.PositionBonus {
		t.Errorf("PositionBonus = %+v, want %+v", r.PositionBonus, want.PositionBonus)
	}
	if r.AmbushDefPenalty != want.AmbushDefPenalty {
		t.Errorf("AmbushDefPenalty = %d, want %d", r.AmbushDefPenalty, want.AmbushDefPenalty)
	}
	if r.HalfStrengthPenalty != want.HalfStrengthPenalty {
		t.Errorf("HalfStrengthPenalty = %d, want %d", r.HalfStrengthPenalty, want.HalfStrengthPenalty)
	}
	if r.SupportPerAlly != want.SupportPerAlly {
		t.Errorf("SupportPerAlly = %d, want %d", r.SupportPerAlly, want.SupportPerAlly)
	}
	if len(r.DirectionOrder) != len(want.DirectionOrder) {
		t.Fatalf("DirectionOrder = %v, want %v", r.DirectionOrder, want.DirectionOrder)
	}
	for i := range want.DirectionOrder {
		if r.DirectionOrder[i] != want.DirectionOrder[i] {
			t.Errorf("DirectionOrder = %v, want %v", r.DirectionOrder, want.DirectionOrder)
			break
		}
	}
}

func TestLoadCoreRulesRejectsBadDirectionOrder(t *testing.T) {
	path := writeTempFile(t, "core-*.yaml", `
position_bonus: { front: 0, flank: 1, rear: 3 }
ambush_def_penalty: 1
half_strength_penalty: 1
support_per_ally: 1
direction_order: [N, SE, NE, S, SW, NW]
`)
	if _, err := LoadCoreRules(path); err == nil {
		t.Fatalf("LoadCoreRules: want error for out-of-order directions, got none")
	}
}
