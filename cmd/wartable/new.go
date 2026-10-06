package main

import (
	"fmt"
	"io"

	"github.com/MrMykalAnderson/wartable/internal/save"
)

func runNew(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: wartable new <state.json>")
	}
	outPath := args[0]

	units, _, scenario, err := save.LoadRulesData()
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
