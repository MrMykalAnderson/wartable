// Wartable web viewer and order builder (docs/dev-plan.md section 7, M6
// and M7.1): draws the hex map with units and facing, loads a saved
// game, builds orders by clicking the map, runs a turn, and steps
// through its event log one event at a time.
//
// Order previews (which hexes/enemies a unit can be ordered against) are
// never computed here: they always come from GET /api/options, which
// asks the engine. This file only turns clicks into order objects and
// renders what the server already decided.
//
// The map replays the board exactly as the engine saw it: each event
// from /api/turn carries its own board snapshot (taken right after that
// event), and the viewer can step through both the events of a turn and
// past turns themselves. Combat events (melee/ranged) also carry every
// number the engine used, broken out, so stepping through shows the
// full calculation rather than just the final hit/miss — again, never
// recomputed here, only formatted.

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
  game: null, // always the live/current state, used for order-building.
  turnHistory: [], // [{turn, boardBefore, events}], oldest first.
  historyIndex: -1, // index into turnHistory being viewed, or -1 for "live".
  stepIndex: -1, // -1 = before the viewed turn's first event.
  activeSide: "north",
  orders: { north: [], south: [] },
  selection: null, // unit ID currently being given an order, or null.
  options: null, // the selected unit's /api/options response.
  pendingOrder: null, // {type, unit, targetHex?, targetUnit?} awaiting a facing pick.
};

// ---- hex geometry -----------------------------------------------------

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

function hexName(col, row) {
  return `${columnLabel(col)}${row + 1}`;
}

// ---- map rendering ------------------------------------------------------

