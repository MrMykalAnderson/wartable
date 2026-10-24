"""Unit token mockup (NATO APP-6 style). Writes an HTML page of SVG tokens; render it to PNG with a browser.

    python3 docs/tools/token_mockup.py out.html          # full mockup: key, zoom levels, battlefield
    python3 docs/tools/token_mockup.py out.html --key    # just the key (docs/images/token-key.png)

Token anatomy (flat-top hex, radius R, centred on the hex):
  * Hex body in the side's colour (North blue, South red): who owns it, readable from far away.
  * A white NATO frame (rectangle, 1.5 : 1) with the black type symbol:
      Infantry  = X (both diagonals)
      Cavalry   = one diagonal, bottom-left to top-right
      Artillery = solid dot; Mobilised adds the NATO "towed" mobility marker (two wheels and an axle)
  * Echelon (unit size) above the frame: I company, II battalion, III regiment, X brigade.
  * Designation (arabic numeral) below the frame.
  * Facing on the hex edges: front yellow with a dark casing (thick), flanks thin white.
  * Half strength: body washed out, and the NATO "reduced" mark (−) at the frame's top right.
Levels of detail, so the map reads at a glance and still identifies units close up:
  far  : body, frame, symbol, front edge
  mid  : + echelon, number, flank edges, reduced mark
  close: + state text (READY / MOBILISED)
"""
import math, sys

SIDE = {"north": ("#2f5fa8", "#9db6dc"), "south": ("#b83a2e", "#e2a59e")}   # (full, washed-out)
ANGLE = {"N": -90, "NE": -30, "SE": 30, "S": 90, "SW": 150, "NW": 210}
ORDER = ["N", "NE", "SE", "S", "SW", "NW"]
FRONT, FLANK, CASING = "#ffd23f", "#ffffff", True   # front yellow with dark casing, flanks white (core-rules 3.3)


def hex_pts(cx, cy, r):
    return [(cx + r * math.cos(math.radians(a)), cy + r * math.sin(math.radians(a))) for a in range(0, 360, 60)]


def edge(cx, cy, r, d):
    """End points of the hex edge facing direction d (flat-top hex)."""
    a = ANGLE[d]
    p1 = (cx + r * math.cos(math.radians(a - 30)), cy + r * math.sin(math.radians(a - 30)))
    p2 = (cx + r * math.cos(math.radians(a + 30)), cy + r * math.sin(math.radians(a + 30)))
    return p1, p2


