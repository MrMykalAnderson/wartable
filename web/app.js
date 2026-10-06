// Wartable web viewer (docs/dev-plan.md section 7, M6): draws the hex
// map with units and facing, loads a saved game via /api/game, and runs
// a turn via /api/turn, stepping through its event log one at a time.
//
// The map always shows the board's current state (as loaded, or as
// returned after running a turn) — stepping through events narrates
// what happened and highlights the unit involved, rather than replaying
// each hex-by-hex position change.

const HEX_SIZE = 28;
const MARGIN = HEX_SIZE * 2;

// Direction order matches internal/hex.Directions: clockwise from N.
const DIRECTIONS = ["N", "NE", "SE", "S", "SW", "NW"];

const SIDE_COLORS = {
  north: "#6f93c9",
  south: "#c97b6f",
};

const TYPE_SYMBOLS = {
  infantry: "cross",
  cavalry: "slash",
  artillery: "dot",
};

let state = {
  game: null,
  events: [],
  stepIndex: -1, // -1 means "no event selected yet".
};

function hexCenter(col, row) {
  const x = MARGIN + col * HEX_SIZE * 1.5;
  const y = MARGIN + row * HEX_SIZE * Math.sqrt(3) + (col % 2 === 1 ? (HEX_SIZE * Math.sqrt(3)) / 2 : 0);
  return { x, y };
}

// hexVertices returns the 6 corners of a flat-top hexagon centered at
// (cx,cy) with the given radius, starting at angle 0 (east point) and
// going clockwise in screen coordinates.
function hexVertices(cx, cy, radius) {
  const pts = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 180) * (60 * i);
    pts.push([cx + radius * Math.cos(angle), cy + radius * Math.sin(angle)]);
  }
  return pts;
}

// edgeVertexIndices maps a direction index (0=N..5=NW) to the pair of
// hexVertices indices bounding that edge. N is the top (flat) edge,
// between the vertices at 240° and 300°; the rest follow clockwise.
function edgeVertexIndices(directionIndex) {
  const a = (4 + directionIndex) % 6;
  const b = (5 + directionIndex) % 6;
  return [a, b];
}

function svgEl(tag, attrs) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", tag);
  for (const [k, v] of Object.entries(attrs || {})) el.setAttribute(k, v);
  return el;
}

function columnLabel(col) {
  let n = col + 1;
  let s = "";
  while (n > 0) {
    n--;
    s = String.fromCharCode(65 + (n % 26)) + s;
    n = Math.floor(n / 26);
  }
  return s;
}

function renderMap(game) {
  const svg = document.getElementById("map");
  svg.innerHTML = "";
  const width = MARGIN * 2 + (game.columns - 1) * HEX_SIZE * 1.5 + HEX_SIZE;
  const height = MARGIN * 2 + (game.rows - 1) * HEX_SIZE * Math.sqrt(3) + HEX_SIZE * Math.sqrt(3);
  svg.setAttribute("width", width);
  svg.setAttribute("height", height);

  // Grid.
  for (let row = 0; row < game.rows; row++) {
    for (let col = 0; col < game.columns; col++) {
      const { x, y } = hexCenter(col, row);
      const pts = hexVertices(x, y, HEX_SIZE);
      svg.appendChild(
        svgEl("polygon", {
          points: pts.map((p) => p.join(",")).join(" "),
          fill: "none",
          stroke: "#b9ad90",
          "stroke-width": "1",
        })
      );
      if (row === 0) {
        svg.appendChild(
          svgEl("text", { x: x, y: MARGIN - HEX_SIZE, "text-anchor": "middle", "font-size": "11" })
        ).textContent = columnLabel(col);
      }
      if (col === 0) {
        svg.appendChild(
          svgEl("text", { x: MARGIN - HEX_SIZE, y: y + 4, "text-anchor": "middle", "font-size": "11" })
        ).textContent = String(row + 1);
      }
    }
  }

  // Units.
  for (const unit of game.units) {
    drawUnit(svg, unit);
  }
}

