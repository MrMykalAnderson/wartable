package main

import "github.com/MrMykalAnderson/wartable/internal/rules"

// Rules data paths, relative to the repository root (docs/dev-plan.md
// section 3).
const (
	unitsPath     = "data/units/standard.yaml"
	coreRulesPath = "data/rules/core.yaml"
	scenarioPath  = "data/scenarios/starter-battle.yaml"
)

// loadRulesData loads the unit templates, core rule numbers and the
// Starter Battle scenario. M5 only plays the Starter Battle; a later
// milestone could take the scenario as a flag.
func loadRulesData() (map[string]rules.Unit, rules.CoreRules, rules.Scenario, error) {
	units, err := rules.LoadUnits(unitsPath)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	core, err := rules.LoadCoreRules(coreRulesPath)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	scenario, err := rules.LoadScenario(scenarioPath, units)
	if err != nil {
		return nil, rules.CoreRules{}, rules.Scenario{}, err
	}
	return units, core, scenario, nil
}
