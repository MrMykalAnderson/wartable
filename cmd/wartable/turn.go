package main

import (
	"fmt"
	"io"
	"os"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
	"github.com/MrMykalAnderson/wartable/internal/save"
)

func runTurn(args []string, out io.Writer) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: wartable turn <state.json> <north-orders.txt> <south-orders.txt>")
	}
	statePath, northPath, southPath := args[0], args[1], args[2]

	f, err := save.Load(statePath)
	if err != nil {
		return err
	}
	_, core, scenario, err := save.LoadRulesData(f.ScenarioID)
	if err != nil {
		return err
	}
	f.AttachTerrain(scenario)

	northRaw, northOrders, err := parseOrderFile(northPath)
	if err != nil {
		return err
	}
	southRaw, southOrders, err := parseOrderFile(southPath)
	if err != nil {
		return err
	}

	boardBefore := f.State.Board
	newState, events, err := game.ExecuteTurn(f.State, core, scenario, &f.TieBreak, northOrders, southOrders)
	if err != nil {
		return fmt.Errorf("execute turn: %w", err)
	}
	f.State = newState
	f.RecordTurn(boardBefore, northRaw, southRaw, events)

	fmt.Fprintf(out, "Turn %d\n\n", f.Turn)
	for _, e := range events {
		fmt.Fprintln(out, formatEvent(e))
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, renderMap(f.State.Board))

	reportOutcome(out, f.State, core, scenario, f.Turn)

	f.Turn++
	return save.Write(statePath, f)
}

func parseOrderFile(path string) (raw string, ords []orders.Order, err error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", path, err)
	}
	ords, err = orders.Parse(string(rawBytes))
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", path, err)
	}
	return string(rawBytes), ords, nil
}

// reportOutcome checks the scenario's win conditions (docs/starter-
// battle.md and docs/two-towns.md "Winning") and prints the result, if
// any.
func reportOutcome(out io.Writer, state game.GameState, core rules.CoreRules, scenario rules.Scenario, turn int) {
	outcome := game.CheckCapture(state, scenario)
	if !outcome.Over {
		outcome = game.CheckAnnihilation(state)
	}
	if !outcome.Over && turn >= scenario.TurnLimit {
		north, south, limitOutcome := game.ScoreAtTurnLimit(state, core, scenario)
		fmt.Fprintf(out, "\nTurn limit reached. North %d, South %d.\n", north, south)
		outcome = limitOutcome
	}
	if !outcome.Over {
		return
	}
	if outcome.Winner != "" {
		fmt.Fprintf(out, "\n%s wins: %s\n", outcome.Winner, outcome.Reason)
	} else {
		fmt.Fprintf(out, "\ndraw: %s\n", outcome.Reason)
	}
}