def token(cx, cy, r, side, kind, num, echelon="II", facing="N", half=False, state="", lod="close"):
    full, pale = SIDE[side]
    body = pale if half else full
    out = []
    pts = " ".join(f"{x:.1f},{y:.1f}" for x, y in hex_pts(cx, cy, r * 0.93))
    out.append(f'<polygon points="{pts}" fill="{body}" stroke="#1d1d1d" stroke-width="{max(.8, r*.025):.2f}"/>')
    # facing edges
    fi = ORDER.index(facing)
    for i, d in enumerate(ORDER):
        diff = (i - fi) % 6
        if diff == 0:
            col, w = FRONT, max(2.4, r * .14)
        elif diff in (1, 5) and lod != "far":
            col, w = FLANK, max(1.2, r * .06)
        else:
            continue
        (x1, y1), (x2, y2) = edge(cx, cy, r * 0.93, d)
        if diff == 0 and CASING:   # dark casing under the front edge so it reads on any body colour
            out.append(f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" stroke="#111" stroke-width="{w+2.2:.1f}" stroke-linecap="round"/>')
        out.append(f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" stroke="{col}" stroke-width="{w:.1f}" stroke-linecap="round"/>')
    # NATO frame
    fw, fh = r * 0.98, r * 0.62
    fx, fy = cx - fw / 2, cy - fh / 2 - r * 0.04
    sw = max(1, r * 0.05)
    out.append(f'<rect x="{fx:.1f}" y="{fy:.1f}" width="{fw:.1f}" height="{fh:.1f}" fill="#ffffff" stroke="#111" stroke-width="{sw:.2f}"/>')
    ink = f'stroke="#111" stroke-width="{sw:.2f}"'
    if kind == "infantry":
        out.append(f'<line x1="{fx:.1f}" y1="{fy:.1f}" x2="{fx+fw:.1f}" y2="{fy+fh:.1f}" {ink}/>')
        out.append(f'<line x1="{fx:.1f}" y1="{fy+fh:.1f}" x2="{fx+fw:.1f}" y2="{fy:.1f}" {ink}/>')
    elif kind == "cavalry":
        out.append(f'<line x1="{fx:.1f}" y1="{fy+fh:.1f}" x2="{fx+fw:.1f}" y2="{fy:.1f}" {ink}/>')
    elif kind.startswith("artillery"):
        out.append(f'<circle cx="{cx:.1f}" cy="{fy+fh/2:.1f}" r="{fh*0.2:.1f}" fill="#111"/>')
        if kind == "artillery-mobilised":   # NATO towed marker under the frame
            wy = fy + fh + r * 0.1
            out.append(f'<line x1="{cx-fw*0.3:.1f}" y1="{wy:.1f}" x2="{cx+fw*0.3:.1f}" y2="{wy:.1f}" stroke="#111" stroke-width="{sw*0.8:.2f}"/>')
            for dx in (-fw * 0.3, fw * 0.3):
                out.append(f'<circle cx="{cx+dx:.1f}" cy="{wy:.1f}" r="{r*0.07:.1f}" fill="#fff" stroke="#111" stroke-width="{sw*0.8:.2f}"/>')
    if lod != "far":
        # echelon above, number below (white on the coloured body, dark halo for contrast)
        fs = r * 0.27
        out.append(f'<text x="{cx:.1f}" y="{fy - r*0.07:.1f}" font-size="{fs:.1f}" text-anchor="middle" font-family="Arial, sans-serif" font-weight="700" fill="#fff" stroke="#111" stroke-width="{fs*0.12:.2f}" paint-order="stroke">{echelon}</text>')
        ny = fy + fh + (r * 0.38 if kind == "artillery-mobilised" else r * 0.30)
        out.append(f'<text x="{cx:.1f}" y="{ny:.1f}" font-size="{r*0.33:.1f}" text-anchor="middle" font-family="Arial, sans-serif" font-weight="700" fill="#fff" stroke="#111" stroke-width="{r*0.04:.2f}" paint-order="stroke">{num}</text>')
        if half:
            out.append(f'<text x="{fx+fw+r*0.02:.1f}" y="{fy+r*0.04:.1f}" font-size="{r*0.3:.1f}" font-family="Arial, sans-serif" font-weight="700" fill="#111">−</text>')
    if lod == "close" and state:
        out.append(f'<text x="{cx:.1f}" y="{cy + r*1.18:.1f}" font-size="{r*0.2:.1f}" text-anchor="middle" font-family="Arial, sans-serif" fill="#333">{state}</text>')
    return "\n".join(out)


def terrain_hex(cx, cy, r, fill):
    pts = " ".join(f"{x:.1f},{y:.1f}" for x, y in hex_pts(cx, cy, r))
    return f'<polygon points="{pts}" fill="{fill}" stroke="#cfc8b8" stroke-width="0.6"/>'


def key_page():
    R = 46
    kinds = [("infantry", "1", "", False, "Infantry"), ("cavalry", "2", "", False, "Cavalry"),
             ("artillery-ready", "1", "READY", False, "Artillery, Ready"), ("artillery-mobilised", "2", "MOBILISED", False, "Artillery, Mobilised"),
             ("infantry", "3", "", True, "Half strength")]
    svg = [f'<svg width="{150*len(kinds)}" height="320" xmlns="http://www.w3.org/2000/svg">']
    for row, side, face in ((0, "north", "NE"), (1, "south", "SW")):
        for i, (k, n, st, half, lab) in enumerate(kinds):
            cx, cy = 75 + 150 * i, 70 + 150 * row
            svg.append(token(cx, cy, R, side, k, n, facing=face, half=half, state=st))
            if row == 1:
                svg.append(f'<text x="{cx}" y="{cy+82}" font-size="13" font-family="Arial, sans-serif" text-anchor="middle">{lab}</text>')
    svg.append("</svg>")
    return "<html><head><meta charset='utf-8'><style>body{margin:8px;background:#fff}</style></head><body>" + "".join(svg) + "</body></html>"


def page():
    css = "body{font-family:Arial,sans-serif;background:#f7f5ef;margin:24px;color:#222}h2{margin:28px 0 6px}p{margin:4px 0 10px;max-width:980px}"
    facing_label = "front yellow with dark casing, flanks white" if CASING else "front red, flanks green (current)"
    parts = [f"<html><head><meta charset='utf-8'><style>{css}</style></head><body>",
             f"<h1>Wartable unit tokens: NATO APP-6 style (mockup)</h1><p><b>Facing colours: {facing_label}</b></p>"]
    # 1. key at close zoom
    R = 46
    kinds = [("infantry", "1", "", False, "Infantry"), ("cavalry", "2", "", False, "Cavalry"),
             ("artillery-ready", "1", "READY", False, "Artillery, Ready"), ("artillery-mobilised", "2", "MOBILISED", False, "Artillery, Mobilised"),
             ("infantry", "3", "", True, "Infantry, half strength")]
    parts.append("<h2>1. The set, close up</h2><p>Side by body colour; type by the NATO symbol in the white frame; size (II = battalion) above; number below; front edge and flanks marked on the hex edges. Half strength: washed-out body and the NATO reduced mark (−).</p>")
    w = 150 * len(kinds)
    svg = [f'<svg width="{w}" height="{2*150+10}" xmlns="http://www.w3.org/2000/svg">']
    for row, side, face in ((0, "north", "NE"), (1, "south", "SW")):
        for i, (k, n, st, half, lab) in enumerate(kinds):
            cx, cy = 75 + 150 * i, 70 + 150 * row
            svg.append(token(cx, cy, R, side, k, n, facing=face, half=half, state=st))
            if row == 1:
                svg.append(f'<text x="{cx}" y="{cy+82}" font-size="12" text-anchor="middle">{lab}</text>')
    svg.append("</svg>"); parts.append("".join(svg))
    # 2. levels of detail
    parts.append("<h2>2. Levels of detail as you zoom</h2><p>Far: colour, symbol and front edge only, so the battlefield reads at a glance. Mid: size, number, flanks and strength appear. Close: state text.</p>")
    svg = ['<svg width="900" height="170" xmlns="http://www.w3.org/2000/svg">']
    x = 40
    for lod, r in (("far", 16), ("mid", 30), ("close", 46)):
        for k, n, st in (("infantry", "1", ""), ("cavalry", "2", ""), ("artillery-mobilised", "1", "MOBILISED")):
            svg.append(token(x + r, 80, r, "north", k, n, facing="N", state=st, lod=lod)); x += 2 * r + 14
        svg.append(f'<text x="{x - 3*(r+7) - r/2:.0f}" y="160" font-size="13" text-anchor="middle" font-weight="700">{lod} (hex radius {r}px)</text>')
        x += 40
    svg.append("</svg>"); parts.append("".join(svg))
    # 3. a slice of battlefield at far zoom
    parts.append("<h2>3. At a glance: a slice of battlefield, far zoom</h2><p>Two lines facing each other across open ground and forest. Can you tell, without reading anything, where the guns and cavalry are and which way everyone faces?</p>")
    r = 18; cols, rows = 22, 9
    svg = [f'<svg width="{int(1.5*r*cols+2*r)}" height="{int(math.sqrt(3)*r*(rows+1))}" xmlns="http://www.w3.org/2000/svg">']
    forest = {(c, rr) for c in range(9, 14) for rr in range(3, 8)}
    centers = {}
    for c in range(cols):
        for rr in range(rows):
            cx = r + 1.5 * r * c; cy = r + math.sqrt(3) * r * (rr + 0.5 * (c % 2)); centers[(c, rr)] = (cx, cy)
            svg.append(terrain_hex(cx, cy, r, "#9cbf8a" if (c, rr) in forest else "#f2eee3"))
    north = [((3, 2), "artillery-ready", "1", False), ((5, 2), "infantry", "1", False), ((5, 4), "infantry", "2", False), ((6, 3), "infantry", "3", True),
             ((5, 6), "infantry", "4", False), ((3, 5), "cavalry", "1", False), ((2, 7), "artillery-mobilised", "2", False)]
    south = [((18, 2), "artillery-ready", "1", False), ((16, 3), "infantry", "1", False), ((16, 5), "infantry", "2", False), ((15, 4), "cavalry", "2", False),
             ((17, 6), "infantry", "3", False), ((19, 7), "cavalry", "1", True)]
    for pos, k, n, half in north: svg.append(token(*centers[pos], r, "north", k, n, facing="SE", half=half, lod="far"))
    for pos, k, n, half in south: svg.append(token(*centers[pos], r, "south", k, n, facing="NW", half=half, lod="far"))
    svg.append("</svg>"); parts.append("".join(svg))
    # 4. same slice, mid zoom (cropped)
    parts.append("<h2>4. The same units, mid zoom</h2><p>Zoomed in, each unit can be identified: North's 3rd battalion is at half strength; South's 1st cavalry too.</p>")
    r = 32; svg = [f'<svg width="{int(1.5*r*9+2*r)}" height="{int(math.sqrt(3)*r*6.5)}" xmlns="http://www.w3.org/2000/svg">']
    for c in range(9):
        for rr in range(6):
            cx = r + 1.5 * r * c; cy = r + math.sqrt(3) * r * (rr + 0.5 * (c % 2)); centers[(c + 1, rr + 1)] = (cx, cy)
            svg.append(terrain_hex(cx, cy, r, "#f2eee3"))
    for pos, k, n, half in north:
        if 1 <= pos[0] <= 9 and 1 <= pos[1] <= 6: svg.append(token(*centers[pos], r, "north", k, n, facing="SE", half=half, lod="mid"))
    svg.append("</svg>"); parts.append("".join(svg))
    parts.append("</body></html>")
    return "\n".join(parts)


if __name__ == "__main__":
    open(sys.argv[1], "w").write(key_page() if "--key" in sys.argv else page())
