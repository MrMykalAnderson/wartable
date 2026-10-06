package hex

// Knockback returns the hex the loser of a melee is pushed to: one hex
// directly away from the winner, continuing the line from winner through
// loser (docs/core-rules.md section 8.3).
func Knockback(winner, loser Offset) Offset {
	w := winner.ToCube()
	l := loser.ToCube()
	delta := l.sub(w)
	return l.add(delta).ToOffset()
}
