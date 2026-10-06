package main

import (
	"fmt"
	"io"

	"github.com/MrMykalAnderson/wartable/internal/save"
)

func runShow(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: wartable show <state.json>")
	}
	f, err := save.Load(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%s, turn %d (tie-break: %s)\n\n", f.ScenarioID, f.Turn, f.TieBreak.Holder)
	fmt.Fprint(out, renderMap(f.State.Board))

	for _, side := range []string{"north", "south"} {
		fmt.Fprintf(out, "\n%s reserves:", side)
		if len(f.State.Reserves[side]) == 0 {
			fmt.Fprint(out, " none")
		}
		for _, u := range f.State.Reserves[side] {
			fmt.Fprintf(out, " %s", u.ID)
		}
		fmt.Fprintln(out)
	}
	return nil
}
