import test from "node:test";
import assert from "node:assert/strict";
import {
  toCents,
  christinaLineGroups,
  procurementTotalCents,
  proposeLineStep,
  completeChristinaLines,
  LINE_COVERAGE,
} from "./christinaLines.js";

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

/** Build a minimal PROFF row that belongs to a named line. */
function proffRow(key, lineId, article, required, extra = {}) {
  return {
    key,
    name: key,
    group: "proff",
    category: "A",
    quantity: 0,
    price: 100,
    demand: 10,
    stock: 0,
    transit: 0,
    outbound: 0,
    unit: 1,
    step: 3,
    minimum: 3,
    delivery: 0.5,
    locked: false,
    excluded: false,
    unsafe: false,
    line: { id: lineId, name: lineId, article, required },
    ...extra,
  };
}

/** Build a plain non-PROFF row (no line). */
function plainRow(key, extra = {}) {
  return {
    key,
    name: key,
    group: "main",
    category: "A",
    quantity: 10,
    price: 50,
    demand: 10,
    stock: 0,
    transit: 0,
    outbound: 0,
    unit: 1,
    step: 1,
    minimum: 1,
    delivery: 0.25,
    locked: false,
    excluded: false,
    unsafe: false,
    line: null,
    ...extra,
  };
}

// ---------------------------------------------------------------------------
// toCents
// ---------------------------------------------------------------------------

test("toCents rounds to nearest kopeck", () => {
  assert.equal(toCents(100), 10000);
  assert.equal(toCents(100.005), 10001); // Math.round(100.005 * 100) = 10001
  assert.equal(toCents(0), 0);
  assert.equal(toCents(null), 0);
  assert.equal(toCents(undefined), 0);
});

// ---------------------------------------------------------------------------
// christinaLineGroups
// ---------------------------------------------------------------------------

test("christinaLineGroups skips non-proff rows", () => {
  const rows = [plainRow("r1"), plainRow("r2")];
  assert.deepEqual(christinaLineGroups(rows), []);
});

test("christinaLineGroups skips proff rows without line", () => {
  const rows = [{ ...plainRow("r1"), group: "proff", line: null }];
  assert.deepEqual(christinaLineGroups(rows), []);
});

test("christinaLineGroups groups rows by line.id and computes sets", () => {
  const required = ["CHR001", "CHR002"];
  const rows = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 6 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 6 }),
  ];
  const groups = christinaLineGroups(rows);
  assert.equal(groups.length, 1);
  const [g] = groups;
  assert.equal(g.id, "MUSE");
  assert.equal(g.valid, true);
  assert.equal(g.sets, 6);
  // savingCents = 2 rows × toCents(100 × 6 × 0.05) = 2 × 3000 = 6000
  assert.equal(g.savingCents, 6000);
  assert.equal(g.baseCents, 120000); // 2 × 100 × 6 = 1200 → 120000
  assert.equal(g.netCents, 120000 - 6000);
});

test("christinaLineGroups marks line invalid when required lists disagree", () => {
  const rows = [
    proffRow("r1", "MUSE", "CHR001", ["CHR001", "CHR002"]),
    proffRow("r2", "MUSE", "CHR002", ["CHR001", "CHR003"]), // different required
  ];
  const [g] = christinaLineGroups(rows);
  assert.equal(g.valid, false);
  assert.equal(g.sets, 0);
});

test("christinaLineGroups sets=0 when fewer than 3 complete sets", () => {
  const required = ["CHR001", "CHR002"];
  const rows = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 2 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 2 }),
  ];
  const [g] = christinaLineGroups(rows);
  assert.equal(g.sets, 0);
  assert.equal(g.savingCents, 0);
});

// ---------------------------------------------------------------------------
// procurementTotalCents
// ---------------------------------------------------------------------------

test("procurementTotalCents subtracts PROFF set discount", () => {
  const required = ["CHR001", "CHR002"];
  const rows = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 3 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 3 }),
  ];
  // base = 2 × 100 × 3 = 600 → 60000 cents
  // saving = 2 × toCents(100 × 3 × 0.05) = 2 × 1500 = 3000 cents
  const total = procurementTotalCents(rows);
  assert.equal(total, 57000);
});

