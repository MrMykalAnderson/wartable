package rules

import "testing"

const unitsPath = "../../data/units/standard.yaml"

// TestLoadUnitsMatchesDocs checks the loaded stats against the tables in
// docs/core-rules.md section 3.3/3.4.
func TestLoadUnitsMatchesDocs(t *testing.T) {
	units, err := LoadUnits(unitsPath)
	if err != nil {
		t.Fatalf("LoadUnits(%q): %v", unitsPath, err)
	}

	cases := []struct {
		id                                                   string
		cost, move, minRange, range_, rngDmg, attack, attDmg int
		melee                                                bool
	}{
		{"infantry", 10, 4, 0, 0, 0, 2, 2, true},
		{"cavalry", 20, 7, 1, 2, 2, 3, 3, true},
		{"artillery", 15, 4, 3, 5, 3, 3, 0, false},
	}
	for _, c := range cases {
		u, ok := units[c.id]
		if !ok {
			t.Errorf("unit %q not loaded", c.id)
			continue
		}
		if u.Cost != c.cost {
			t.Errorf("%s: Cost = %d, want %d", c.id, u.Cost, c.cost)
		}
		if u.Move != c.move {
			t.Errorf("%s: Move = %d, want %d", c.id, u.Move, c.move)
		}
		if u.MinRange != c.minRange {
			t.Errorf("%s: MinRange = %d, want %d", c.id, u.MinRange, c.minRange)
		}
		if u.Range != c.range_ {
			t.Errorf("%s: Range = %d, want %d", c.id, u.Range, c.range_)
		}
		if u.RngDmg != c.rngDmg {
			t.Errorf("%s: RngDmg = %d, want %d", c.id, u.RngDmg, c.rngDmg)
		}
		if u.Attack != c.attack {
			t.Errorf("%s: Attack = %d, want %d", c.id, u.Attack, c.attack)
		}
		if u.AttDmg != c.attDmg {
			t.Errorf("%s: AttDmg = %d, want %d", c.id, u.AttDmg, c.attDmg)
		}
		if u.Melee != c.melee {
			t.Errorf("%s: Melee = %v, want %v", c.id, u.Melee, c.melee)
		}
	}

	infantry := units["infantry"]
	if got := infantry.Def.Value(""); got != 2 {
		t.Errorf("infantry: Def = %d, want 2", got)
	}
	cavalry := units["cavalry"]
	if got := cavalry.Def.Value(""); got != 3 {
		t.Errorf("cavalry: Def = %d, want 3", got)
	}

	artillery := units["artillery"]
	if got := artillery.Def.Value("ready"); got != 4 {
		t.Errorf("artillery: Def(ready) = %d, want 4", got)
	}
	if got := artillery.Def.Value("mobilised"); got != 2 {
		t.Errorf("artillery: Def(mobilised) = %d, want 2", got)
	}
	if artillery.DeployState != "mobilised" {
		t.Errorf("artillery: deploy_state = %q, want mobilised", artillery.DeployState)
	}
	mobilised, ok := artillery.States["mobilised"]
	if !ok {
		t.Fatalf("artillery: no mobilised state")
	}
	if !mobilised.CanMove || mobilised.CanFire || !mobilised.CanTurn {
		t.Errorf("artillery mobilised state = %+v, want {CanMove:true CanFire:false CanTurn:true}", mobilised)
	}
	ready, ok := artillery.States["ready"]
	if !ok {
		t.Fatalf("artillery: no ready state")
	}
	if ready.CanMove || !ready.CanFire || ready.CanTurn {
		t.Errorf("artillery ready state = %+v, want {CanMove:false CanFire:true CanTurn:false} (section 3.3: Ready cannot turn either)", ready)
	}
}

func TestLoadUnitsRejectsDuplicateID(t *testing.T) {
	_, err := loadUnitsFromYAML(t, `
- id: infantry
  name: Infantry
  cost: 10
  move: 4
  def: 2
  range: 0
  rng_dmg: 0
  attack: 2
  att_dmg: 2
  melee: true
- id: infantry
  name: Infantry Again
  cost: 10
  move: 4
  def: 2
  range: 0
  rng_dmg: 0
  attack: 2
  att_dmg: 2
  melee: true
`)
	if err == nil {
		t.Fatalf("LoadUnits: want error for duplicate id, got none")
	}
}

func TestLoadUnitsRejectsStateWithoutDef(t *testing.T) {
	_, err := loadUnitsFromYAML(t, `
- id: artillery
  name: Artillery
  cost: 15
  move: 4
  def: 2
  range: 5
  rng_dmg: 3
  attack: 3
  att_dmg: 0
  melee: false
  states:
    mobilised: { can_move: true, can_fire: false, can_turn: true }
    ready: { can_move: false, can_fire: true, can_turn: false }
  deploy_state: mobilised
`)
	if err == nil {
		t.Fatalf("LoadUnits: want error for states without a per-state def, got none")
	}
}

func TestLoadUnitsRejectsMinRangeAboveRange(t *testing.T) {
	_, err := loadUnitsFromYAML(t, `
- id: cavalry
  name: Cavalry
  cost: 20
  move: 7
  def: 3
  min_range: 3
  range: 2
  rng_dmg: 2
  attack: 3
  att_dmg: 3
  melee: true
`)
	if err == nil {
		t.Fatalf("LoadUnits: want error for min_range above range, got none")
	}
}

// TestLoadUnitsDefaultsMinRange checks docs/dev-plan.md section 7.3: a
// unit with a ranged attack and no min_range given defaults to 1.
func TestLoadUnitsDefaultsMinRange(t *testing.T) {
	units, err := loadUnitsFromYAML(t, `
- id: cavalry
  name: Cavalry
  cost: 20
  move: 7
  def: 3
  range: 2
  rng_dmg: 2
  attack: 3
  att_dmg: 3
  melee: true
`)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	if got := units["cavalry"].MinRange; got != 1 {
		t.Errorf("MinRange = %d, want 1 (default)", got)
	}
}

func loadUnitsFromYAML(t *testing.T, content string) (map[string]Unit, error) {
	t.Helper()
	path := writeTempFile(t, "units-*.yaml", content)
	return LoadUnits(path)
}
