package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Def is a unit's Defence stat: either a single flat value, or a value per
// artillery-style state (docs/core-rules.md section 3.3).
type Def struct {
	Flat    int
	ByState map[string]int
}

func (d *Def) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Decode(&d.Flat)
	case yaml.MappingNode:
		return value.Decode(&d.ByState)
	default:
		return fmt.Errorf("rules: def must be a number or a map of state to number, got %v", value.Tag)
	}
}

// Value returns the Def for the given state. For units with a flat Def
// (ByState is nil), state is ignored.
func (d Def) Value(state string) int {
	if d.ByState != nil {
		return d.ByState[state]
	}
	return d.Flat
}

// UnitState describes what a unit in a particular state (e.g. artillery's
// "ready" and "mobilised") is able to do (docs/core-rules.md section 3.3).
type UnitState struct {
	CanMove bool `yaml:"can_move"`
	CanFire bool `yaml:"can_fire"`
	CanTurn bool `yaml:"can_turn"`
}

// Unit is a unit template's full stat block (docs/core-rules.md section
// 3.1). Range is written as a band, MinRange to Range (the maximum); a
// unit with no ranged attack has Range 0.
type Unit struct {
	ID          string               `yaml:"id"`
	Name        string               `yaml:"name"`
	Cost        int                  `yaml:"cost"`
	Move        int                  `yaml:"move"`
	Def         Def                  `yaml:"def"`
	MinRange    int                  `yaml:"min_range"`
	Range       int                  `yaml:"range"`
	RngDmg      int                  `yaml:"rng_dmg"`
	Attack      int                  `yaml:"attack"`
	AttDmg      int                  `yaml:"att_dmg"`
	Melee       bool                 `yaml:"melee"`
	States      map[string]UnitState `yaml:"states,omitempty"`
	DeployState string               `yaml:"deploy_state,omitempty"`
}

func (u Unit) validate() error {
	if u.ID == "" {
		return fmt.Errorf("unit %q: id is required", u.Name)
	}
	if u.Name == "" {
		return fmt.Errorf("unit %q: name is required", u.ID)
	}
	if u.Cost < 0 || u.Move < 0 || u.Range < 0 || u.MinRange < 0 || u.RngDmg < 0 || u.Attack < 0 || u.AttDmg < 0 {
		return fmt.Errorf("unit %q: stats must not be negative", u.ID)
	}
	if u.Range > 0 && u.MinRange > u.Range {
		return fmt.Errorf("unit %q: min_range %d is greater than range %d", u.ID, u.MinRange, u.Range)
	}
	if u.Def.ByState == nil && u.Def.Flat < 0 {
		return fmt.Errorf("unit %q: def must not be negative", u.ID)
	}
	for state, def := range u.Def.ByState {
		if def < 0 {
			return fmt.Errorf("unit %q: def for state %q must not be negative", u.ID, state)
		}
	}
	if len(u.States) > 0 {
		if u.Def.ByState == nil {
			return fmt.Errorf("unit %q: has states but def is not given per state", u.ID)
		}
		for state := range u.States {
			if _, ok := u.Def.ByState[state]; !ok {
				return fmt.Errorf("unit %q: state %q has no def", u.ID, state)
			}
		}
		if _, ok := u.States[u.DeployState]; !ok {
			return fmt.Errorf("unit %q: deploy_state %q is not one of its states", u.ID, u.DeployState)
		}
	}
	return nil
}

// LoadUnits reads and validates a unit templates file such as
// data/units/standard.yaml, keyed by unit ID.
func LoadUnits(path string) (map[string]Unit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("rules: %w", err)
	}
	var list []Unit
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("rules: %s: %w", path, err)
	}
	units := make(map[string]Unit, len(list))
	for _, u := range list {
		if u.Range > 0 && u.MinRange == 0 {
			u.MinRange = 1 // default (docs/dev-plan.md section 7.3).
		}
		if err := u.validate(); err != nil {
			return nil, fmt.Errorf("rules: %s: %w", path, err)
		}
		if _, dup := units[u.ID]; dup {
			return nil, fmt.Errorf("rules: %s: duplicate unit id %q", path, u.ID)
		}
		units[u.ID] = u
	}
	return units, nil
}
