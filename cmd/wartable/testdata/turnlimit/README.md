# Golden game: turn limit

A full 12-order-sheet Starter Battle, played by hand through the CLI,
reaching the turn limit (docs/starter-battle.md "Winning", rule 2) with
both sides still fielding units; South wins on points (35 vs North's 30).
See `golden_test.go` (`TestGoldenTurnLimit`).

Notable moments (by turn):

- **Turns 4-5**: `North 1st Artillery` fires on both South infantry units
  from range, dropping each to half strength, to set up turn 6.
- **Turn 6**: `North 1st Cavalry` moves to E9, adjacent to both weakened
  South infantry units at once. **Ambushed by more than one enemy**: both
  attack in sequence — `South 2nd Infantry` attacks first and loses (it
  was already half strength, so it's destroyed outright), then
  `South 1st Infantry` attacks and wins, knocking the cavalry back a hex.
- **Turn 8**: `North 1st Infantry` moves into F5, is ambushed by
  `South 1st Cavalry`, loses, and is knocked back to E5. `North 2nd
  Infantry` then moves into the same now-empty F5 and is ambushed by the
  same cavalry; it loses too, but this time **the knockback destination
  (E5) is occupied** by the first infantry's retreat, so it's destroyed
  instead of knocked back.
- **Turns 9-11**: survivors retreat out of contact range on both sides.
- **Turn 12**: the turn limit is reached with three units left on each
  side; scores are each side's surviving Cost (half-strength units count
  half, rounded down): North 30 (half infantry 5 + half cavalry 10 + full
  artillery 15), South 35 (full cavalry 20 + full artillery 15). South
  wins on points.
