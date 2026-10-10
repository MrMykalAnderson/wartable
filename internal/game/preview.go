package game

import (
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// PreviewMovePath returns the route a Move order from mover's current
// position to dest would take on a board with no units on it at all,
// friendly or enemy (docs/dev-plan.md section 7.9): a planning estimate,
// not a prediction of what will actually happen, since every other unit
// may itself move before this order runs. It still obeys every terrain
// rule — fording, a road march's own route and +2 Move, forest's halved
// Move — exactly as moveTowards would, just never stopped or rerouted
// by a unit in the way, the same as the reach guide (OptionsView's
// MoveHexes et al. in cmd/wartable-web/options.go).
func PreviewMovePath(core rules.CoreRules, board Board, mover UnitInstance, dest hex.Offset) []hex.Offset {
	empty := Board{Columns: board.Columns, Rows: board.Rows, Terrain: board.Terrain, Units: []UnitInstance{mover}}
	return moveTowards(core, empty, mover, dest, mover.Stats(core).Move, "").Path
}
