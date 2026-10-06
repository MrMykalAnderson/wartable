package game

import (
	"fmt"

	"github.com/MrMykalAnderson/wartable/internal/orders"
	"github.com/MrMykalAnderson/wartable/internal/rules"
)

// Initiative sums the effective Move stat of every unit given a Move order
// (docs/core-rules.md section 5.2); other orders don't count.
func Initiative(board Board, core rules.CoreRules, sideOrders []orders.Order) (int, error) {
	total := 0
	for _, o := range sideOrders {
		if !o.Type.CountsForInitiative() {
			continue
		}
		u, ok := board.Unit(o.Unit)
		if !ok {
			return 0, fmt.Errorf("game: initiative: unknown unit %q", o.Unit)
		}
		total += u.Stats(core).Move
	}
	return total, nil
}

// TieBreak tracks which side holds the initiative tie-break token
// (docs/core-rules.md section 5.2): its holder wins a tied initiative, and
// the token then passes to the other side.
type TieBreak struct {
	Holder string
}

// Resolve returns which of sideA/sideB has initiative, given both totals
// (the lower total wins), advancing the tie-break token if they're equal.
func (t *TieBreak) Resolve(sideA string, totalA int, sideB string, totalB int) string {
	switch {
	case totalA < totalB:
		return sideA
	case totalB < totalA:
		return sideB
	}
	winner := t.Holder
	if winner == sideA {
		t.Holder = sideB
	} else {
		t.Holder = sideA
	}
	return winner
}

// Step is one order in the alternating execution sequence.
type Step struct {
	Side  string
	Order orders.Order
}

// BuildExecutionOrder alternates between the two sides' orders, starting
// with firstSide's first order, then secondSide's first, and so on
// (docs/core-rules.md section 5.3). Once one side's orders run out, the
// other side's remaining orders are carried out back to back.
func BuildExecutionOrder(firstSide string, firstOrders []orders.Order, secondSide string, secondOrders []orders.Order) []Step {
	var steps []Step
	for i := 0; i < len(firstOrders) || i < len(secondOrders); i++ {
		if i < len(firstOrders) {
			steps = append(steps, Step{Side: firstSide, Order: firstOrders[i]})
		}
		if i < len(secondOrders) {
			steps = append(steps, Step{Side: secondSide, Order: secondOrders[i]})
		}
	}
	return steps
}
