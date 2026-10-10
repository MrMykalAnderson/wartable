"""Convert Mykal's square-grid paper map to a 26x18 flat-top hex map (odd-q, A1 top-left).

Paper coordinates: x = column index (A=0), y = row index (row 1 = 0), read off the photo.
"""
import math, sys, yaml
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import RegularPolygon

COLS, ROWS = 26, 18
L = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
S3 = math.sqrt(3)
DIRS = [("N", (0, -1, 1)), ("NE", (1, -1, 0)), ("SE", (1, 0, -1)), ("S", (0, 1, -1)), ("SW", (-1, 1, 0)), ("NW", (-1, 0, 1))]

def name(h): return f"{L[h[0]]}{h[1]+1}"
def cube(c, r): q = c; rr = r - (c - (c & 1)) // 2; return (q, rr, -q - rr)
def offc(q, r, s): return (q, r + (q - (q & 1)) // 2)
def center(h): c, r = h; return (1.5 * c, S3 * (r + 0.5 * (c & 1)))
def paper(x, y): return (1.5 * x, S3 * (y + 0.25))
def inb(h): return 0 <= h[0] < COLS and 0 <= h[1] < ROWS
def nbrs(h):
    q, r, s = cube(*h)
    for d, (a, b, c) in DIRS:
        o = offc(q + a, r + b, s + c)
        if inb(o): yield d, o
HEXES = [(c, r) for c in range(COLS) for r in range(ROWS)]

def nearest(x, y):
    X, Y = paper(x, y)
    return min(HEXES, key=lambda h: (center(h)[0] - X) ** 2 + (center(h)[1] - Y) ** 2)

def cube_round(q, r, s):
    rq, rr, rs = round(q), round(r), round(s)
    dq, dr, ds = abs(rq - q), abs(rr - r), abs(rs - s)
    if dq > dr and dq > ds: rq = -rr - rs
    elif dr > ds: rr = -rq - rs
    else: rs = -rq - rr
    return rq, rr, rs

def line(a, b):
    A, B = cube(*a), cube(*b)
    n = max(abs(A[i] - B[i]) for i in range(3))
    out = []
    for i in range(n + 1):
        t = i / n if n else 0
        p = [A[k] + (B[k] - A[k]) * t + 1e-6 * (k + 1) for k in range(3)]
        h = offc(*cube_round(*p))
        if not out or out[-1] != h: out.append(h)
    return out

def chain(points):
    hs = [nearest(x, y) for x, y in points]
    out = []
    for a, b in zip(hs, hs[1:]):
        for h in line(a, b):
            if not out or out[-1] != h: out.append(h)
    return out

def seg_x(p1, p2, p3, p4):
    def o(a, b, c): return (b[0] - a[0]) * (c[1] - a[1]) - (b[1] - a[1]) * (c[0] - a[0])
    return o(p1, p2, p3) * o(p1, p2, p4) < 0 and o(p3, p4, p1) * o(p3, p4, p2) < 0

def in_poly(pt, poly):
    x, y = pt; inside = False
    for (x1, y1), (x2, y2) in zip(poly, poly[1:] + poly[:1]):
        if (y1 > y) != (y2 > y) and x < x1 + (y - y1) * (x2 - x1) / (y2 - y1): inside = not inside
    return inside

# ---- features read off the photo -------------------------------------------------
px = lambda X, Y: ((Y - 265) / 66.8, (1390 - X) / 69.4)   # photo pixel -> paper (x, y)

TOWNS = {"North town": [(1, 14), (1, 15), (2, 14), (2, 15), (3, 14), (3, 15)],
         "South town": [(22, 1), (22, 2), (22, 3), (23, 1), (23, 2), (23, 3)]}
HAMLETS = {"West hamlet": [(1, 10), (1, 11)], "East hamlet": [(24, 9), (25, 9)]}

RIVER = [(10.0, -0.6), (9.95, 2.7), (10.9, 4.2), (12.0, 5.3), (12.9, 6.6), (13.4, 7.8), (13.5, 8.8),
         (13.8, 9.9), (14.1, 10.95), (15.3, 12.7), (15.5, 15.7), (15.6, 16.4), (16.1, 16.9), (16.2, 17.8)]
ROADS = {
    "R1": [(3.1, 17.4), (2.4, 15.1), (2.0, 13.4), (1.2, 10.7), (0.8, 7.5), (0, 7.0)],
    "R2": [(0, 15.4), (0.8, 14.8), (3.5, 14.7), (6.4, 14.3), (7.4, 13.8), (10.4, 12.5), (13.2, 11.4),
           (14.1, 11.0), (16.2, 10.4), (18.5, 10.1), (21.5, 9.7), (23.9, 9.1), (25.4, 8.6)],
    "R3": [(-0.3, 11.2), (1.3, 10.6), (3.9, 9.5), (5.2, 8.2), (6.2, 6.6), (7.7, 5.9), (8.5, 5.8), (10.3, 5.8),
           (11.6, 5.55), (12.0, 5.3), (13.7, 4.6), (15.0, 3.9), (16.1, 3.5), (18.2, 3.4), (20.2, 3.0),
           (21.3, 2.6), (22.6, 2.0), (24.0, 1.6), (25.4, 1.5)],
    "R4": [(22.0, -0.1), (22.5, 2.2), (23.15, 3.36), (23.15, 4.2), (23.7, 5.5), (24.25, 6.9), (24.5, 8.9),
           (25.06, 10.1), (25.5, 11.1)],
}
FOREST_PX = [(665, 850), (800, 850), (865, 915), (930, 985), (1000, 1050), (1065, 1130), (1065, 1370),
             (1000, 1370), (1000, 1680), (935, 1680), (935, 1800), (870, 1800), (870, 1680), (800, 1680),
             (800, 1550), (735, 1550), (735, 1300), (665, 1300)]

# ---- build ---------------------------------------------------------------------------
# Each hex is on the east (South) bank if its centre lies east of the river line.
east_poly = [paper(RIVER[0][0], -10)] + [paper(x, y) for x, y in RIVER] + [paper(RIVER[-1][0], 40), paper(60, 40), paper(60, -10)]
east = {h for h in HEXES if in_poly(center(h), east_poly)}
river_edges = set()
for h in HEXES:
    for _, o in nbrs(h):
        if h < o and ((h in east) != (o in east)):
            river_edges.add((h, o))

roads = {k: chain(v) for k, v in ROADS.items()}
bridges = set()
for k, ch in roads.items():
    for a, b in zip(ch, ch[1:]):
        e = (min(a, b), max(a, b))
        if e in river_edges: bridges.add(e)

FOREST_COLS = {"J": (10, 11), "K": (9, 11), "L": (8, 11), "M": (7, 11), "N": (7, 11), "O": (7, 11),
               "P": (6, 11), "Q": (6, 10), "R": (7, 9), "S": (8, 8)}
settled = {h for v in list(TOWNS.values()) + list(HAMLETS.values()) for h in v}
forest_raw = {(L.index(c), n - 1) for c, (a, b) in FOREST_COLS.items() for n in range(a, b + 1)} - settled
road_hexes = {h for ch in roads.values() for h in ch}
print("forest cleared from road hexes:", sorted(name(h) for h in forest_raw & road_hexes))
forest = forest_raw - road_hexes   # roads are cleared through the forest

# which side of the river is each hex on (flood fill without crossing river edges)
def side_fill(start):
    seen = {start}; st = [start]
    while st:
        h = st.pop()
        for _, o in nbrs(h):
            e = (min(h, o), max(h, o))
            if o not in seen and e not in river_edges: seen.add(o); st.append(o)
    return seen
north_side = side_fill((2, 15)); south_side = side_fill((23, 2))

# ---- report --------------------------------------------------------------------------
print("river edges:", len(river_edges))
print("bridges:", sorted(f"{name(a)}|{name(b)}" for a, b in bridges))
print("forest hexes:", len(forest))
print("north bank hexes:", len(north_side), " south bank hexes:", len(south_side), " total:", COLS * ROWS)
for k, ch in roads.items(): print(k, len(ch), " ".join(name(h) for h in ch))

# ---- yaml ----------------------------------------------------------------------------
grid = []
for r in range(ROWS):
    row = ""
    for c in range(COLS):
        h = (c, r)
        row += "T" if any(h in v for v in TOWNS.values()) else "H" if any(h in v for v in HAMLETS.values()) else "F" if h in forest else "."
    grid.append(row)
data = {
    "id": "two-towns", "name": "Two Towns (draft)", "columns": COLS, "rows": ROWS,
    "terrain_key": {".": "open", "F": "forest", "T": "town", "H": "hamlet"},
    "terrain": grid,
    "towns": {k: [name(h) for h in v] for k, v in TOWNS.items()},
    "hamlets": {k: [name(h) for h in v] for k, v in HAMLETS.items()},
    "roads": {k: [name(h) for h in v] for k, v in roads.items()},
    "river": sorted(f"{name(a)}|{name(b)}" for a, b in river_edges),
    "bridges": sorted(f"{name(a)}|{name(b)}" for a, b in bridges),
}

# ---- render --------------------------------------------------------------------------
def render(path):
    fig, ax = plt.subplots(figsize=(16, 11), dpi=110)
    size = 1.0
    col = {"open": "#f4f1e8", "forest": "#9cbf8a", "town": "#c9a98a", "hamlet": "#dcc6ad"}
    for h in HEXES:
        X, Y = center(h)
        t = "town" if grid[h[1]][h[0]] == "T" else "hamlet" if grid[h[1]][h[0]] == "H" else "forest" if h in forest else "open"
        ax.add_patch(RegularPolygon((X, -Y), 6, radius=size, orientation=math.radians(30),
                                    facecolor=col[t], edgecolor="#b8b2a3", lw=0.6))
        ax.text(X, -Y - 0.55, name(h), ha="center", va="center", fontsize=4.2, color="#7d776a")
    for ch in roads.values():
        xs = [center(h)[0] for h in ch]; ys = [-center(h)[1] for h in ch]
        ax.plot(xs, ys, color="#6b4f2a", lw=2.2, solid_capstyle="round", zorder=3)
    def edge_pts(a, b):
        (ax_, ay), (bx, by) = center(a), center(b)
        mx, my = (ax_ + bx) / 2, (ay + by) / 2
        dx, dy = bx - ax_, by - ay; n = math.hypot(dx, dy); ux, uy = -dy / n, dx / n
        half = 0.5 * size
        return (mx + ux * half, mx - ux * half), (-(my + uy * half), -(my - uy * half))
    for a, b in river_edges:
        xs, ys = edge_pts(a, b)
        ax.plot(xs, ys, color="#2f6db5", lw=4, solid_capstyle="round", zorder=4)
    for a, b in bridges:
        xs, ys = edge_pts(a, b)
        ax.plot(xs, ys, color="#222", lw=7, zorder=5)
        ax.plot(xs, ys, color="#c9a98a", lw=3.5, zorder=6)
    for c in range(COLS):
        ax.text(1.5 * c, 1.3, L[c], ha="center", fontsize=9, weight="bold")
    for r in range(ROWS):
        ax.text(-1.6, -S3 * r, str(r + 1), ha="right", va="center", fontsize=9, weight="bold")
    for k, v in TOWNS.items():
        X = sum(center(h)[0] for h in v) / len(v); Y = sum(center(h)[1] for h in v) / len(v)
        ax.text(X, -Y, k, ha="center", va="center", fontsize=8, weight="bold", zorder=7)
    ax.set_xlim(-2.5, 1.5 * (COLS - 1) + 1.5); ax.set_ylim(-S3 * (ROWS - 0.2), 2.2)
    ax.set_aspect("equal"); ax.axis("off")
    ax.set_title("Two Towns: hex conversion (draft). Blue = river (hex edges), dark bars = bridges, brown = roads", fontsize=11)
    fig.savefig(path, bbox_inches="tight"); plt.close(fig)

if len(sys.argv) > 1:
    out = sys.argv[1]
    class Q(str): pass
    yaml.SafeDumper.add_representer(Q, lambda d, v: d.represent_scalar("tag:yaml.org,2002:str", v, style="'"))
    data["terrain"] = [Q(r) for r in data["terrain"]]
    header = ("# Two Towns map. Rules: docs/core-rules.md 2.3; scenario: docs/two-towns.md.\n"
              "# Flat-top hexes, A1 top-left, columns A-Z, rows 1-18; B, D, F ... sit half a hex lower.\n"
              "# terrain: one string per row, one character per column (see terrain_key).\n"
              "# roads: hex chains through hex centres. river/bridges: hex edges, written HEX|HEX.\n"
              "# Converted from the paper map (docs/maps/two-towns-paper.jpg) by docs/maps/twotowns.py.\n")
    open(out + ".yaml", "w").write(header + yaml.safe_dump(data, sort_keys=False, width=200))
    render(out + ".png")
