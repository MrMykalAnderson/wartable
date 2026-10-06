package main

import (
	"fmt"
	"strings"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// renderMap draws a plain ASCII grid of the board (docs/dev-plan.md
// section 7, M5): column letters across the top, row numbers down the
// side, and a two-letter code per occupied hex (side initial + unit-type
// initial, lowercase for a half-strength unit). It does not attempt to
// draw the hex grid's visual stagger (docs/core-rules.md section 2.1).
func renderMap(board game.Board) string {
	cell := make(map[hex.Offset]string, len(board.Units))
	for _, u := range board.Units {
		cell[u.Pos] = unitCode(u)
	}

	var b strings.Builder
	b.WriteString("    ")
	for col := 0; col < board.Columns; col++ {
		fmt.Fprintf(&b, "%-3s", columnLabel(col))
	}
	b.WriteString("\n")
	for row := 0; row < board.Rows; row++ {
		fmt.Fprintf(&b, "%3d ", row+1)
		for col := 0; col < board.Columns; col++ {
			code, ok := cell[hex.Offset{Col: col, Row: row}]
			if !ok {
				code = "."
			}
			fmt.Fprintf(&b, "%-3s", code)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func columnLabel(col int) string {
	return strings.TrimRight(hex.Offset{Col: col, Row: 0}.String(), "0123456789")
}

func unitCode(u game.UnitInstance) string {
	code := strings.ToUpper(u.Side[:1]) + strings.ToUpper(u.Template.ID[:1])
	if u.Strength == game.Half {
		code = strings.ToLower(code)
	}
	return code
}

// formatEvent renders one turn-execution event (docs/dev-plan.md section
// 4), including the numbers behind a melee or ranged attack.
func formatEvent(e game.Event) string {
	return e.Summary()
}
