package game

import "github.com/MrMykalAnderson/wartable/internal/rules"

// InitialObjectives sets the starting ownership of scenario's
// objectives (docs/two-towns.md "Objectives"): each side owns its own
// town (named by its deployment zone's near_town); hamlets, and a
// scenario with no terrain, start with no owner.
func InitialObjectives(scenario rules.Scenario) map[string]string {
	objectives := map[string]string{}
	for side, dz := range scenario.DeploymentZones {
		if dz.NearTown != "" {
			objectives[dz.NearTown] = side
		}
	}
	return objectives
}

// UpdateObjectives updates which side owns each of scenario's
// objectives: an objective currently held by units of exactly one side
// changes (or stays) owned by that side; one held by both sides, or by
// neither, keeps its existing owner (docs/two-towns.md "Objectives":
// "the last side to have had a unit in it"). Called once per turn,
// after all orders and the barrage. A no-op for a scenario with no
// terrain.
func UpdateObjectives(state GameState, scenario rules.Scenario) GameState {
	if scenario.Terrain == nil {
		return state
	}
	objectives := make(map[string]string, len(state.Objectives))
	for k, v := range state.Objectives {
		objectives[k] = v
	}
	for name, hexes := range scenario.Terrain.AllObjectives() {
		sides := map[string]bool{}
		for _, h := range hexes {
			if u, ok := state.Board.UnitAt(h); ok {
				sides[u.Side] = true
			}
		}
		if len(sides) == 1 {
			for side := range sides {
				objectives[name] = side
			}
		}
	}
	state.Objectives = objectives
	return state
}
