"""Reference implementation of Wartable movement (core-rules.md section 7).

Not part of the engine: a small, readable model of the rules, used to draw the
movement diagrams in docs/images/ and to cross-check the Go engine. If this file
and core-rules.md disagree, core-rules.md is right.

Usage:
    python3 docs/tools/movement_reference.py data/maps/two-towns.yaml            # print every example, step by step
    python3 docs/tools/movement_reference.py data/maps/two-towns.yaml out.png    # also draw docs/images/movement-examples.png

The step rule (7.2), for one order:
  * Options are open neighbouring hexes CLOSER to the destination (counting hexes).
  * If none is open, options are hexes at the SAME distance that are nearer the
    destination as the crow flies (a slide). A unit never moves away.
  * No options: stop. One option: take it. Two: not next to an enemy, then not
    forest, then keep left (the one on the left as the unit faces the destination).
  * Closed hexes: off the map, holding any unit, across a river edge (not a bridge),
    or already entered this order.
  * Forest costs 2 Move to enter, other hexes 1; the first step is always allowed.
  * Stop at the destination, on contact, or when the next step can't be paid.
Road march (2.3): start and destination both road hexes -> follow the road, +2 Move.
Only where there is a choice of road hex (the first step, or a junction) does it use
closer, then the usual tie order. It doesn't stop at bends, and can't pass units.
"""
import math, sys, yaml

DIRS = ["N", "NE", "SE", "S", "SW", "NW"]
VEC = [(0, -1, 1), (1, -1, 0), (1, 0, -1), (0, 1, -1), (-1, 1, 0), (-1, 0, 1)]
L = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
S3 = math.sqrt(3)


class Map:
    def __init__(self, path):
        m = yaml.safe_load(open(path))
        self.cols, self.rows, self.terrain = m["columns"], m["rows"], m["terrain"]
        self.river = set(m.get("river", [])); self.bridges = set(m.get("bridges", []))
        self.roads = m.get("roads", {})
        self.road = {P(x) for ch in self.roads.values() for x in ch}
        self.road_links = {frozenset((P(a), P(b))) for ch in self.roads.values() for a, b in zip(ch, ch[1:])}

    def inb(self, h): return 0 <= h[0] < self.cols and 0 <= h[1] < self.rows
    def forest(self, h): return self.terrain[h[1]][h[0]] == "F"
    def river_blocks(self, a, b): e = edge(a, b); return e in self.river and e not in self.bridges
    def nbrs(self, h):
        q, r, s = cube(h)
        for d, (a, b, c) in zip(DIRS, VEC):
            o = offc(q + a, r + b, s + c)
            if self.inb(o): yield d, o


