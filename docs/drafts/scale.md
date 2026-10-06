# Wartable Scale (Draft)

> **WORK IN PROGRESS. NOT FOR IMPLEMENTATION.**
> This chapter is a design draft under discussion. It is not part of the rules yet. Do not build, test or change data from it until it moves out of `docs/drafts/` into `docs/`.

Version 0.0 (draft). Authors: Mykal Anderson, Thomas Robertson.

Wartable can be played at any size of battle, from single soldiers to army groups. This chapter defines **scale**: what a unit, a hex and a turn represent, and how a game states its scale. It also defines **formations**, which let a player command large groups on a big map and break them down when needed.

Rules marked **(Provisional)** are first guesses for discussion. Supply, terrain and other mechanics that vary with scale will be their own chapters.

---

## Contents

1. [Principles](#1-principles)
2. [The scale ladder](#2-the-scale-ladder)
3. [Eras](#3-eras)
4. [The scale indicator](#4-the-scale-indicator)
5. [The scale card](#5-the-scale-card)
6. [Converting ranges](#6-converting-ranges)
7. [Formations](#7-formations)
8. [Order delay](#8-order-delay)
9. [Size](#9-size)
10. [The core rules as a scale](#10-the-core-rules-as-a-scale)
11. [Open questions](#11-open-questions)

---

## 1. Principles

- **Scale is the echelon.** A unit's scale is its place in the chain of command (squad, company, battalion …), not an exact headcount.
- **Stats don't change with scale.** Move, Range and every other stat are counted in hexes and turns. What a hex and a turn mean is set by the scale card, so Infantry Move 4 plays the same at every scale.
- **Only weapon reach converts.** Real weapons reach a fixed distance, so a unit's Range in hexes depends on the hex size (see [Section 6](#6-converting-ranges)).
- **Bigger scales should mean different decisions,** not just bigger counters. Higher scales gain mechanics such as supply in their own chapters.

## 2. The scale ladder

Scales are written **n0, n1, n2 …**. Each step up is roughly four of the step below. **4^n** is the nominal headcount, used as a sanity check and for points, never as a rule.

| Scale | Nominal men (4^n) | Echelon | Typical real strength |
| --- | --- | --- | --- |
| n0 | 1 | Soldier, vehicle or gun | 1 |
| n1 | 4 | Fireteam | 2–4 |
| n2 | 16 | Squad / section | 6–24 |
| n3 | 64 | Platoon | 20–50 |
| n4 | 256 | Company / battery / squadron | 100–250 |
| n5 | 1,024 | Battalion | 300–1,000 |
| n6 | 4,096 | Brigade / regiment | 3,000–5,000 |
| n7 | 16,384 | Division | 6,000–25,000 |
| n8 | 65,536 | Corps | 20,000–60,000 |
| n9 | 262,144 | Field army | 100,000–200,000 |
| n10 | 1,048,576 | Army group | 400,000–1,000,000 |

Historical units take the nearest rung. Examples: a Roman century (80) is n3, a cohort (480) is n4–n5, a legion (about 5,000) is n6; a Mongol mingghan (1,000) is n5; a Napoleonic battalion (600–800) is n5.

## 3. Eras

The era says how densely troops fight, which sets the size of a hex. Armies have spread out as weapons became deadlier, so the same echelon covers far more ground in later eras.

| Era | Covers (roughly) | n0 hex |
| --- | --- | --- |
| **Ancient** | Antiquity and the medieval period | 1.5 m |
| **Musket** | Pike-and-shot to the 1860s | 2.5 m |
| **WWII** | The world wars | 7 m |
| **Modern** | Late 20th century to today | 15 m |

**(Provisional)** These four eras and their n0 hex sizes are calibrations, not sourced figures. More eras can be added.

## 4. The scale indicator

Every scenario states its scale as **era + scale**, so players can see at a glance that two games of the "same" scale are different. For example:

- **Musket n5**: horse-and-musket battalions.
- **Ancient n4**: units of about 256, such as Macedonian syntagmata or Roman cohorts.
- **WWII n4**: rifle and tank companies.

A game with formations ([Section 7](#7-formations)) states both scales, formation first:

- **Musket n5/4**: units are companies (n4), grouped into battalion formations (n5).

Two games with the same scale number but different eras use the same unit counts but very different maps: Musket n5 hexes are about 325 m across, Modern n5 hexes about 1.9 km.

## 5. The scale card

Each scenario includes a scale card:

| Field | Meaning |
| --- | --- |
| Scale | The scale indicator, e.g. Musket n5/4. |
| Hex | The width of one hex. |
| Turn | The time one turn represents. |
| Order delay | 0 or more turns (see [Section 8](#8-order-delay)). |

**Hex size** at scale n is the era's n0 hex × 2.65^n (2.65 is √7, the ratio between nested 7-hex groups; it also matches how real frontages widen per echelon). Rounded:

| Scale | Ancient | Musket | WWII | Modern |
| --- | --- | --- | --- | --- |
| n0 | 2 m | 2 m | 7 m | 15 m |
| n1 | 4 m | 7 m | 19 m | 40 m |
| n2 | 11 m | 18 m | 49 m | 105 m |
| n3 | 28 m | 46 m | 130 m | 280 m |
| n4 | 74 m | 120 m | 340 m | 735 m |
| n5 | 195 m | 325 m | 900 m | 1.9 km |
| n6 | 515 m | 860 m | 2.4 km | 5.1 km |
| n7 | 1.4 km | 2.3 km | 6.4 km | 13.6 km |
| n8 | 3.6 km | 6 km | 17 km | 36 km |
| n9 | 9.5 km | 16 km | 45 km | 95 km |
| n10 | 25 km | 42 km | 120 km | 250 km |

**Turn length (Provisional):** the time infantry needs to cover 4 hexes: at about 50 m a minute up to n6, and about 25 km a day from n7. At WWII scales that gives about 25 minutes at n4, 3 hours at n6, 1 day at n7 and 1 week at n9.

The scale card is descriptive. It changes no rule except Range conversion and order delay.

## 6. Converting ranges

A unit template for a given era and scale sets its Range as:

**Range in hexes = the weapon's effective range ÷ hex size, rounded down.**

- If the maximum comes out as **0**, the unit has no ranged attack at that scale. Its firepower is part of the close fight and belongs in Attack.
- A minimum range that comes out as 0 or 1 is dropped (written as a single number).

For example, a Napoleonic field gun (about 1,000 m) has Range 3 at Musket n5 and Range 1 at Musket n6. A musket (about 100 m) has no Range at Musket n4 or above.

**(Provisional)** Weapon ranges come from the unit template's description, not from a central table.

## 7. Formations

A **formation** is a named group of units of one scale, moved and ordered as one. It lets a player command at the scale above their units on a single large map, and break a formation down when a unit needs to act alone.

### 7.1 What a formation is

- A formation has a name (e.g. *1st Battalion*) and **2 to 4 units** of the same scale. **(Provisional)** The usual pattern is 3 fighting units and 1 support unit.
- Its scale is one above its units: four n4 companies make an n5 formation.
- Its units must each be adjacent to at least one other unit of the formation. **(Provisional)**
- A unit belongs to at most one formation.
- The scenario's army list sets the starting formations.

### 7.2 Formation orders

A **formation order** is one line on the orders sheet, naming the formation. It counts as the order for every unit in the formation.

**(Provisional)** Formation orders:

| Order | Detail | Effect |
| --- | --- | --- |
| **Move** | A hex, a drill, optional facing | The formation moves to the hex and takes up the drill. |
| **Close and Attack** | A named enemy unit | Each unit closes on the target and attacks if it can. |
| **Hold** | Optional facing | Units stay put and may turn. |

**Drills** set how units stand around the formation's lead unit (its first listed unit): **Line** (side by side across the facing) and **Column** (one behind another). Exact drill placement and blocking are still to be written.

- A formation moves at the **Move of its slowest unit**.
- **(Provisional)** For initiative, a formation Move order counts that Move **once**, not once per unit. Moving in formation is cheaper on initiative than moving every unit separately.

### 7.3 Breaking down and reforming

- **Detach:** a unit given its own order (any core order) leaves its formation and acts alone. **(Provisional)** It stays detached until it reforms.
- **Form:** a Form order naming a formation and units that are adjacent to it (or to each other) joins them into that formation. This uses each unit's order for the turn. **(Provisional)**
- A formation reduced to one unit stops being a formation.

### 7.4 Combat

Combat is unchanged: every fight is between units, using the core rules. Formations matter for orders and movement only. Supporting neighbours in a tight drill get the core rules' support bonus naturally.

## 8. Order delay

Order delay is a dice-free way to model slow communications. It is set on the scale card by era and scale.

- **Delay 0:** orders are carried out on the turn they are written (the core rules).
- **Delay 1:** orders written this turn are carried out next turn. Each Orders Phase, players write orders for the following turn, and the orders written last turn are revealed and executed.

**(Provisional)** Suggested use: delay 1 for Ancient and Musket games at n6 and above, where orders travelled by messenger; delay 0 otherwise.

## 9. Size

**Open.** The stat block no longer uses Size, and every unit occupies one hex at every scale. Candidates under discussion:

- **Size = scale:** a unit's rung (n4, n5 …), only needed if units of different scales share a map.
- **Size = strength steps:** the number of hits a unit can take, generalising full/half strength (e.g. 4 steps for a unit made of four smaller ones).
- Drop it.

## 10. The core rules as a scale

The core rules and the Starter Battle read naturally as **Musket n5**: infantry has no ranged attack (musket fire is part of the close fight at 325 m hexes), artillery reaches 3–5 hexes (about 1–1.6 km), and the 12 × 10 map is about 4 × 3 km.

One mismatch to settle: **Cavalry Range 2** is about 650 m at Musket n5, well beyond carbines. It could drop to no Range for this era, or the unit could be read as horse artillery or (at WWII scales) mechanised troops.

## 11. Open questions

1. Era list and n0 hex sizes: are four eras enough?
2. Drill placement: exactly where each unit goes in Line and Column, and what happens when a slot is blocked.
3. Formation Close and Attack: one target for all units, or each unit picks the nearest enemy?
4. Initiative: should a formation Move count once, or once per unit?
5. Detach and Form: should breaking down or reforming cost anything beyond the unit's order?
6. Order delay: apply to all orders, or only to orders below the player's own rung?
7. Size (see [Section 9](#9-size)).
8. Points: should Cost scale ×4 per rung so armies of different scales can be compared?
9. What a hit means at large scales ("combat ineffective" rather than casualties).