function drawUnit(svg, unit) {
  const { x, y } = hexCenter(unit.col, unit.row);
  const radius = HEX_SIZE * 0.72;
  const pts = hexVertices(x, y, radius);
  const facingIndex = DIRECTIONS.indexOf(unit.facing);

  const group = svgEl("g", { "data-unit-id": unit.id, class: "unit" + (unit.strength === "half" ? " unit-half" : "") });

  group.appendChild(
    svgEl("polygon", {
      points: pts.map((p) => p.join(",")).join(" "),
      fill: SIDE_COLORS[unit.side] || "#999",
      "fill-opacity": "0.85",
      stroke: "#222",
      "stroke-width": "1",
    })
  );

  // Edge markers: front (red), the two flanks (green), rear (gray),
  // per docs/core-rules.md section 3.3's token convention.
  for (let d = 0; d < 6; d++) {
    const diff = ((d - facingIndex) % 6 + 6) % 6;
    let cls = "unit-rear";
    if (diff === 0) cls = "unit-front";
    else if (diff === 1 || diff === 5) cls = "unit-flank";
    const [ai, bi] = edgeVertexIndices(d);
    group.appendChild(
      svgEl("line", {
        x1: pts[ai][0],
        y1: pts[ai][1],
        x2: pts[bi][0],
        y2: pts[bi][1],
        class: cls,
        "stroke-width": "3",
      })
    );
  }

  // Type symbol.
  const symbol = TYPE_SYMBOLS[unit.type];
  if (symbol === "cross") {
    const r = radius * 0.4;
    group.appendChild(svgEl("line", { x1: x - r, y1: y - r, x2: x + r, y2: y + r, stroke: "#fff", "stroke-width": "2.5" }));
    group.appendChild(svgEl("line", { x1: x - r, y1: y + r, x2: x + r, y2: y - r, stroke: "#fff", "stroke-width": "2.5" }));
  } else if (symbol === "slash") {
    const r = radius * 0.4;
    group.appendChild(svgEl("line", { x1: x - r, y1: y + r, x2: x + r, y2: y - r, stroke: "#fff", "stroke-width": "2.5" }));
  } else if (symbol === "dot") {
    group.appendChild(svgEl("circle", { cx: x, cy: y, r: radius * 0.3, fill: "#fff" }));
  }

  if (unit.state) {
    group.appendChild(
      svgEl("text", { x: x, y: y + radius + 11, "text-anchor": "middle", "font-size": "9" })
    ).textContent = unit.state;
  }

  svg.appendChild(group);
}

function renderGameInfo(game) {
  const el = document.getElementById("game-info");
  const reserveLine = (side) => `${side}: ${(game.reserves[side] || []).join(", ") || "none"}`;
  el.textContent =
    `${game.scenarioId} — turn ${game.turn} — tie-break: ${game.tieBreakHolder} — ` +
    `reserves — ${reserveLine("north")} / ${reserveLine("south")}`;
}

function highlightUnit(unitId) {
  document.querySelectorAll("#map .unit").forEach((g) => g.classList.remove("selected"));
  if (!unitId) return;
  const g = document.querySelector(`#map .unit[data-unit-id="${CSS.escape(unitId)}"]`);
  if (g) {
    g.classList.add("selected");
    const poly = g.querySelector("polygon");
    if (poly) poly.setAttribute("stroke-width", "4");
  }
}

function renderEventList() {
  const list = document.getElementById("event-list");
  list.innerHTML = "";
  state.events.forEach((e, i) => {
    const li = document.createElement("li");
    li.textContent = e.summary;
    if (i === state.stepIndex) li.classList.add("current");
    list.appendChild(li);
  });
}

function renderStep() {
  const counter = document.getElementById("step-counter");
  const summary = document.getElementById("step-summary");
  counter.textContent = `${state.events.length ? state.stepIndex + 1 : 0} / ${state.events.length}`;
  if (state.stepIndex >= 0 && state.stepIndex < state.events.length) {
    const e = state.events[state.stepIndex];
    summary.textContent = e.summary;
    highlightUnit(e.unit);
  } else {
    summary.textContent = "";
    highlightUnit(null);
  }
  renderEventList();
}

async function loadGame(path) {
  const status = document.getElementById("load-status");
  status.textContent = "";
  try {
    const res = await fetch(`/api/game?path=${encodeURIComponent(path)}`);
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    state.game = body;
    state.events = [];
    state.stepIndex = -1;
    renderMap(state.game);
    renderGameInfo(state.game);
    renderStep();
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

async function runTurn(path, north, south) {
  const status = document.getElementById("turn-status");
  status.textContent = "";
  try {
    const res = await fetch("/api/turn", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, north, south }),
    });
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    state.game = body.game;
    state.events = body.events || [];
    state.stepIndex = state.events.length ? 0 : -1;
    renderMap(state.game);
    renderGameInfo(state.game);
    if (body.outcome && body.outcome.over) {
      const who = body.outcome.winner ? `${body.outcome.winner} wins` : "draw";
      status.textContent = `Game over: ${who} (${body.outcome.reason})`;
    }
    renderStep();
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

document.getElementById("load").addEventListener("click", () => {
  loadGame(document.getElementById("path").value.trim());
});

document.getElementById("run-turn").addEventListener("click", () => {
  const path = document.getElementById("path").value.trim();
  const north = document.getElementById("north-orders").value;
  const south = document.getElementById("south-orders").value;
  runTurn(path, north, south);
});

document.getElementById("step-prev").addEventListener("click", () => {
  if (state.stepIndex > 0) {
    state.stepIndex--;
    renderStep();
  }
});

document.getElementById("step-next").addEventListener("click", () => {
  if (state.stepIndex < state.events.length - 1) {
    state.stepIndex++;
    renderStep();
  }
});
