// Package orders parses order sheets (docs/dev-plan.md section 6): plain
// text, one order per line, fields separated by "|". It checks sheet
// structure (recognized order types, required fields, valid facings, at
// most one order per unit) but not game semantics such as whether a named
// unit actually exists — that's checked at execution time by
// internal/game, which has the roster.
package orders

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// Type is an order type (docs/core-rules.md section 6).
type Type string

const (
	Deploy         Type = "Deploy"
	Move           Type = "Move"
	CloseAndAttack Type = "Close and Attack"
	Fire           Type = "Fire"
	CloseAndFire   Type = "Close and Fire"
	Ready          Type = "Ready"
	Mobilise       Type = "Mobilise"
)

var typesByName = map[string]Type{
	"deploy":           Deploy,
	"move":             Move,
	"close and attack": CloseAndAttack,
	"fire":             Fire,
	"close and fire":   CloseAndFire,
	"ready":            Ready,
	"mobilise":         Mobilise,
	"mobilize":         Mobilise,
}

// CountsForInitiative reports whether an order of this type counts toward
// its side's initiative total (docs/core-rules.md section 5.2): only Move.
func (t Type) CountsForInitiative() bool {
	return t == Move
}

// Order is one line of an order sheet.
type Order struct {
	Line int
	Unit string
	Type Type

	// HasTargetHex/TargetHex is set when the detail names a hex (Deploy;
	// Move to a hex). TargetUnit is set when it names a unit (Move to a
	// unit; Close and Attack; Fire; Close and Fire).
	HasTargetHex bool
	TargetHex    hex.Offset
	TargetUnit   string

	HasFacing bool
	Facing    hex.Direction
}

// Parse parses a whole order sheet. "#" starts a comment; blank lines are
// ignored. It returns every error found (via errors.Join), not just the
// first, so "parse errors ... are reported before execution" (section 6)
// can show the player everything wrong with the sheet at once.
func Parse(text string) ([]Order, error) {
	var ords []Order
	var errs []error
	firstLineFor := map[string]int{}

	for i, rawLine := range strings.Split(text, "\n") {
		lineNum := i + 1
		line := rawLine
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		o, err := parseLine(lineNum, line)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if first, dup := firstLineFor[o.Unit]; dup {
			errs = append(errs, fmt.Errorf("line %d: %s already has an order (line %d)", lineNum, o.Unit, first))
			continue
		}
		firstLineFor[o.Unit] = lineNum
		ords = append(ords, o)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return ords, nil
}

func parseLine(lineNum int, line string) (Order, error) {
	fields := strings.Split(line, "|")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) < 2 {
		return Order{}, fmt.Errorf("line %d: expected at least \"unit | order\", got %q", lineNum, line)
	}

	unit, typeStr := fields[0], fields[1]
	if unit == "" {
		return Order{}, fmt.Errorf("line %d: missing unit name", lineNum)
	}
	t, ok := typesByName[strings.ToLower(typeStr)]
	if !ok {
		return Order{}, fmt.Errorf("line %d: unknown order %q", lineNum, typeStr)
	}

	o := Order{Line: lineNum, Unit: unit, Type: t}
	rest := fields[2:]

	if n := len(rest); n > 0 && strings.HasPrefix(strings.ToLower(rest[n-1]), "facing ") {
		facingStr := strings.TrimSpace(rest[n-1][len("facing "):])
		d, ok := ParseDirection(facingStr)
		if !ok {
			return Order{}, fmt.Errorf("line %d: unknown facing %q", lineNum, facingStr)
		}
		o.HasFacing, o.Facing = true, d
		rest = rest[:n-1]
	}
	if len(rest) > 1 {
		return Order{}, fmt.Errorf("line %d: too many fields", lineNum)
	}
	var target string
	if len(rest) == 1 {
		target = rest[0]
	}

	switch t {
	case Deploy:
		if target == "" {
			return Order{}, fmt.Errorf("line %d: Deploy needs a hex", lineNum)
		}
		off, err := hex.ParseOffset(target)
		if err != nil {
			return Order{}, fmt.Errorf("line %d: Deploy target %q is not a hex", lineNum, target)
		}
		o.HasTargetHex, o.TargetHex = true, off
	case Move:
		if target == "" {
			return Order{}, fmt.Errorf("line %d: Move needs a hex or a unit", lineNum)
		}
		if off, err := hex.ParseOffset(target); err == nil {
			o.HasTargetHex, o.TargetHex = true, off
		} else {
			o.TargetUnit = target
		}
	case CloseAndAttack, Fire, CloseAndFire:
		if target == "" {
			return Order{}, fmt.Errorf("line %d: %s needs a target unit", lineNum, t)
		}
		o.TargetUnit = target
	case Ready, Mobilise:
		if target != "" {
			return Order{}, fmt.Errorf("line %d: %s takes no target", lineNum, t)
		}
	}
	return o, nil
}

// ParseDirection parses one of the six direction names (case-insensitive).
func ParseDirection(s string) (hex.Direction, bool) {
	switch strings.ToUpper(s) {
	case "N":
		return hex.N, true
	case "NE":
		return hex.NE, true
	case "SE":
		return hex.SE, true
	case "S":
		return hex.S, true
	case "SW":
		return hex.SW, true
	case "NW":
		return hex.NW, true
	default:
		return 0, false
	}
}
