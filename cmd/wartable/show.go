package main

import "fmt"

func runShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: wartable show <state.json>")
	}
	save, err := loadSave(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%s, turn %d (tie-break: %s)\n\n", save.ScenarioID, save.Turn, save.TieBreak.Holder)
	fmt.Print(renderMap(save.State.Board))

	for _, side := range []string{"north", "south"} {
		fmt.Printf("\n%s reserves:", side)
		if len(save.State.Reserves[side]) == 0 {
			fmt.Print(" none")
		}
		for _, u := range save.State.Reserves[side] {
			fmt.Printf(" %s", u.ID)
		}
		fmt.Println()
	}
	return nil
}