def P(n): return (L.index(n[0]), int(n[1:]) - 1)
def N(h): return f"{L[h[0]]}{h[1] + 1}"
def cube(h): c, r = h; q = c; rr = r - (c - (c & 1)) // 2; return (q, rr, -q - rr)
def offc(q, r, s): return (q, r + (q - (q & 1)) // 2)
def center(h): c, r = h; return (1.5 * c, S3 * (r + 0.5 * (c & 1)))
def hexdist(a, b): A, B = cube(a), cube(b); return max(abs(A[i] - B[i]) for i in range(3))
def crow(a, b): (x1, y1), (x2, y2) = center(a), center(b); return math.hypot(x1 - x2, y1 - y2)
def edge(a, b): return f"{N(a)}|{N(b)}" if a < b else f"{N(b)}|{N(a)}"


def keep_left(options):
    """Of tied options, the one anticlockwise of the others (on the left facing the destination)."""
    for c in options:
        if all(((DIRS.index(o[0]) - DIRS.index(c[0])) % 6) in (1, 2) for o in options if o is not c):
            return c
    return options[0]


def move_order(mp, start, dest, move, enemies=(), friends=(), target=None, budget_half=False):
    """Carry out one Move (or Close) order. Returns (end hex, steps, why_stopped).
    Each step is a dict: hex, kind ('closer'|'slide'|'road'), options [(hex, notes)], reason."""
    occupied = set(enemies) | set(friends)
    def next_to_enemy(h):
        return any(e != target and o == e and not mp.river_blocks(h, e) for _, o in mp.nbrs(h) for e in enemies)
    def contact(h):
        return any(o == e and not mp.river_blocks(h, e) for _, o in mp.nbrs(h) for e in enemies)
    road_march = start in mp.road and dest in mp.road and target is None
    budget = (move // 2 if budget_half else move) + (2 if road_march else 0)
    cur, used, visited, steps = start, 0, {start}, []
    while True:
        if cur == dest: return cur, steps, "reached the destination"
        if target is not None and target in [o for _, o in mp.nbrs(cur)] and not mp.river_blocks(cur, target):
            return cur, steps, "next to the target"
        open_ = [(d, o) for d, o in mp.nbrs(cur)
                 if o not in occupied and o not in visited and not mp.river_blocks(cur, o)]
        if road_march:
            opts = [(d, o) for d, o in open_ if frozenset((cur, o)) in mp.road_links]
            kind = "road"
            if not opts:
                return cur, steps, "the road ahead is blocked or ends"
            if len(opts) > 1:  # a choice of road: prefer closer
                here_best = min(hexdist(o, dest) for _, o in opts)
                opts = [x for x in opts if hexdist(x[1], dest) == here_best]
        else:
            here = hexdist(cur, dest)
            closer = [x for x in open_ if hexdist(x[1], dest) < here]
            slide = [x for x in open_ if hexdist(x[1], dest) == here and crow(x[1], dest) < crow(cur, dest) - 1e-9]
            opts, kind = (closer, "closer") if closer else (slide, "slide")
            if not opts:
                return cur, steps, "no open hex gets any closer"
        # the tie order: not next to an enemy, then not forest, then keep left
        reason = "only option" if len(opts) == 1 else ""
        pick = opts
        if len(pick) > 1:
            safe = [x for x in pick if not next_to_enemy(x[1])]
            if safe and len(safe) < len(pick): pick, reason = safe, "avoids the enemy"
        if len(pick) > 1:
            clear = [x for x in pick if not mp.forest(x[1])]
            if clear and len(clear) < len(pick): pick, reason = clear, "avoids the forest"
        if len(pick) > 1:
            pick, reason = [keep_left(pick)], "keep left"
        d, o = pick[0]
        cost = 1 if road_march else (2 if mp.forest(o) else 1)
        if steps and used + cost > budget:
            return cur, steps, f"can't pay {cost} for {N(o)} ({budget - used} Move left)"
        used += cost
        notes = lambda h: ", ".join(x for x in [("forest" if mp.forest(h) else ""), ("next to enemy" if next_to_enemy(h) else "")] if x)
        steps.append({"hex": o, "dir": d, "kind": kind, "reason": reason,
                      "options": [(N(h), notes(h)) for _, h in opts]})
        visited.add(o); cur = o
        if cur == dest: return cur, steps, "reached the destination"
        if contact(o) and (target is None or o not in [h for _, h in mp.nbrs(target)]):
            return cur, steps, f"contact at {N(o)}"
        if used >= budget:
            return cur, steps, "used all its Move"


def explain(mp, start, dest, move, **kw):
    cur, steps, why = move_order(mp, P(start), P(dest), move,
                                 enemies=[P(x) for x in kw.get("enemies", [])], friends=[P(x) for x in kw.get("friends", [])])
    lines = []
    for i, s in enumerate(steps, 1):
        opts = "; ".join(f"{h}" + (f" ({n})" if n else "") for h, n in s["options"])
        lines.append(f"  step {i}: {s['kind']} options {opts} -> {N(s['hex'])} [{s['reason']}]")
    lines.append(f"  stops at {N(cur)}: {why}")
    return lines


EXAMPLES = [  # title, start, dest, move, enemies, friends
    ("Across the river, next to a bridge", "J14", "S13", 4, [], []),
    ("Across the river, far from a bridge", "H4", "P2", 4, [], []),
    ("Straight through the forest", "R12", "R4", 4, [], []),
    ("Into the forest", "H13", "M9", 4, [], []),
    ("Road march over the upper bridge", "K7", "S5", 4, [], []),
    ("Past an enemy: ties let it dodge", "D9", "K9", 4, ["H7"], []),
    ("Past an enemy: one straight line, so contact", "G13", "G3", 4, ["H8"], []),
    ("A friend in the way: keep left", "E14", "E6", 4, [], ["E11"]),
]


def multi_turn(mp, start, dest, move, enemies, friends, turns=6):
    cur, out = P(start), []
    for t in range(1, turns + 1):
        end, steps, why = move_order(mp, cur, P(dest), move, [P(x) for x in enemies], [P(x) for x in friends])
        out += [(s, t) for s in steps]
        if not steps or end == P(dest) or why.startswith("contact") or why.startswith("no open") or why.startswith("the road"):
            return out, f"turn {t}: {why}" if steps else f"stops at {N(end)}: {why}"
        cur = end
    return out, "still going"


def draw(mp, path):
    import matplotlib; matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    from matplotlib.patches import RegularPolygon
    fig, axes = plt.subplots(4, 2, figsize=(22, 30), dpi=80)
    tc = ["#1f3a93", "#2e86de", "#48b1bf", "#7f8fa6", "#a29bfe", "#636e72"]
    for ax, (title, start, dest, move, enemies, friends) in zip(axes.flat, EXAMPLES):
        for c in range(mp.cols):
            for r in range(mp.rows):
                t = mp.terrain[r][c]; X, Y = center((c, r))
                f = {"F": "#9cbf8a", "T": "#c9a98a", "H": "#dcc6ad"}.get(t, "#f4f1e8")
                ax.add_patch(RegularPolygon((X, -Y), 6, radius=1, orientation=math.radians(30), facecolor=f, edgecolor="#c8c2b3", lw=.4))
        for ch in mp.roads.values():
            pts = [center(P(x)) for x in ch]; ax.plot([p[0] for p in pts], [-p[1] for p in pts], color="#8a6a3a", lw=1.6)
        def ep(e):
            a, b = (P(x) for x in e.split("|")); (x1, y1), (x2, y2) = center(a), center(b)
            mx, my = (x1 + x2) / 2, (y1 + y2) / 2; n = math.hypot(x2 - x1, y2 - y1); ux, uy = -(y2 - y1) / n, (x2 - x1) / n
            return (mx + ux * .5, mx - ux * .5), (-(my + uy * .5), -(my - uy * .5))
        for e in mp.river: ax.plot(*ep(e), color="#2f6db5", lw=3)
        for e in mp.bridges: ax.plot(*ep(e), color="#222", lw=6); ax.plot(*ep(e), color="#c9a98a", lw=3)
        for lab, units, col in (("EN", enemies, "#c0392b"), ("FR", friends, "#1f3a93")):
            for u in units:
                X, Y = center(P(u)); ax.add_patch(RegularPolygon((X, -Y), 6, radius=.75, orientation=math.radians(30), facecolor=col, zorder=6))
                ax.text(X, -Y, lab, ha="center", va="center", color="white", fontsize=7, weight="bold", zorder=7)
        steps, note = multi_turn(mp, start, dest, move, enemies, friends)
        pts = [center(P(start))] + [center(s["hex"]) for s, _ in steps]
        for i, (s, t) in enumerate(steps):
            a, b = pts[i], pts[i + 1]
            ax.annotate("", xy=(b[0], -b[1]), xytext=(a[0], -a[1]), zorder=8,
                        arrowprops=dict(arrowstyle="-|>", color=tc[(t - 1) % 6], lw=2.2, ls="--" if s["kind"] == "slide" else "-"))
            if s["reason"] not in ("only option", ""): ax.plot(b[0], -b[1], "o", ms=5, mfc="none", mec="#e67e22", mew=1.6, zorder=9)
            if i == len(steps) - 1 or steps[i + 1][1] != t:
                ax.text(b[0] + .45, -b[1] + .45, f"T{t}", fontsize=8, weight="bold", color=tc[(t - 1) % 6], zorder=10)
        for h, mk, ms, col in ((start, "o", 11, "#1f3a93"), (dest, "*", 17, "#e67e22")):
            X, Y = center(P(h)); ax.plot(X, -Y, mk, ms=ms, color=col, mec="#222", zorder=9); ax.text(X, -Y - 1, h, ha="center", fontsize=8, weight="bold")
        for c in range(0, mp.cols, 2): ax.text(1.5 * c, 1.1, L[c], ha="center", fontsize=7)
        for r in range(0, mp.rows, 2): ax.text(-1.4, -S3 * r, str(r + 1), ha="right", va="center", fontsize=7)
        ax.set_title(f"{title}\n{start} → {dest}, Move {move}: {note}", fontsize=10.5)
        ax.set_xlim(-2, 1.5 * (mp.cols - 1) + 1.2); ax.set_ylim(-S3 * (mp.rows - .2), 2); ax.set_aspect("equal"); ax.axis("off")
    fig.text(.5, .004, "Solid arrow = closer step. Dashed = slide. Orange ring = a tie, broken by avoiding the enemy, then forest, then keeping left. "
             "T1, T2… = where each turn ends (same order each turn).", ha="center", fontsize=11)
    fig.tight_layout(rect=(0, .012, 1, 1)); fig.savefig(path, bbox_inches="tight")


if __name__ == "__main__":
    mp = Map(sys.argv[1])
    for title, start, dest, move, enemies, friends in EXAMPLES:
        print(f"{title}: {start} -> {dest}, Move {move}" + (f", enemies {enemies}" if enemies else "") + (f", friends {friends}" if friends else ""))
        print("\n".join(explain(mp, start, dest, move, enemies=enemies, friends=friends)))
    if len(sys.argv) > 2: draw(mp, sys.argv[2])
