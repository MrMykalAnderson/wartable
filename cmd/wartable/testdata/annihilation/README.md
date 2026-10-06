# Golden game: annihilation

A full 12-order-sheet Starter Battle, played by hand through the CLI,
ending in annihilation (docs/starter-battle.md "Winning", rule 1) at
turn 12. See `golden_test.go` (`TestGoldenAnnihilation`).

Notable moments (by turn):

- **Turn 2**: `North 2nd Infantry | Deploy | D2` fails — D2 is already
  occupied by `North 1st Infantry` (docs/core-rules.md section 12,
  provisional rule 4).
- **Turn 4**: `North 1st Infantry` closes on `South 1st Infantry` and
  attacks its **rear** (approaching from the hex directly behind its
  facing).
- **Turn 5**: `North 2nd Infantry` closes on `South 2nd Infantry`
  (pre-turned to face NW on turn 4 via `Move` to its own hex) and attacks
  its **flank**.
- **Turn 12**: South's cavalry destroys the last North unit, ending the
  game in annihilation.

This game did not end up producing a clean multi-enemy ambush or a
blocked-knockback destruction (both attempts in play either destroyed the
ambushed unit on the first blow, via the "already half strength" path
rather than a blocked retreat, or broke contact before a second attacker
could act) — both are instead demonstrated cleanly in the `turnlimit`
game.
