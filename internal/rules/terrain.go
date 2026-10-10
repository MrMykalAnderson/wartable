package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TerrainEffects holds the tunable numbers behind terrain's effects on
// movement and combat (docs/core-rules.md section 2.3;
// data/rules/terrain.yaml).
type TerrainEffects struct {
	ForestDefBonus       int `yaml:"forest_def_bonus"`
	TownDefBonus         int `yaml:"town_def_bonus"`
	HamletDefBonus       int `yaml:"hamlet_def_bonus"`
	CavalryForestPenalty int `yaml:"cavalry_forest_penalty"`
	BridgeAttackPenalty  int `yaml:"bridge_attack_penalty"`
	RoadMarchMoveBonus   int `yaml:"road_march_move_bonus"`
	TownPoints           int `yaml:"town_points"`
	HamletPoints         int `yaml:"hamlet_points"`
}

func (e TerrainEffects) validate() error {
	if e.ForestDefBonus < 0 || e.TownDefBonus < 0 || e.HamletDefBonus < 0 ||
		e.CavalryForestPenalty < 0 || e.BridgeAttackPenalty < 0 || e.RoadMarchMoveBonus < 0 ||
		e.TownPoints < 0 || e.HamletPoints < 0 {
		return fmt.Errorf("terrain effects must not be negative")
	}
	return nil
}

// LoadTerrainEffects reads and validates data/rules/terrain.yaml.
func LoadTerrainEffects(path string) (TerrainEffects, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TerrainEffects{}, fmt.Errorf("rules: %w", err)
	}
	var e TerrainEffects
	if err := yaml.Unmarshal(raw, &e); err != nil {
		return TerrainEffects{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	if err := e.validate(); err != nil {
		return TerrainEffects{}, fmt.Errorf("rules: %s: %w", path, err)
	}
	return e, nil
}
