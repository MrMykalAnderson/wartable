package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PositionBonus is the melee attacker's bonus for which of the defender's
// edges it attacks across (docs/core-rules.md section 8.2).
type PositionBonus struct {
	Front int `yaml:"front"`
	Flank int `yaml:"flank"`
	Rear  int `yaml:"rear"`
}

// CoreRules holds the tunable numbers behind the core rules (data/rules/core.yaml).
type CoreRules struct {
	PositionBonus       PositionBonus `yaml:"position_bonus"`
	AmbushDefPenalty    int           `yaml:"ambush_def_penalty"`
	HalfStrengthPenalty int           `yaml:"half_strength_penalty"`
	SupportPerAlly      int           `yaml:"support_per_ally"`
	DirectionOrder      []string      `yaml:"direction_order"`
}

func (r CoreRules) validate() error {
	if r.PositionBonus.Front < 0 || r.PositionBonus.Flank < 0 || r.PositionBonus.Rear < 0 {
		return fmt.Errorf("position_bonus must not be negative")
	}
	if r.AmbushDefPenalty < 0 || r.HalfStrengthPenalty < 0 || r.SupportPerAlly < 0 {
		return fmt.Errorf("penalties and bonuses must not be negative")
	}
	want := []string{"N", "NE", "SE", "S", "SW", "NW"}
	if len(r.DirectionOrder) != len(want) {
		return fmt.Errorf("direction_order must list all six directions, got %v", r.DirectionOrder)
	}
	for i, d := range want {
		if r.DirectionOrder[i] != d {
			return fmt.Errorf("direction_order must be clockwise from N (%v), got %v", want, r.DirectionOrder)
		}
	}
	return nil
}

// LoadCoreRules reads and validates data/rules/core.yaml.
func LoadCoreRules(path string) (CoreRules, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CoreRules{}, fmt.Errorf("rules: %w", err)
	}
	var r CoreRules
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return CoreRules{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	if err := r.validate(); err != nil {
		return CoreRules{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	return r, nil
}
