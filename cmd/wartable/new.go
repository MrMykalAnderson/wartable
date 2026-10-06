package main

import (
	"fmt"
	"strings"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// suggestedArmy is the Starter Battle's suggested army
// (docs/starter-battle.md "Armies"): 2 Infantry, 1 Cavalry, 1 Artillery,
// Cost 55. M5 always starts both sides with it; choosing a different army
// is a later milestone's concern.
var suggestedArmy = []string{"infantry", "infantry", "cavalry", "artillery"}

func runNew(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: wartable new <state.json>")
	}
	outPath := args[0]

	units, _, scenario, err := loadRulesData()
	if err != nil {
		return err
	}

	reserves := map[string][]game.UnitInstance{}
	for _, side := range []string{"north", "south"} {
		reserves[side] = buildArmy(units, side)
	}

	save := SaveFile{
		ScenarioID: scenario.ID,
		Turn:       1,
		TieBreak:   game.TieBreak{Holder: scenario.TieBreakHolder},
		State: game.GameState{
			Board:    game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows},
			Reserves: reserves,
		},
	}
	if err := writeSave(outPath, save); err != nil {
		return err
	}
	fmt.Printf("created %s: %s, turn 1\n", outPath, scenario.Name)
	return nil
}

func buildArmy(units map[string]rules.Unit, side string) []game.UnitInstance {
	counts := map[string]int{}
	army := make([]game.UnitInstance, 0, len(suggestedArmy))
	for _, templateID := range suggestedArmy {
		counts[templateID]++
		tmpl := units[templateID]
		name := fmt.Sprintf("%s %s %s", capitalize(side), ordinal(counts[templateID]), capitalize(templateID))
		army = append(army, game.UnitInstance{
			ID:       name,
			Side:     side,
			Template: tmpl,
			Strength: game.Full,
			State:    tmpl.DeployState,
		})
	}
	return army
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}
