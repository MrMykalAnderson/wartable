// Package save is the on-disk JSON format shared by cmd/wartable and
// cmd/wartable-web, plus loading the rules data both use.
package save

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// File is a saved game's on-disk JSON representation.
type File struct {
	ScenarioID string
	Turn       int
	TieBreak   game.TieBreak
	State      game.GameState
}

// Load reads a save file.
func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return f, nil
}

// Write saves a game to path.
func Write(path string, f File) error {
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Rules data paths, relative to the repository root (docs/dev-plan.md
// section 3). Both cmd/wartable and cmd/wartable-web must be run from
// there.
const (
	UnitsPath     = "data/units/standard.yaml"
	CoreRulesPath = "data/rules/core.yaml"
	ScenarioPath  = "data/scenarios/starter-battle.yaml"
)

// LoadRulesData loads the unit templates, core rule numbers and the
// Starter Battle scenario. Only the Starter Battle is supported so far;
// a later milestone could take the scenario as a parameter.
func LoadRulesData() (map[string]rules.Unit, rules.CoreRules, rules.Scenario, error) {
	units, err := rules.LoadUnits(UnitsPath)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	core, err := rules.LoadCoreRules(CoreRulesPath)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	scenario, err := rules.LoadScenario(ScenarioPath, units)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	return units, core, scenario, nil
}
