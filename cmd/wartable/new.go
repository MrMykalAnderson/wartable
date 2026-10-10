package main

import (
	"fmt"
	"io"

	"github.com/MrMykalAnderson/wartable/internal/save"
)

func runNew(args []string, out io.Writer) error {
	if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("usage: wartable new <state.json> [scenario-id]")
	}
	outPath := args[0]
	scenarioID := save.DefaultScenarioID
	if len(args) == 2 {
		scenarioID = args[1]
	}

	units, _, scenario, err := save.LoadRulesData(scenarioID)
	if err != nil {
		return err
	}

	f := save.NewGame(units, scenario)
	if err := save.Write(outPath, f); err != nil {
		return err
	}
	fmt.Fprintf(out, "created %s: %s, turn 1\n", outPath, scenario.Name)
	return nil
}
