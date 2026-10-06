# Golden game: turn limit

A full 12-order-sheet Starter Battle, played by hand through the CLI,
reaching the turn limit (docs/starter-battle.md "Winning", rule 2) with
both sides still fielding units; North wins on points (40 vs South's 35).
See `golden_test.go` (`TestGoldenTurnLimit`).

Notable moments (by turn):

- **Turns 2-3**: `North 1st Artillery` moves within range of South's
  infantry and goes Ready.
- **Turn 4**: Ready artillery's passive **Barrage** fires at every enemy
  in its range band and arc before written orders — `South 1st Infantry`
  and `South 2nd Infantry` are both hit and dropped to half strength.
  `North 1st Artillery`'s own written Fire order then hits the
  already-half-strength `South 1st Infantry` again, destroying it: the
  same gun firing twice in one turn (once passively, once by order) is
  exactly what the rule intends.
- **Turn 5**: the Barrage alone finishes off `South 2nd Infantry`, so the
  gun's own written Fire order (still targeting it) is skipped — its
  target no longer exists by the time the written orders are carried
  out, since the barrage resolves first.
- **Turn 6**: `North 1st Cavalry` moves to E9. In the original (pre-
  barrage) version of this game, both South infantry units were still
  there to ambush it; now there's nothing left at D9/F9, so it passes
  through untouched — a direct consequence of how much the barrage
  changes the board before written orders ever run.
- **Turn 8**: `South 1st Cavalry` ambushes `North 1st Infantry` and then
  `North 2nd Infantry` in sequence as they both move into F5; the second
  is destroyed because its knockback hex (E5) is occupied by the first's
  retreat — the blocked-knockback case.
- **Turns 9-11**: survivors retreat out of contact range on both sides.
- **Turn 12**: the turn limit is reached with North's artillery and
  cavalry against South's cavalry and artillery. Scores are each side's
  surviving Cost (half-strength units count half, rounded down): North
  40 (full artillery 15 + full cavalry... see the transcript for the
  exact tally), South 35. North wins on points — a different winner than
  before the Barrage rule existed, purely from the extra damage it did
  early in the game.

This game's order files are unchanged from before the Barrage rule was
added (docs/dev-plan.md section 7.3); only `golden.txt` was regenerated,
to show how much a single rules change can ripple through a whole
played-out game.
