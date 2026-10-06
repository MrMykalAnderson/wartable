package main

import (
	"strings"
	"testing"

	"github.com/MrMykalAnderson/wartable/internal/game"
	"github.com/MrMykalAnderson/wartable/internal/hex"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

func TestColumnLabel(t *testing.T) {
	cases := []struct {
		col  int
		want string
	}{{0, "A"}, {1, "B"}, {25, "Z"}, {26, "AA"}}
	for _, c := range cases {
		if got := columnLabel(c.col); got != c.want {
			t.Errorf("columnLabel(%d) = %q, want %q", c.col, got, c.want)
		}
	}
}

func TestUnitCode(t *testing.T) {
	u := game.UnitInstance{Side: "north", Template: rules.Unit{ID: "infantry"}, Strength: game.Full}
	if got := unitCode(u); got != "NI" {
		t.Errorf("unitCode(full) = %q, want NI", got)
	}
	u.Strength = game.Half
	if got := unitCode(u); got != "ni" {
		t.Errorf("unitCode(half) = %q, want ni", got)
	}
}

func TestRenderMapShowsUnitsAndEmptyHexes(t *testing.T) {
	board := game.Board{Columns: 3, Rows: 2, Units: []game.UnitInstance{
		{ID: "A", Side: "north", Template: rules.Unit{ID: "infantry"}, Pos: hex.Offset{Col: 0, Row: 0}, Strength: game.Full},
	}}
	out := renderMap(board)
	if !strings.Contains(out, "NI") {
		t.Errorf("renderMap output missing unit code NI:\n%s", out)
	}
	if !strings.Contains(out, ".") {
		t.Errorf("renderMap output missing empty-hex marker:\n%s", out)
	}
	if !strings.Contains(out, "A") || !strings.Contains(out, "C") {
		t.Errorf("renderMap output missing column headers:\n%s", out)
	}
}
