package rules

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MrMykalAnderson/wartable/internal/hex"
)

// mapFile is the raw YAML shape of a data/maps/*.yaml file (e.g.
// data/maps/two-towns.yaml).
type mapFile struct {
	ID         string              `yaml:"id"`
	Name       string              `yaml:"name"`
	Columns    int                 `yaml:"columns"`
	Rows       int                 `yaml:"rows"`
	TerrainKey map[string]string   `yaml:"terrain_key"`
	Terrain    []string            `yaml:"terrain"`
	Towns      map[string][]string `yaml:"towns"`
	Hamlets    map[string][]string `yaml:"hamlets"`
	Roads      map[string][]string `yaml:"roads"`
	River      []string            `yaml:"river"`
	Bridges    []string            `yaml:"bridges"`
}

// TerrainMap is a scenario's map: terrain, roads, river and bridges,
// and objectives (docs/core-rules.md section 2.3). A nil *TerrainMap
// means a scenario with no terrain at all (e.g. the Starter Battle):
// every method on it is safe to call and reports "no terrain here".
type TerrainMap struct {
	ID, Name      string
	Columns, Rows int
	Forest        map[hex.Offset]bool
	TownAt        map[hex.Offset]string // hex -> objective name.
	HamletAt      map[hex.Offset]string
	RoadAt        map[hex.Offset]bool
	River         map[hex.Edge]bool // every river edge, including bridges.
	Bridge        map[hex.Edge]bool // the subset of River that are bridges.
	Towns         map[string][]hex.Offset
	Hamlets       map[string][]hex.Offset
	Roads         map[string][]hex.Offset // road chains, in map order, for road-march routing.
}

// IsForest reports whether h is forest (never true for a road hex: core-
// rules.md 2.3 says roads are cleared through it).
func (m *TerrainMap) IsForest(h hex.Offset) bool {
	return m != nil && m.Forest[h]
}

// IsRoad reports whether h is a road hex.
func (m *TerrainMap) IsRoad(h hex.Offset) bool {
	return m != nil && m.RoadAt[h]
}

// Objective returns the name of the town or hamlet at h, if any.
func (m *TerrainMap) Objective(h hex.Offset) (name string, ok bool) {
	if m == nil {
		return "", false
	}
	if n, ok := m.TownAt[h]; ok {
		return n, true
	}
	if n, ok := m.HamletAt[h]; ok {
		return n, true
	}
	return "", false
}

// IsTown and IsHamlet report whether h belongs to a town or hamlet
// (docs/core-rules.md 2.3: +1 Def for a defender there).
func (m *TerrainMap) IsTown(h hex.Offset) bool   { return m != nil && m.TownAt[h] != "" }
func (m *TerrainMap) IsHamlet(h hex.Offset) bool { return m != nil && m.HamletAt[h] != "" }

// RiverBlocks reports whether a and b (which must be adjacent) are
// separated by a river edge with no bridge: movement, and adjacency for
// contact/support/ambush/overrun/no-man's-land, are blocked across it.
func (m *TerrainMap) RiverBlocks(a, b hex.Offset) bool {
	if m == nil {
		return false
	}
	e := hex.NewEdge(a, b)
	return m.River[e] && !m.Bridge[e]
}

// IsBridge reports whether a and b (which must be adjacent) are joined
// by a bridge.
func (m *TerrainMap) IsBridge(a, b hex.Offset) bool {
	if m == nil {
		return false
	}
	return m.Bridge[hex.NewEdge(a, b)]
}

// AllObjectives returns every town and hamlet's name mapped to its
// hexes, for scoring and ownership tracking.
func (m *TerrainMap) AllObjectives() map[string][]hex.Offset {
	all := map[string][]hex.Offset{}
	if m == nil {
		return all
	}
	for name, hexes := range m.Towns {
		all[name] = hexes
	}
	for name, hexes := range m.Hamlets {
		all[name] = hexes
	}
	return all
}

