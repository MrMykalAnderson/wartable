package main

import (
	"fmt"
	"os"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

func runTurn(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: wartable turn <state.json> <north-orders.txt> <south-orders.txt>")
	}
	statePath, northPath, southPath := args[0], args[1], args[2]

	save, err := loadSave(statePath)
	if err != nil {
		return err
	}
	_, core, scenario, err := loadRulesData()
	if err != nil {
		return err
	}

	northOrders, err := parseOrderFile(northPath)
	if err != nil {
		return err
	}
	southOrders, err := parseOrderFile(southPath)
	if err != nil {
		return err
	}

	newState, events, err := game.ExecuteTurn(save.State, core, scenario, &save.TieBreak, northOrders, southOrders)
	if err != nil {
		return fmt.Errorf("execute turn: %w", err)
	}
	save.State = newState

	fmt.Printf("Turn %d\n\n", save.Turn)
	for _, e := range events {
		fmt.Println(formatEvent(e))
	}
	fmt.Println()
	fmt.Print(renderMap(save.State.Board))

	reportOutcome(save.State, scenario, save.Turn)

	save.Turn++
	return writeSave(statePath, save)
}

func parseOrderFile(path string) ([]orders.Order, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	ords, err := orders.Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return ords, nil
}

// reportOutcome checks the Starter Battle's win conditions
// (docs/starter-battle.md "Winning") and prints the result, if any.
func reportOutcome(state game.GameState, scenario rules.Scenario, turn int) {
	outcome := game.CheckAnnihilation(state)
	if !outcome.Over && turn >= scenario.TurnLimit {
		north, south, limitOutcome := game.ScoreAtTurnLimit(state)
		fmt.Printf("\nTurn limit reached. North %d, South %d.\n", north, south)
		outcome = limitOutcome
	}
	if !outcome.Over {
		return
	}
	if outcome.Winner != "" {
		fmt.Printf("\n%s wins: %s\n", outcome.Winner, outcome.Reason)
	} else {
		fmt.Printf("\ndraw: %s\n", outcome.Reason)
	}
}
