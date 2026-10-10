# Golden game: annihilation

A full Starter Battle, played by hand through the CLI, ending in
annihilation (docs/starter-battle.md "Winning", rule 1). See
`golden_test.go` (`TestGoldenAnnihilation`).

Regenerated for docs/dev-plan.md section 7.7's playtest round 4 changes
(artillery never ambushes and is overrun instead; Ready artillery is
dug in — destroyed by any margin of loss, never knocked back; safe-
route pathfinding). Turns 1-9 are unchanged from before; with the
ambush bug fixed (artillery could previously attack in an ambush it
should never have taken part in), North's cavalry survives longer and
the game no longer ends by turn 9, so `north-10.txt`/`south-10.txt` and
`north-11.txt`/`south-11.txt` were added to finish it off.
`golden_test.go`'s turn count for this scenario moved from 9 to 11.

Notable moments (by turn):

- **Turn 2**: `North 2nd Infantry | Deploy | D2` fails — D2 is already
  occupied by `North 1st Infantry` (docs/core-rules.md section 12,
  provisional rule 4).
- **Turns 5, 7, 8**: `South 1st Cavalry`, picking up support as the
  game goes on, destroys `North 1st Cavalry`, `North 2nd Infantry` and
  `North 1st Infantry` in separate ambushes.
- **Turn 9**: `North 1st Artillery` (Mobilised, so able to move) moves
  into contact with `South 1st Artillery`. Neither can make melee
  attacks, so there's no ambush — instead North's gun **overruns**
  South's, destroying it at margin 3. Overrun isn't limited to
  melee-capable movers: the rule text only excludes artillery from
  Close and Attack orders and from attacking *in* an ambush.
- **Turns 10-11**: `South 1st Cavalry` hunts down North's last unit
  (its own artillery) and destroys it. South wins: north annihilated.

This game doesn't happen to exercise Ready (dug-in) artillery losing a
melee, or a unit taking the safe route around an enemy — both are
covered directly by `TestOverrunEX9Destroyed`/`TestOverrunEX9Repelled`
and `TestSafeRouteEX8` in `internal/game`.
