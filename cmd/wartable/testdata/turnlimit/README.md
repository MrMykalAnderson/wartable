# Golden game: turn limit

A full 12-order-sheet Starter Battle, played by hand through the CLI,
reaching the turn limit (docs/starter-battle.md "Winning", rule 2) with
both sides still fielding units. See `golden_test.go`
(`TestGoldenTurnLimit`).

Regenerated for docs/dev-plan.md section 7.5's combat rewrite (margin
decides the result directly; no more separate AttDmg/RngDmg damage
check): the result flipped from North winning 40-35 to a **35-35
draw**, and turn 8's double ambush now destroys both North infantry
units outright (margin 3) rather than knocking the first back and
blocking the second's retreat — this game no longer demonstrates the
blocked-knockback case (see `TestMeleeEX4DestroyedIfRetreatBlocked` and
`TestAmbushEndsWhenKnockbackWouldBeAdjacentToAnotherEnemy` in
`internal/game` instead).

Notable moments (by turn):

- **Turns 2-3**: `North 1st Artillery` moves within range of South's
  infantry and goes Ready.
- **Turn 4**: Ready artillery's passive **Barrage** fires at every enemy
  in its range band and arc before written orders — `South 1st Infantry`
  and `South 2nd Infantry` are both hit. `North 1st Artillery`'s own
  written Fire order then hits the already-half-strength
  `South 1st Infantry` again (margin 2, but any hit on a half-strength
  unit destroys it): the same gun firing twice in one turn (once
  passively, once by order) is exactly what the rule intends.
- **Turn 5**: the Barrage alone finishes off `South 2nd Infantry` (any
  hit destroys an already-half unit), so the gun's own written Fire
  order (still targeting it) is skipped — its target no longer exists by
  the time the written orders are carried out, since the barrage
  resolves first.
- **Turn 6**: `North 1st Cavalry` moves to E9 and passes through
  untouched — both South infantry units are already gone by this point.
- **Turn 8**: `South 1st Cavalry`, attacking across the flank, destroys
  `North 1st Infantry` and then `North 2nd Infantry` outright (margin 3
  each time: no knockback is attempted at that margin). The second
  infantry's safe route (docs/dev-plan.md section 7.7) now stops it one
  hex short, at F6 rather than F5 — the first infantry's destruction
  left F5 adjacent to the cavalry — but it's ambushed there just the
  same, with an identical outcome.
- **Turns 9-11**: survivors retreat out of contact range on both sides.
- **Turn 12**: the turn limit is reached with North's artillery and
  cavalry against South's cavalry and artillery. Both sides field one
  full-strength artillery (15) and one full-strength cavalry (20): 35
  each. Draw.

This game's order files are unchanged from before this combat rewrite;
only `golden.txt` was regenerated, to show how much a single rules
change can ripple through a whole played-out game.

Regenerated again for docs/dev-plan.md section 7.7 (safe-route
pathfinding, artillery overrun instead of ambushing, dug-in Ready
artillery): only the one safe-route change above, since this game
never involves artillery in melee. Order files unchanged.
