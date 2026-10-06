package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RowRange is an inclusive, 1-based range of rows, as written in scenario
// docs (e.g. "rows 1-2").
type RowRange struct {
	From, To int
}

func (r *RowRange) UnmarshalYAML(value *yaml.Node) error {
	var pair [2]int
	if err := value.Decode(&pair); err != nil {
		return fmt.Errorf("rules: row range must be [from, to], got %v", value.Tag)
	}
	r.From, r.To = pair[0], pair[1]
	return nil
}

// MapSize is a scenario's map dimensions in hexes.
type MapSize struct {
	Columns int `yaml:"columns"`
	Rows    int `yaml:"rows"`
}

// DeploymentZone is where a side may Deploy units onto the map
// (docs/core-rules.md section 10).
type DeploymentZone struct {
	Rows RowRange `yaml:"rows"`
}

// Scenario is a scenario's settings (data/scenarios/starter-battle.yaml),
// e.g. the Starter Battle (docs/starter-battle.md).
type Scenario struct {
	ID              string                    `yaml:"id"`
	Name            string                    `yaml:"name"`
	Map             MapSize                   `yaml:"map"`
	DeploymentZones map[string]DeploymentZone `yaml:"deployment_zones"`
	DefaultFacing   map[string]string         `yaml:"default_facing"`
	Capacity        int                       `yaml:"capacity"`
	UnitsAllowed    []string                  `yaml:"units_allowed"`
	TurnLimit       int                       `yaml:"turn_limit"`
	TieBreakHolder  string                    `yaml:"tie_break_holder"`
}

func (s Scenario) validate() error {
	if s.ID == "" || s.Name == "" {
		return fmt.Errorf("scenario: id and name are required")
	}
	if s.Map.Columns <= 0 || s.Map.Rows <= 0 {
		return fmt.Errorf("scenario %q: map dimensions must be positive", s.ID)
	}
	if s.Capacity <= 0 {
		return fmt.Errorf("scenario %q: capacity must be positive", s.ID)
	}
	if s.TurnLimit <= 0 {
		return fmt.Errorf("scenario %q: turn_limit must be positive", s.ID)
	}
	if len(s.DeploymentZones) == 0 {
		return fmt.Errorf("scenario %q: at least one deployment zone is required", s.ID)
	}
	for side, dz := range s.DeploymentZones {
		if dz.Rows.From < 1 || dz.Rows.To > s.Map.Rows || dz.Rows.From > dz.Rows.To {
			return fmt.Errorf("scenario %q: deployment zone %q rows %v are outside the %d-row map", s.ID, side, dz.Rows, s.Map.Rows)
		}
		if _, ok := s.DefaultFacing[side]; !ok {
			return fmt.Errorf("scenario %q: side %q has a deployment zone but no default_facing", s.ID, side)
		}
	}
	if _, ok := s.DeploymentZones[s.TieBreakHolder]; !ok {
		return fmt.Errorf("scenario %q: tie_break_holder %q is not a side with a deployment zone", s.ID, s.TieBreakHolder)
	}
	if len(s.UnitsAllowed) == 0 {
		return fmt.Errorf("scenario %q: units_allowed must not be empty", s.ID)
	}
	return nil
}

// LoadScenario reads and validates a scenario file such as
// data/scenarios/starter-battle.yaml. units is used to check that every
// unit named in units_allowed actually exists.
func LoadScenario(path string, units map[string]Unit) (Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, fmt.Errorf("rules: %w", err)
	}
	var s Scenario
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return Scenario{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	if err := s.validate(); err != nil {
		return Scenario{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	for _, id := range s.UnitsAllowed {
		if _, ok := units[id]; !ok {
			return Scenario{}, fmt.Errorf("rules: %s: units_allowed references unknown unit %q", path, id)
		}
	}
	return s, nil
}
