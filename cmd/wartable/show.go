package main

import (
	"fmt"
	"io"
)

func runShow(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: wartable show <state.json>")
	}
	save, err := loadSave(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%s, turn %d (tie-break: %s)\n\n", save.ScenarioID, save.Turn, save.TieBreak.Holder)
	fmt.Fprint(out, renderMap(save.State.Board))

	for _, side := range []string{"north", "south"} {
		fmt.Fprintf(out, "\n%s reserves:", side)
		if len(save.State.Reserves[side]) == 0 {
			fmt.Fprint(out, " none")
		}
		for _, u := range save.State.Reserves[side] {
			fmt.Fprintf(out, " %s", u.ID)
		}
		fmt.Fprintln(out)
	}
	return nil
}
