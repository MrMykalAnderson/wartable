# Golden game: annihilation

A full Starter Battle, played by hand through the CLI, ending in
annihilation (docs/starter-battle.md "Winning", rule 1). See
`golden_test.go` (`TestGoldenAnnihilation`).

Regenerated for docs/dev-plan.md section 7.5's combat rewrite (margin
decides the result; a margin of 3+ destroys any unit outright, with no
knockback attempted at all): North now loses its whole force by **turn
9** instead of turn 12, so `north-10.txt` through `north-12.txt` (and
their `south-*` counterparts) were deleted — nothing was left to order.
`golden_test.go` now takes a turn count per scenario rather than a
hardcoded 12.

Notable moments (by turn):

- **Turn 2**: `North 2nd Infantry | Deploy | D2` fails — D2 is already
  occupied by `North 1st Infantry` (docs/core-rules.md section 12,
  provisional rule 4).
- **Turn 4**: a half-strength `North 1st Infantry` (Attack reduced to 1)
  closes on `South 1st Infantry` and attacks its **rear**; margin 2 is
  still only a hit at that reduced Attack, so South is knocked back
  rather than destroyed.
- **Turn 5**: `North 1st Cavalry` is ambushed by `South 1st Artillery`
  at margin 3 and destroyed outright, at full strength, with no
  knockback attempted. `North 2nd Infantry` separately closes on
  `South 2nd Infantry` and hits its flank for a knockback.
- **Turns 7-9**: `South 1st Cavalry`, built up to +2 support, destroys
  `North 2nd Infantry` and then `North 1st Infantry` in separate
  ambushes (margins 4 and 6); `South 1st Artillery` finishes the game by
  destroying `North 1st Artillery` in an ambush at margin 4. South wins:
  north annihilated.

Margin 3+ bypasses knockback entirely now, so neither golden game
produces a blocked-knockback destruction any more (both did, before
this rewrite). That case is still covered directly by
`TestMeleeEX4DestroyedIfRetreatBlocked` and
`TestAmbushEndsWhenKnockbackWouldBeAdjacentToAnotherEnemy` in
`internal/game`.