// LoadTerrainMap reads and validates a map file such as
// data/maps/two-towns.yaml (docs/dev-plan.md section 7.8): river and
// bridge edges must join adjacent hexes, every bridge must be a river
// edge with road on both sides, and road chains must be contiguous.
func LoadTerrainMap(path string) (*TerrainMap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("rules: %w", err)
	}
	var mf mapFile
	if err := yaml.Unmarshal(raw, &mf); err != nil {
		return nil, fmt.Errorf("rules: %s: %w", path, err)
	}
	if mf.Columns <= 0 || mf.Rows <= 0 {
		return nil, fmt.Errorf("rules: %s: columns/rows must be positive", path)
	}

	m := &TerrainMap{
		ID: mf.ID, Name: mf.Name, Columns: mf.Columns, Rows: mf.Rows,
		Forest: map[hex.Offset]bool{}, TownAt: map[hex.Offset]string{}, HamletAt: map[hex.Offset]string{},
		RoadAt: map[hex.Offset]bool{}, River: map[hex.Edge]bool{}, Bridge: map[hex.Edge]bool{},
		Towns: map[string][]hex.Offset{}, Hamlets: map[string][]hex.Offset{}, Roads: map[string][]hex.Offset{},
	}
	inBounds := func(h hex.Offset) bool {
		return h.Col >= 0 && h.Col < m.Columns && h.Row >= 0 && h.Row < m.Rows
	}

	// Roads first, so forest-parsing below can exclude road hexes
	// (a road hex is never forest: core-rules.md 2.3).
	for name, names := range mf.Roads {
		hexes, err := parseHexes(names)
		if err != nil {
			return nil, fmt.Errorf("rules: %s: road %q: %w", path, name, err)
		}
		for _, h := range hexes {
			if !inBounds(h) {
				return nil, fmt.Errorf("rules: %s: road %q: %s is outside the %dx%d map", path, name, h, m.Columns, m.Rows)
			}
			m.RoadAt[h] = true
		}
		for i := 1; i < len(hexes); i++ {
			if _, ok := hex.AdjacentDirection(hexes[i-1], hexes[i]); !ok {
				return nil, fmt.Errorf("rules: %s: road %q: %s and %s are not adjacent", path, name, hexes[i-1], hexes[i])
			}
		}
		m.Roads[name] = hexes
	}

	for row, line := range mf.Terrain {
		for col, ch := range line {
			kind, ok := mf.TerrainKey[string(ch)]
			if !ok {
				return nil, fmt.Errorf("rules: %s: unknown terrain character %q", path, string(ch))
			}
			h := hex.Offset{Col: col, Row: row}
			switch kind {
			case "forest":
				if !m.RoadAt[h] {
					m.Forest[h] = true
				}
			case "open", "town", "hamlet":
				// Town/hamlet hexes are set below, from the towns/
				// hamlets lists, so the objective name is known.
			default:
				return nil, fmt.Errorf("rules: %s: unknown terrain kind %q", path, kind)
			}
		}
	}

	for name, names := range mf.Towns {
		hexes, err := parseHexes(names)
		if err != nil {
			return nil, fmt.Errorf("rules: %s: town %q: %w", path, name, err)
		}
		for _, h := range hexes {
			m.TownAt[h] = name
		}
		m.Towns[name] = hexes
	}
	for name, names := range mf.Hamlets {
		hexes, err := parseHexes(names)
		if err != nil {
			return nil, fmt.Errorf("rules: %s: hamlet %q: %w", path, name, err)
		}
		for _, h := range hexes {
			m.HamletAt[h] = name
		}
		m.Hamlets[name] = hexes
	}

	for _, s := range mf.River {
		e, err := parseEdge(s)
		if err != nil {
			return nil, fmt.Errorf("rules: %s: river edge %q: %w", path, s, err)
		}
		m.River[e] = true
	}
	for _, s := range mf.Bridges {
		e, err := parseEdge(s)
		if err != nil {
			return nil, fmt.Errorf("rules: %s: bridge %q: %w", path, s, err)
		}
		if !m.River[e] {
			return nil, fmt.Errorf("rules: %s: bridge %q is not a river edge", path, s)
		}
		if !m.RoadAt[e.A] || !m.RoadAt[e.B] {
			return nil, fmt.Errorf("rules: %s: bridge %q needs road on both sides", path, s)
		}
		m.Bridge[e] = true
	}

	return m, nil
}

func parseHexes(names []string) ([]hex.Offset, error) {
	hexes := make([]hex.Offset, len(names))
	for i, s := range names {
		h, err := hex.ParseOffset(s)
		if err != nil {
			return nil, err
		}
		hexes[i] = h
	}
	return hexes, nil
}

func parseEdge(s string) (hex.Edge, error) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return hex.Edge{}, fmt.Errorf("invalid edge %q, want HEX|HEX", s)
	}
	a, err := hex.ParseOffset(parts[0])
	if err != nil {
		return hex.Edge{}, err
	}
	b, err := hex.ParseOffset(parts[1])
	if err != nil {
		return hex.Edge{}, err
	}
	if _, ok := hex.AdjacentDirection(a, b); !ok {
		return hex.Edge{}, fmt.Errorf("%s and %s are not adjacent", a, b)
	}
	return hex.NewEdge(a, b), nil
}