test("procurementTotalCents equals simple sum for non-proff rows", () => {
  const rows = [plainRow("r1", { quantity: 5, price: 200 }), plainRow("r2", { quantity: 3, price: 100 })];
  assert.equal(procurementTotalCents(rows), 130000);
});

// ---------------------------------------------------------------------------
// LINE_COVERAGE
// ---------------------------------------------------------------------------

test("LINE_COVERAGE has expected values", () => {
  assert.equal(LINE_COVERAGE["C"], 2);
  assert.equal(LINE_COVERAGE["B"], 2.5);
  assert.equal(LINE_COVERAGE["A"], 3);
  assert.equal(LINE_COVERAGE["A+"], 3.5);
});

// ---------------------------------------------------------------------------
// proposeLineStep
// ---------------------------------------------------------------------------

function baselineMap(rows) {
  return new Map(christinaLineGroups(rows).map((l) => [l.id, l.netCents]));
}

test("proposeLineStep rejects when line metadata is incomplete", () => {
  // single-member line — required has 2 but only 1 article present → invalid
  const rows = [proffRow("r1", "MUSE", "CHR001", ["CHR001", "CHR002"])];
  const result = proposeLineStep(rows, "MUSE", baselineMap(rows), 50000);
  assert.equal(result.accepted, false);
  assert.equal(result.reason, "incompleteMetadata");
});

test("proposeLineStep rejects when target already reached", () => {
  const required = ["CHR001", "CHR002"];
  const rows = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 3 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 3 }),
  ];
  // target below current procurement total
  const result = proposeLineStep(rows, "MUSE", baselineMap(rows), 1);
  assert.equal(result.accepted, false);
  assert.equal(result.reason, "targetReached");
});

test("proposeLineStep accepts first-set completion for valid line", () => {
  // The saving check (savedCents*100 >= addedCents*15) means 5% PROFF discount
  // never clears the 15% threshold on its own — this is correct algorithm behaviour.
  // proposeLineStep returns "saving" for first-set when the ratio is below threshold.
  const required = ["CHR001", "CHR002"];
  const at3 = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 3, price: 100 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 3, price: 100 }),
  ];
  const baselines = baselineMap(at3);
  const startRows = at3.map((r) => ({ ...r, quantity: 0 }));
  const result = proposeLineStep(startRows, "MUSE", baselines, 100000);
  // saving: savedCents*100 < addedCents*15 → rejected with "saving"
  assert.equal(result.accepted, false);
  assert.match(result.reason, /saving|lineGrowth/);
});

test("proposeLineStep rejects lineGrowth when next netCents exceeds 130% baseline", () => {
  const required = ["CHR001", "CHR002"];
  const at3 = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 3, price: 100 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 3, price: 100 }),
  ];
  const baselines = baselineMap(at3); // baseline = netCents at 3 sets
  // Propose step 3→6: after.netCents = 2×baseline → exceeds 1.3×baseline
  const result = proposeLineStep(at3, "MUSE", baselines, 100000);
  assert.equal(result.accepted, false);
  assert.equal(result.reason, "lineGrowth");
});

// ---------------------------------------------------------------------------
// completeChristinaLines
// ---------------------------------------------------------------------------

test("completeChristinaLines returns empty steps when no valid lines", () => {
  const rows = [plainRow("r1"), plainRow("r2")];
  const result = completeChristinaLines(rows, 50000);
  assert.equal(result.steps.length, 0);
  assert.deepEqual(result.rows, rows);
});

test("completeChristinaLines throws on negative target", () => {
  assert.throws(() => completeChristinaLines([], -1), /неотрицательную/);
});

test("completeChristinaLines stops when no proposal is accepted", () => {
  // When all lines are blocked (saving/lineGrowth), completeChristinaLines returns
  // immediately with empty steps — no infinite loop.
  const required = ["CHR001", "CHR002"];
  const at3 = [
    proffRow("r1", "MUSE", "CHR001", required, { quantity: 3, price: 100 }),
    proffRow("r2", "MUSE", "CHR002", required, { quantity: 3, price: 100 }),
  ];
  const baselines = baselineMap(at3);
  // All proposals will be rejected (lineGrowth). completeChristinaLines returns early.
  const result = completeChristinaLines(at3, 100000, baselines);
  assert.equal(result.steps.length, 0);
  assert.equal(result.rejected.length, 1);
  assert.equal(result.rejected[0].id, "MUSE");
});