function renderMap(game) {
  const svg = document.getElementById("map");
  svg.innerHTML = "";
  const width = MARGIN * 2 + (game.columns - 1) * HEX_SIZE * 1.5 + HEX_SIZE;
  const height = MARGIN * 2 + (game.rows - 1) * HEX_SIZE * Math.sqrt(3) + HEX_SIZE * Math.sqrt(3);
  svg.setAttribute("width", width);
  svg.setAttribute("height", height);

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
          "pointer-events": "all",
          class: "hex-cell",
          "data-col": col,
          "data-row": row,
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

  // A native tooltip, as a lightweight hover summary (see the unit card
  // for the full detail, shown on selection).
  const range = unit.stats.maxRange > 0 ? `${unit.stats.minRange}-${unit.stats.maxRange}` : "-";
  const title = svgEl("title", {});
  title.textContent =
    `${unit.id} (${unit.side}) — ${unit.typeName}, ${unit.strength} strength, facing ${unit.facing}` +
    `${unit.state ? ", " + unit.state : ""}\n` +
    `Def ${unit.stats.def} · Move ${unit.stats.move} · Range ${range} · Attack ${unit.stats.attack}`;
  group.appendChild(title);

  // A ring just outside the token, shown only while targetable: the
  // token's own border is fully covered by the facing-indicator lines
  // below, so the "can be targeted" highlight needs its own element.
  const ringPts = hexVertices(x, y, radius * 1.22);
  group.appendChild(
    svgEl("polygon", {
      points: ringPts.map((p) => p.join(",")).join(" "),
      fill: "none",
      class: "target-ring",
    })
  );

  group.appendChild(
    svgEl("polygon", {
      points: pts.map((p) => p.join(",")).join(" "),
      fill: SIDE_COLORS[unit.side] || "#999",
      "fill-opacity": "0.85",
      stroke: "#222",
      "stroke-width": "1",
      class: "token-body",
    })
  );

  // Edge markers: front (red), the two flanks (green), rear (gray),
  // per docs/core-rules.md section 3.3's token convention.
  for (let d = 0; d < 6; d++) {
    const diff = (((d - facingIndex) % 6) + 6) % 6;
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

// ---- order highlights ---------------------------------------------------

function clearHighlights() {
  document.querySelectorAll("#map .hex-cell").forEach((c) => {
    c.classList.remove("selectable", "highlight-move", "highlight-deploy", "highlight-fire");
  });
  document.querySelectorAll("#map .unit").forEach((g) => g.classList.remove("targetable"));
  const closeLayer = document.getElementById("close-move-layer");
  if (closeLayer) closeLayer.remove();
}

// renderHighlights draws the selected unit's range visualiser (docs/dev-
// plan.md section 7.3, interface item 5): green for the hexes within its
// Move (or, for a reserve unit, the hexes it can Deploy into), red for
// hexes it can fire into (band + arc; for Ready artillery this is also
// its passive Barrage zone), a ring around every enemy it has a valid
// order against, and (docs/dev-plan.md section 7.5, interface item 1) a
// purple dot over the shorter half-move reach a Close order can
// actually use.
//
// For a Move order, the green area is a reach *guide*, not a
// restriction (docs/dev-plan.md section 7.6): every hex on the map is
// clickable, since a Move order's path is only worked out at execution,
// by which time other units will have moved — planning can't know
// what's actually in the way. Deploy stays restricted to DeployHexes,
// since placing a reserve unit is immediate, not a path.
function renderHighlights() {
  clearHighlights();
  const opts = state.options;
  if (!opts) return;

  if (opts.isReserve) {
    for (const h of opts.deployHexes || []) {
      const cell = document.querySelector(`#map .hex-cell[data-col="${h.col}"][data-row="${h.row}"]`);
      if (cell) cell.classList.add("selectable", "highlight-deploy");
    }
  } else {
    document.querySelectorAll("#map .hex-cell").forEach((c) => c.classList.add("selectable"));
    for (const h of opts.moveHexes || []) {
      const cell = document.querySelector(`#map .hex-cell[data-col="${h.col}"][data-row="${h.row}"]`);
      if (cell) cell.classList.add("highlight-move");
    }
  }
  for (const h of opts.fireHexes || []) {
    const cell = document.querySelector(`#map .hex-cell[data-col="${h.col}"][data-row="${h.row}"]`);
    if (cell) cell.classList.add("highlight-fire");
  }
  renderCloseMoveMarkers(opts.closeMoveHexes);

  const targetable = new Set([...(opts.meleeTargets || []), ...(opts.fireTargets || []), ...(opts.closeFireTargets || [])]);
  targetable.forEach((id) => {
    const g = document.querySelector(`#map .unit[data-unit-id="${CSS.escape(id)}"]`);
    if (g) g.classList.add("targetable");
  });
}

function renderCloseMoveMarkers(hexes) {
  if (!hexes || hexes.length === 0) return;
  const svg = document.getElementById("map");
  const layer = svgEl("g", { id: "close-move-layer" });
  svg.appendChild(layer);
  for (const h of hexes) {
    const { x, y } = hexCenter(h.col, h.row);
    layer.appendChild(svgEl("circle", { cx: x, cy: y, r: HEX_SIZE * 0.16, class: "close-move-dot" }));
  }
}

document.getElementById("map").addEventListener("click", (e) => {
  const unitEl = e.target.closest(".unit");
  if (unitEl) {
    onUnitClick(unitEl.getAttribute("data-unit-id"));
    return;
  }
  const cellEl = e.target.closest(".hex-cell");
  if (cellEl && cellEl.classList.contains("selectable")) {
    onHexClick(Number(cellEl.getAttribute("data-col")), Number(cellEl.getAttribute("data-row")));
  }
});

// ---- unit detail card (docs/dev-plan.md section 7.3, interface item 3) --
//
// Shown when a unit is selected for ordering (see selectUnit/
// clearSelection below). Each token also carries a native SVG <title>
// (set in drawUnit) as a lightweight hover tooltip — deliberately not a
// custom JS mouseover/mouseout handler, which is prone to firing
// repeatedly as the pointer crosses a token's internal sub-elements.

const TYPE_DESCRIPTIONS = {
  infantry: "The most basic unit: no ranged attack and a short move. On its own it can only damage an enemy by attacking a flank or the rear.",
  cavalry: "Fast shock troops with both melee and a short-ranged attack.",
  artillery: "Long-ranged guns that must be set up before they can fire, and are vulnerable while moving.",
};

function showUnitCard(unitId) {
  const u = state.game && state.game.units.find((x) => x.id === unitId);
  if (!u) {
    // Not yet deployed (in reserves): no position/facing to show yet.
    hideUnitCard();
    return;
  }
  const card = document.getElementById("unit-card");
  const range = u.stats.maxRange > 0 ? `${u.stats.minRange}-${u.stats.maxRange}` : "-";
  card.innerHTML =
    `<strong>${u.id}</strong> (${u.side})<br>` +
    `${u.typeName}, ${u.strength} strength, facing ${u.facing}${u.state ? ", " + u.state : ""}<br>` +
    `Cost ${u.cost} &middot; Def ${u.stats.def} &middot; Move ${u.stats.move} &middot; Range ${range} &middot; ` +
    `Attack ${u.stats.attack}<br>` +
    `<em>${TYPE_DESCRIPTIONS[u.type] || ""}</em>`;
  card.hidden = false;
}

function hideUnitCard() {
  document.getElementById("unit-card").hidden = true;
}

// ---- order building ------------------------------------------------------

function currentPath() {
  return document.getElementById("save-select").value;
}

function unitHasOrder(side, unitId) {
  return state.orders[side].some((o) => o.unit === unitId);
}

function renderAvailableUnits() {
  const list = document.getElementById("available-units");
  list.innerHTML = "";
  if (!state.game) return;
  const side = state.activeSide;

  const onBoard = state.game.units.filter((u) => u.side === side && !unitHasOrder(side, u.id));
  const reserves = (state.game.reserves[side] || []).filter((id) => !unitHasOrder(side, id));

  const addButton = (id) => {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.textContent = id;
    if (state.selection === id) btn.classList.add("selected");
    btn.addEventListener("click", () => selectUnit(id));
    li.appendChild(btn);
    list.appendChild(li);
  };
  onBoard.forEach((u) => addButton(u.id));
  reserves.forEach((id) => addButton(id));
}

async function selectUnit(unitId) {
  const path = currentPath();
  if (!path) return;
  try {
    const res = await fetch(`/api/options?path=${encodeURIComponent(path)}&side=${state.activeSide}&unit=${encodeURIComponent(unitId)}`);
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    state.selection = unitId;
    state.options = body;
    state.pendingOrder = null;
    // Order-building always targets the live state: jump back to it if a
    // past turn was being reviewed, so the highlighted hexes/enemies match
    // what's actually drawn.
    state.historyIndex = -1;
    state.stepIndex = -1;
    renderStep(); // redraws the (now live) map, highlights, and ghosts.
    renderAvailableUnits();
    renderBuilder();
    showUnitCard(unitId);
  } catch (err) {
    document.getElementById("turn-status").textContent = String(err.message || err);
  }
}

function clearSelection() {
  state.selection = null;
  state.options = null;
  state.pendingOrder = null;
  renderAvailableUnits();
  clearHighlights();
  renderBuilder();
  hideUnitCard();
}

// ATTACK_TYPES are the order types that need a predicted-result
// confirmation (docs/dev-plan.md section 7.5, interface item 2) rather
// than a facing pick: they always target a unit, and the engine, not the
// player, decides the resulting facing.
const ATTACK_TYPES = ["Close and Attack", "Fire", "Close and Fire"];

function renderBuilder() {
  const builder = document.getElementById("order-builder");
  const label = document.getElementById("builder-label");
  const typeChooser = document.getElementById("type-chooser");
  const stateButtons = document.getElementById("state-buttons");
  const facingPicker = document.getElementById("facing-picker");
  const predictPanel = document.getElementById("predict-panel");

  if (!state.selection) {
    builder.hidden = true;
    return;
  }
  builder.hidden = false;
  typeChooser.hidden = true;
  typeChooser.innerHTML = "";

  if (state.pendingOrder && ATTACK_TYPES.includes(state.pendingOrder.type)) {
    label.textContent = `${state.pendingOrder.unit}: ${state.pendingOrder.type} ${state.pendingOrder.targetUnit}`;
    stateButtons.hidden = true;
    facingPicker.hidden = true;
    predictPanel.hidden = false;
    return;
  }
  predictPanel.hidden = true;

  if (state.pendingOrder) {
    label.textContent = `${state.pendingOrder.unit}: ${state.pendingOrder.type}${state.pendingOrder.targetHex ? " to " + hexName(state.pendingOrder.targetHex.col, state.pendingOrder.targetHex.row) : ""} — pick a facing (or Auto)`;
    stateButtons.hidden = true;
    facingPicker.hidden = false;
    return;
  }

  facingPicker.hidden = true;
  const opts = state.options;
  label.textContent = opts.isReserve
    ? `${state.selection}: click a highlighted hex to Deploy`
    : `${state.selection}: click a highlighted hex to Move, an outlined enemy to attack, or a state button`;
  if (opts.hasBarrage) {
    label.textContent += " (red hexes are also its passive Barrage zone: it auto-fires there every turn while Ready)";
  }
  if (opts.closeMove > 0) {
    label.textContent += ` — Close: ${opts.closeMove} hexes (purple dots)`;
  }

  const buttons = [];
  if (opts.canReady) buttons.push("Ready");
  if (opts.canMobilise) buttons.push("Mobilise");
  stateButtons.innerHTML = "";
  if (buttons.length === 0) {
    stateButtons.hidden = true;
  } else {
    stateButtons.hidden = false;
    buttons.forEach((type) => {
      const b = document.createElement("button");
      b.textContent = type;
      b.addEventListener("click", () => startFacingPick({ type, unit: state.selection }));
      stateButtons.appendChild(b);
    });
  }
}

function startFacingPick(partial) {
  state.pendingOrder = partial;
  renderBuilder();
}

function onHexClick(col, row) {
  if (!state.selection || !state.options) return;
  const opts = state.options;
  if (opts.isReserve) {
    if (!(opts.deployHexes || []).some((h) => h.col === col && h.row === row)) return;
    startFacingPick({ type: "Deploy", unit: state.selection, targetHex: { col, row } });
    return;
  }
  // Any hex is a valid Move objective (docs/dev-plan.md section 7.6):
  // the path is only worked out at execution, since other units will
  // have moved by then.
  startFacingPick({ type: "Move", unit: state.selection, targetHex: { col, row } });
}

function onUnitClick(unitId) {
  const unit = state.game && state.game.units.find((u) => u.id === unitId);
  // Clicking a unit on your own side selects it for ordering (docs/dev-plan.md
  // section 7.3, interface item 2), whether or not something else was
  // already selected; clicking an already-ordered unit does nothing.
  if (unit && unit.side === state.activeSide) {
    if (!unitHasOrder(state.activeSide, unitId)) selectUnit(unitId);
    return;
  }

  if (!state.selection || !state.options) return;
  const opts = state.options;
  const types = [];
  if ((opts.meleeTargets || []).includes(unitId)) types.push("Close and Attack");
  if ((opts.fireTargets || []).includes(unitId)) types.push("Fire");
  if ((opts.closeFireTargets || []).includes(unitId)) types.push("Close and Fire");
  if (types.length === 0) return;
  if (types.length === 1) {
    startAttackConfirm({ type: types[0], unit: state.selection, targetUnit: unitId });
    return;
  }
  const chooser = document.getElementById("type-chooser");
  chooser.innerHTML = "";
  chooser.hidden = false;
  types.forEach((type) => {
    const b = document.createElement("button");
    b.textContent = type;
    b.addEventListener("click", () => startAttackConfirm({ type, unit: state.selection, targetUnit: unitId }));
    chooser.appendChild(b);
  });
}

// startAttackConfirm shows a predicted result (docs/dev-plan.md section
// 7.5, interface item 2) before an attack or Fire order is added to the
// order list: the engine's own resolution against the current board,
// labelled as a prediction since the target may move first. The order
// isn't added until the player clicks Confirm.
async function startAttackConfirm(partial) {
  state.pendingOrder = partial;
  renderBuilder();
  const predictText = document.getElementById("predict-text");
  predictText.textContent = "Predicting…";
  document.getElementById("confirm-order").disabled = true;

  const path = currentPath();
  try {
    const url =
      `/api/predict?path=${encodeURIComponent(path)}&side=${state.activeSide}` +
      `&unit=${encodeURIComponent(partial.unit)}&type=${encodeURIComponent(partial.type)}&target=${encodeURIComponent(partial.targetUnit)}`;
    const res = await fetch(url);
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    if (state.pendingOrder !== partial) return; // superseded by a later click.

    predictText.innerHTML = "";
    const labelEl = document.createElement("span");
    labelEl.className = "predict-label";
    labelEl.textContent = "Prediction (if the target stays put — it may move first):";
    predictText.appendChild(labelEl);
    const lines = body.melee ? describeMelee(body.melee) : body.ranged ? describeRanged(body.ranged) : ["Won't make contact this turn."];
    lines.forEach((line) => {
      const div = document.createElement("div");
      div.textContent = line;
      predictText.appendChild(div);
    });
  } catch (err) {
    predictText.textContent = String(err.message || err);
  } finally {
    if (state.pendingOrder === partial) {
      document.getElementById("confirm-order").disabled = false;
    }
  }
}

document.getElementById("confirm-order").addEventListener("click", () => {
  if (state.pendingOrder) commitOrder(state.pendingOrder);
});

document.getElementById("facing-picker").addEventListener("click", (e) => {
  const btn = e.target.closest("button[data-facing]");
  if (!btn || !state.pendingOrder) return;
  const facing = btn.getAttribute("data-facing");
  const order = { ...state.pendingOrder };
  if (facing) order.facing = facing;
  commitOrder(order);
});

document.getElementById("cancel-order").addEventListener("click", clearSelection);

function commitOrder(order) {
  state.orders[state.activeSide].push(order);
  clearSelection();
  renderOrderLists();
}

function orderLine(o) {
  let detail = "";
  if (o.targetHex) detail = ` | ${hexName(o.targetHex.col, o.targetHex.row)}`;
  else if (o.targetUnit) detail = ` | ${o.targetUnit}`;
  const facing = o.facing ? ` | facing ${o.facing}` : "";
  return `${o.unit} | ${o.type}${detail}${facing}`;
}

function renderOrderLists() {
  for (const side of ["north", "south"]) {
    const list = document.getElementById(`order-list-${side}`);
    list.innerHTML = "";
    state.orders[side].forEach((o, i) => {
      const li = document.createElement("li");
      const text = document.createElement("span");
      text.textContent = orderLine(o);
      li.appendChild(text);

      const up = document.createElement("button");
      up.textContent = "↑";
      up.disabled = i === 0;
      up.addEventListener("click", () => moveOrder(side, i, -1));
      li.appendChild(up);

      const down = document.createElement("button");
      down.textContent = "↓";
      down.disabled = i === state.orders[side].length - 1;
      down.addEventListener("click", () => moveOrder(side, i, 1));
      li.appendChild(down);

      const del = document.createElement("button");
      del.textContent = "✕";
      del.addEventListener("click", () => {
        state.orders[side].splice(i, 1);
        renderOrderLists();
        renderAvailableUnits();
      });
      li.appendChild(del);

      list.appendChild(li);
    });
    document.getElementById(`order-text-${side}`).textContent = state.orders[side].map(orderLine).join("\n");
  }
  renderOrderGhosts();
}

// toCube/hexDistance mirror internal/hex's offset-to-cube conversion and
// distance formula (docs/dev-plan.md section 5), purely to position the
// pending-Move ghost below — not a rule decision, the same category as
// the hexCenter/hexVertices geometry already computed client-side.
function toCube(h) {
  const q = h.col;
  const r = h.row - (h.col - (h.col & 1)) / 2;
  return { q, r, s: -q - r };
}

function hexDistance(a, b) {
  const ca = toCube(a);
  const cb = toCube(b);
  return (Math.abs(ca.q - cb.q) + Math.abs(ca.r - cb.r) + Math.abs(ca.s - cb.s)) / 2;
}

// renderOrderGhosts shows pending orders on the map while planning
// (docs/dev-plan.md section 7.3, interface item 1): a faint token at a
// Deploy order's destination, or a dashed line to the target for an
// attack or Fire order. A Move order's objective can be anywhere on the
// map (docs/dev-plan.md section 7.6), so its ghost token instead marks a
// straight-line *estimate* of where the unit would stop this turn (full
// Move, ignoring every other unit — the real path is only worked out at
// execution), labelled "≈ here" whenever that's short of the actual
// objective.
function renderOrderGhosts() {
  const svg = document.getElementById("map");
  const old = document.getElementById("ghost-layer");
  if (old) old.remove();
  if (!state.game) return;
  const layer = svgEl("g", { id: "ghost-layer" });
  svg.appendChild(layer);

  for (const side of ["north", "south"]) {
    for (const o of state.orders[side]) {
      const unit = state.game.units.find((u) => u.id === o.unit);
      const startPos = unit ? hexCenter(unit.col, unit.row) : null;

      if (o.targetHex) {
        const target = hexCenter(o.targetHex.col, o.targetHex.row);
        let gx = target.x;
        let gy = target.y;
        let isEstimate = false;
        if (o.type === "Move" && unit && startPos) {
          const dist = hexDistance({ col: unit.col, row: unit.row }, o.targetHex);
          const moveStat = unit.stats.move;
          if (dist > moveStat && dist > 0) {
            const ratio = moveStat / dist;
            gx = startPos.x + (target.x - startPos.x) * ratio;
            gy = startPos.y + (target.y - startPos.y) * ratio;
            isEstimate = true;
          }
        }
        if (startPos) {
          layer.appendChild(svgEl("line", { x1: startPos.x, y1: startPos.y, x2: target.x, y2: target.y, class: "ghost-path" }));
        }
        const pts = hexVertices(gx, gy, HEX_SIZE * 0.55);
        layer.appendChild(
          svgEl("polygon", { points: pts.map((p) => p.join(",")).join(" "), class: `ghost-token ghost-${side}` })
        );
        if (isEstimate) {
          const label = svgEl("text", { x: gx, y: gy - HEX_SIZE * 0.75, "text-anchor": "middle", class: "ghost-estimate-label" });
          label.textContent = "≈ here";
          layer.appendChild(label);
        }
      } else if (o.targetUnit) {
        const target = state.game.units.find((u) => u.id === o.targetUnit);
        if (startPos && target) {
          const end = hexCenter(target.col, target.row);
          layer.appendChild(svgEl("line", { x1: startPos.x, y1: startPos.y, x2: end.x, y2: end.y, class: "ghost-arrow" }));
        }
      }
    }
  }
}

function moveOrder(side, index, delta) {
  const arr = state.orders[side];
  const j = index + delta;
  if (j < 0 || j >= arr.length) return;
  [arr[index], arr[j]] = [arr[j], arr[index]];
  renderOrderLists();
}

document.getElementById("side-toggle").addEventListener("click", (e) => {
  const btn = e.target.closest("button[data-side]");
  if (!btn) return;
  state.activeSide = btn.getAttribute("data-side");
  document.querySelectorAll(".side-btn").forEach((b) => b.classList.toggle("active", b === btn));
  clearSelection();
});

// ---- turn history and step playback --------------------------------------
//
// Each played turn is kept client-side as {turn, boardBefore, events}:
// boardBefore is a snapshot of the live board taken right before the turn
// ran, and events[i].board (from the server) is the board exactly as it
// stood right after that event. "Viewing" a turn means historyIndex points
// at one of these; historyIndex === -1 means "live" (state.game, the
// current/actual state, used for order-building) rather than history.

function currentTurnEntry() {
  if (state.historyIndex < 0 || state.historyIndex >= state.turnHistory.length) return null;
  return state.turnHistory[state.historyIndex];
}

function currentEvents() {
  const entry = currentTurnEntry();
  return entry ? entry.events : [];
}

function currentDisplayBoard() {
  const entry = currentTurnEntry();
  if (!entry) return state.game;
  if (state.stepIndex >= 0 && state.stepIndex < entry.events.length) {
    return entry.events[state.stepIndex].board;
  }
  return entry.boardBefore;
}

function highlightUnit(unitId) {
  document.querySelectorAll("#map .unit").forEach((g) => g.classList.remove("selected"));
  if (!unitId) return;
  const g = document.querySelector(`#map .unit[data-unit-id="${CSS.escape(unitId)}"]`);
  if (g) {
    g.classList.add("selected");
    const poly = g.querySelector(".token-body");
    if (poly) poly.setAttribute("stroke-width", "4");
  }
}

function renderEventList() {
  const list = document.getElementById("event-list");
  list.innerHTML = "";
  currentEvents().forEach((e, i) => {
    const li = document.createElement("li");
    li.textContent = e.summary;
    if (i === state.stepIndex) li.classList.add("current");
    list.appendChild(li);
  });
}

// describeMelee turns a MeleeView into plain-language breakdown lines
// (docs/dev-plan.md section 7.5: combat is one comparison, and the
// margin is the result). Every number comes from the server; this only
// formats them.
function describeMelee(m) {
  const lines = [`${m.attackerId} attacks ${m.defenderId}'s ${m.edge}${m.ambushed ? " (ambush)" : ""}`];

  let att = `${m.attackerBase}`;
  if (m.attackerSupport) att += ` +${m.attackerSupport} support`;
  if (m.positionBonus) att += ` +${m.positionBonus} ${m.edge}`;
  let def = `${m.defenderBase}`;
  if (m.defenderSupport) def += ` +${m.defenderSupport} support`;
  if (m.ambushPenalty) def += ` -${m.ambushPenalty} ambush`;
  lines.push(`${att} = ${m.attackerTotal}  vs  ${def} = ${m.defenderTotal}  →  margin ${m.margin}`);

  lines.push(meleeOutcomeLine(m));
  return lines;
}

function meleeOutcomeLine(m) {
  if (m.loserDestroyed) return `${m.loserId}: destroyed`;
  if (m.hits === 0) return `${m.loserId} (the attacker): repelled${knockbackSuffix(m)}`;
  return `${m.loserId}: hit${knockbackSuffix(m)}`;
}

function knockbackSuffix(m) {
  if (!m.knockbackTo) return "";
  const dest = hexName(m.knockbackTo.col, m.knockbackTo.row);
  return m.knockbackBlocked ? ` (couldn't retreat to ${dest}: destroyed)` : `, knocked back to ${dest}`;
}

// describeRanged turns a RangedView into the same kind of breakdown for a
// Fire/Close and Fire/Barrage shot.
function describeRanged(r) {
  const lines = [`${r.shooterId} fires at ${r.targetId}`];
  if (!r.inRange || !r.inArc) {
    lines.push(`Out of range or arc (in range: ${r.inRange}, in arc: ${r.inArc}) → no effect.`);
    return lines;
  }
  let shooter = `${r.shooterBase}`;
  if (r.shooterSupport) shooter += ` +${r.shooterSupport} support`;
  let target = `${r.targetBase}`;
  if (r.targetSupport) target += ` +${r.targetSupport} support`;
  lines.push(`${shooter} = ${r.shooterTotal}  vs  ${target} = ${r.targetTotal}  →  margin ${r.margin}`);
  lines.push(r.hits === 0 ? "miss" : r.destroyed ? `${r.targetId}: destroyed` : `${r.targetId}: hit`);
  return lines;
}

// describePath turns an event's Path (the hexes its unit actually
// entered, in order) into a readable line, so a Close order's move leg
// shows the route taken rather than just the hex it stopped at.
function describePath(e) {
  if (!e.path || e.path.length === 0) return [];
  return [`Path: ${e.path.map((h) => hexName(h.col, h.row)).join(" → ")}`];
}

function renderStepDetail(e) {
  const detail = document.getElementById("step-detail");
  detail.innerHTML = "";
  if (!e) return;
  const lines = [
    ...describePath(e),
    ...(e.melee ? describeMelee(e.melee) : e.ranged ? describeRanged(e.ranged) : []),
  ];
  lines.forEach((line) => {
    const div = document.createElement("div");
    div.textContent = line;
    detail.appendChild(div);
  });
}

// priorBoardForStep returns the board exactly as it stood right before
// the currently-viewed step, for finding where a "moved" event's unit
// started from (events only snapshot the board right after themselves).
function priorBoardForStep() {
  const entry = currentTurnEntry();
  if (!entry) return null;
  if (state.stepIndex <= 0) return entry.boardBefore;
  return entry.events[state.stepIndex - 1].board;
}

// renderPathOverlay draws the route a "moved" event's unit actually took
// (docs/dev-plan.md section 7.3, latest interface pass), as a solid line
// from its pre-event position through every hex in event.path.
function renderPathOverlay(e) {
  const svg = document.getElementById("map");
  const old = document.getElementById("path-layer");
  if (old) old.remove();
  if (!e || !e.path || e.path.length === 0) return;
  const prior = priorBoardForStep();
  const startUnit = prior && prior.units.find((u) => u.id === e.unit);
  if (!startUnit) return;

  const layer = svgEl("g", { id: "path-layer" });
  svg.appendChild(layer);
  const points = [hexCenter(startUnit.col, startUnit.row), ...e.path.map((h) => hexCenter(h.col, h.row))];
  for (let i = 0; i < points.length - 1; i++) {
    layer.appendChild(
      svgEl("line", { x1: points[i].x, y1: points[i].y, x2: points[i + 1].x, y2: points[i + 1].y, class: "taken-path" })
    );
  }
  for (let i = 1; i < points.length - 1; i++) {
    layer.appendChild(svgEl("circle", { cx: points[i].x, cy: points[i].y, r: 3, class: "taken-path-dot" }));
  }
}

// renderTurnNav reflects whether a past turn is being reviewed or the
// viewer is live (building the next turn's orders), and enables/disables
// the "« Turn"/"Turn »" buttons accordingly.
function renderTurnNav() {
  const label = document.getElementById("turn-nav-label");
  const entry = currentTurnEntry();
  if (entry) {
    label.textContent = `Viewing turn ${entry.turn} (${state.historyIndex + 1} / ${state.turnHistory.length})`;
  } else {
    label.textContent = state.turnHistory.length ? "Live (building next turn)" : "No turns played yet";
  }
  document.getElementById("turn-nav-prev").disabled = state.historyIndex === 0 || (state.historyIndex === -1 && state.turnHistory.length === 0);
  document.getElementById("turn-nav-next").disabled = state.historyIndex === -1;
}

function renderStep() {
  const events = currentEvents();
  const counter = document.getElementById("step-counter");
  counter.textContent = `${events.length ? state.stepIndex + 1 : 0} / ${events.length}`;

  renderMap(currentDisplayBoard());
  if (!currentTurnEntry()) {
    // Live: re-apply order-building highlights/ghosts on top of the map
    // that renderMap() just redrew from scratch.
    renderHighlights();
    renderOrderGhosts();
  }

  const summary = document.getElementById("step-summary");
  if (state.stepIndex >= 0 && state.stepIndex < events.length) {
    const e = events[state.stepIndex];
    summary.textContent = e.summary;
    renderStepDetail(e);
    renderPathOverlay(e);
    highlightUnit(e.unit);
  } else {
    summary.textContent = currentTurnEntry() ? "(start of turn)" : "";
    renderStepDetail(null);
    renderPathOverlay(null);
    highlightUnit(null);
  }
  renderEventList();
  const current = document.querySelector("#event-list li.current");
  if (current) current.scrollIntoView({ block: "nearest" });
  renderTurnNav();
}

function turnNavPrev() {
  if (state.historyIndex === -1) {
    if (state.turnHistory.length === 0) return;
    state.historyIndex = state.turnHistory.length - 1;
  } else if (state.historyIndex > 0) {
    state.historyIndex--;
  } else {
    return;
  }
  state.stepIndex = -1;
  renderStep();
}

function turnNavNext() {
  if (state.historyIndex === -1) return; // already live.
  state.historyIndex = state.historyIndex < state.turnHistory.length - 1 ? state.historyIndex + 1 : -1;
  state.stepIndex = -1;
  renderStep();
}

// ---- API calls --------------------------------------------------------

function resetOrders() {
  state.orders = { north: [], south: [] };
  clearSelection();
  renderOrderLists();
}

async function loadGame(path) {
  const status = document.getElementById("load-status");
  status.textContent = "";
  try {
    const res = await fetch(`/api/game?path=${encodeURIComponent(path)}`);
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    state.game = body;
    state.turnHistory = [];
    state.historyIndex = -1;
    state.stepIndex = -1;
    resetOrders();
    renderGameInfo(state.game);
    renderAvailableUnits();
    renderStep();
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

async function runTurn(path) {
  const status = document.getElementById("turn-status");
  status.textContent = "";
  const turnNumber = state.game.turn;
  const boardBeforeTurn = state.game; // live board, right before this turn runs.
  const north = state.orders.north.map(orderLine).join("\n");
  const south = state.orders.south.map(orderLine).join("\n");
  try {
    const res = await fetch("/api/turn", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, north, south }),
    });
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    state.game = body.game;
    state.turnHistory.push({ turn: turnNumber, boardBefore: boardBeforeTurn, events: body.events || [] });
    state.historyIndex = state.turnHistory.length - 1;
    state.stepIndex = -1;
    resetOrders();
    renderGameInfo(state.game);
    renderAvailableUnits();
    if (body.outcome && body.outcome.over) {
      const who = body.outcome.winner ? `${body.outcome.winner} wins` : "draw";
      status.textContent = `Game over: ${who} (${body.outcome.reason})`;
    }
    renderStep();
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

// ---- save management (new game, load, delete) ---------------------------

// loadSaveList fetches the save list and (re)populates the dropdown,
// trying to keep whichever path was selected (or selectPath, if given)
// selected afterward. Returns the list.
async function loadSaveList(selectPath) {
  const status = document.getElementById("load-status");
  try {
    const res = await fetch("/api/saves");
    const saves = await res.json();
    if (!res.ok) throw new Error((saves && saves.error) || res.statusText);
    renderSaveList(saves || [], selectPath);
    return saves || [];
  } catch (err) {
    status.textContent = String(err.message || err);
    return [];
  }
}

function renderSaveList(saves, selectPath) {
  const select = document.getElementById("save-select");
  const previous = selectPath || select.value;
  select.innerHTML = "";
  if (saves.length === 0) {
    const opt = document.createElement("option");
    opt.textContent = "(no saves yet — create one below)";
    opt.disabled = true;
    select.appendChild(opt);
    return;
  }
  saves.forEach((s) => {
    const opt = document.createElement("option");
    opt.value = s.path;
    opt.textContent = `${s.path} — ${s.scenarioId}, turn ${s.turn}, ${s.units} unit${s.units === 1 ? "" : "s"}`;
    select.appendChild(opt);
  });
  if (previous && saves.some((s) => s.path === previous)) {
    select.value = previous;
  }
}

async function createNewGame() {
  const status = document.getElementById("load-status");
  status.textContent = "";
  const input = document.getElementById("new-game-name");
  let name = input.value.trim();
  if (!name) {
    status.textContent = "Enter a name for the new game.";
    return;
  }
  if (!name.toLowerCase().endsWith(".json")) name += ".json";
  try {
    const res = await fetch("/api/saves", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: name }),
    });
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || res.statusText);
    input.value = "";
    await loadSaveList(name);
    await loadGame(name);
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

// clearGame resets the viewer to its empty, nothing-loaded state (used
// after deleting the save that's currently open).
function clearGame() {
  state.game = null;
  state.turnHistory = [];
  state.historyIndex = -1;
  state.stepIndex = -1;
  state.orders = { north: [], south: [] };
  clearSelection();
  document.getElementById("map").innerHTML = "";
  document.getElementById("game-info").textContent = "";
  document.getElementById("event-list").innerHTML = "";
  document.getElementById("step-summary").textContent = "";
  document.getElementById("step-detail").innerHTML = "";
  document.getElementById("step-counter").textContent = "0 / 0";
  renderAvailableUnits();
  renderOrderLists();
  renderTurnNav();
}

// Delete needs a second click within a few seconds to actually happen
// (docs/dev-plan.md web TODO: save management), rather than a native
// confirm() dialog.
let deleteArmed = false;
function resetDeleteArm() {
  deleteArmed = false;
  document.getElementById("delete-save").textContent = "Delete";
}

async function deleteCurrentSave() {
  const path = currentPath();
  if (!path) return;
  const btn = document.getElementById("delete-save");
  if (!deleteArmed) {
    deleteArmed = true;
    btn.textContent = `Confirm delete ${path}?`;
    setTimeout(() => {
      if (deleteArmed) resetDeleteArm();
    }, 4000);
    return;
  }
  resetDeleteArm();

  const status = document.getElementById("load-status");
  status.textContent = "";
  try {
    const res = await fetch(`/api/saves?path=${encodeURIComponent(path)}`, { method: "DELETE" });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || res.statusText);
    }
    if (state.game) clearGame();
    await loadSaveList();
  } catch (err) {
    status.textContent = String(err.message || err);
  }
}

document.getElementById("new-game").addEventListener("click", createNewGame);
document.getElementById("delete-save").addEventListener("click", deleteCurrentSave);
document.getElementById("save-select").addEventListener("change", resetDeleteArm);

document.getElementById("load").addEventListener("click", () => {
  loadGame(currentPath());
});

document.getElementById("run-turn").addEventListener("click", () => {
  runTurn(currentPath());
});

// On page load: populate the save list, and if there's exactly one save
// (the common case — one game in progress), load it straight away so
// opening the page just works.
(async function initSaves() {
  const saves = await loadSaveList();
  if (saves.length === 1) {
    await loadGame(saves[0].path);
  }
})();

document.getElementById("step-prev").addEventListener("click", () => {
  if (state.stepIndex > -1) {
    state.stepIndex--;
    renderStep();
  }
});

document.getElementById("step-next").addEventListener("click", () => {
  if (state.stepIndex < currentEvents().length - 1) {
    state.stepIndex++;
    renderStep();
  }
});

document.getElementById("turn-nav-prev").addEventListener("click", turnNavPrev);
document.getElementById("turn-nav-next").addEventListener("click", turnNavNext);
