// Package save is the on-disk JSON format shared by cmd/wartable and
// cmd/wartable-web, plus loading the rules data both use.
package save

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// File is a saved game's on-disk JSON representation.
type File struct {
	ScenarioID string
	Turn       int
	TieBreak   game.TieBreak
	State      game.GameState
	// History is every turn played so far, oldest first (docs/dev-
	// plan.md section 7.7): kept so the web viewer can step through any
	// past turn, not just the final position (and so a later online-
	// play milestone has a full record to work from).
	History []TurnRecord
}

// TurnRecord is one played turn's full record: the order sheets
// submitted (as written, not just parsed) and the resulting event log,
// plus the board exactly as it stood before the turn ran.
type TurnRecord struct {
	Turn        int
	NorthOrders string
	SouthOrders string
	BoardBefore game.Board
	Events      []game.Event
}

// RecordTurn appends a TurnRecord for the turn about to be completed
// (f.Turn, before it's incremented), for both cmd/wartable's runTurn
// and cmd/wartable-web's handleTurn to call identically.
func (f *File) RecordTurn(boardBefore game.Board, northOrders, southOrders string, events []game.Event) {
	f.History = append(f.History, TurnRecord{
		Turn:        f.Turn,
		NorthOrders: northOrders,
		SouthOrders: southOrders,
		BoardBefore: boardBefore,
		Events:      events,
	})
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

// suggestedArmy is the Starter Battle's suggested army (docs/starter-
// battle.md "Armies"): 2 Infantry, 1 Cavalry, 1 Artillery, Cost 55. Both
// cmd/wartable and cmd/wartable-web always start both sides with it;
// choosing a different army is a later milestone's concern.
var suggestedArmy = []string{"infantry", "infantry", "cavalry", "artillery"}

// NewGame builds a fresh game: both sides' reserves seeded with the
// suggested army, turn 1, tie-break holder per the scenario. It doesn't
// write anything to disk; call Write with the result to save it.
func NewGame(units map[string]rules.Unit, scenario rules.Scenario) File {
	reserves := map[string][]game.UnitInstance{}
	for _, side := range []string{"north", "south"} {
		reserves[side] = buildArmy(units, side)
	}
	return File{
		ScenarioID: scenario.ID,
		Turn:       1,
		TieBreak:   game.TieBreak{Holder: scenario.TieBreakHolder},
		State: game.GameState{
			Board:    game.Board{Columns: scenario.Map.Columns, Rows: scenario.Map.Rows},
			Reserves: reserves,
		},
	}
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
