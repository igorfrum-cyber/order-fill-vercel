/**
 * CHRISTINA PROFF procurement pricing, independent of workbook/export prices.
 * `price` is the unit price AFTER the main discount, in rubles. `line` contains
 * { id, name, article, required: string[] }. The full required list must come
 * from the blank, not just matched/recommended rows. HOME never qualifies.
 */

export const toCents = (value) => Math.round((Number(value) || 0) * 100);

export function christinaLineGroups(rows) {
  const groups = new Map();
  for (const row of rows) {
    if (row.group !== "proff" || !row.line) continue;
    const id = row.line.id;
    if (!groups.has(id)) {
      groups.set(id, { id, name: row.line.name, rows: [], required: row.line.required, valid: true });
    }
    const line = groups.get(id);
    line.rows.push(row);
    const expected = [...(line.required || [])].sort().join("\n");
    if ([...(row.line.required || [])].sort().join("\n") !== expected) line.valid = false;
  }
  for (const line of groups.values()) {
    const required = new Set(line.required);
    const articles = line.rows.map((r) => r.line.article);
    line.valid &&=
      required.size > 0 &&
      required.size === line.required.length &&
      articles.length === required.size &&
      new Set(articles).size === articles.length &&
      articles.every((a) => required.has(a)) &&
      line.rows.every((r) => !r.unsafe && r.price > 0 && Number.isFinite(r.quantity) && r.quantity >= 0);
    // Manual non-multiple quantities still contain equal complete sets. The
    // planner itself adds supplier steps of three; pricing must reflect reality.
    line.sets = line.valid ? Math.floor(Math.min(...line.rows.map((r) => r.quantity))) : 0;
    if (line.sets < 3) line.sets = 0;
    line.baseCents = line.rows.reduce((sum, r) => sum + toCents(r.price * r.quantity), 0);
    line.savingCents = line.rows.reduce((sum, r) => sum + (line.sets ? toCents(r.price * line.sets * 0.05) : 0), 0);
    line.netCents = line.baseCents - line.savingCents;
  }
  return [...groups.values()];
}

/** Round monetary line components to kopecks, never round every set separately. */
export function procurementTotalCents(rows) {
  return (
    rows.reduce((sum, r) => sum + toCents(r.price * r.quantity), 0) -
    christinaLineGroups(rows).reduce((sum, line) => sum + line.savingCents, 0)
  );
}

export const LINE_COVERAGE = { C: 2, B: 2.5, A: 3, "A+": 3.5 };

const months = (row, q) =>
  row.demand > 0 ? (row.stock + row.transit - (row.outbound || 0) + q * row.unit) / row.demand : null;

/**
 * A single atomic next-set proposal. No partial line additions on rejection.
 * baselines are fixed net costs at the start of the manager's current run.
 * Rejections are structured so the UI can explain why a line was skipped.
 */
export function proposeLineStep(rows, id, baselines, target) {
  const line = christinaLineGroups(rows).find((l) => l.id === id);
  if (!line?.valid) return { accepted: false, reason: "incompleteMetadata" };
  const nextSets = Math.max(3, (Math.floor(line.sets / 3) + 1) * 3);
  const changes = line.rows.filter((r) => r.quantity < nextSets);
  const firstSingle = line.sets === 0 && changes.length === 1 && changes[0].quantity === 0;
  const current = procurementTotalCents(rows);
  if (current >= toCents(target) && !firstSingle) return { accepted: false, reason: "targetReached" };
  if (changes.some((r) => r.locked || r.excluded)) return { accepted: false, reason: "locked" };
  if (
    !firstSingle &&
    changes.some((r) => {
      const cover = months(r, nextSets);
      return (
        cover == null ||
        !Number.isFinite(r.delivery) ||
        !LINE_COVERAGE[r.category] ||
        cover > LINE_COVERAGE[r.category] + r.delivery + 1e-8 ||
        cover > 6 + 1e-8
      );
    })
  )
    return { accepted: false, reason: "coverage" };
  const keys = new Set(changes.map((r) => r.key));
  const proposed = rows.map((r) => (keys.has(r.key) ? { ...r, quantity: nextSets } : { ...r }));
  const after = christinaLineGroups(proposed).find((l) => l.id === id);
  const addedCents = changes.reduce(
    (sum, r) => sum + toCents(r.price * nextSets) - toCents(r.price * r.quantity),
    0,
  );
  const savedCents = after.savingCents - line.savingCents;
  const baseline = baselines.get(id);
  if (!(baseline > 0) || after.netCents > Math.floor(baseline * 1.3 + 1e-8))
    return { accepted: false, reason: "lineGrowth" };
  if (addedCents <= 0 || savedCents * 100 < addedCents * 15) return { accepted: false, reason: "saving" };
  const totalCents = procurementTotalCents(proposed);
  if (totalCents > Math.floor(toCents(target) * 1.05 + 1e-8)) return { accepted: false, reason: "targetLimit" };
  return {
    accepted: true,
    rows: proposed,
    id,
    name: line.name,
    sets: after.sets,
    addedCents,
    savedCents,
    totalCents,
    firstSingle,
    rank: ["A+", "A", "B"].map((cat) => line.rows.filter((r) => r.category === cat).length),
  };
}

/** Complete eligible lines first; the caller then runs normal coverage top-up. */
export function completeChristinaLines(
  input,
  target,
  baselines = new Map(christinaLineGroups(input).map((l) => [l.id, l.netCents])),
) {
  if (!Number.isFinite(target) || target < 0) throw new Error("Введите неотрицательную сумму.");
  let rows = input.map((r) => ({ ...r }));
  const steps = [];
  for (let iteration = 0; iteration < 250000; iteration++) {
    const proposals = christinaLineGroups(rows).map((l) => ({
      id: l.id,
      ...proposeLineStep(rows, l.id, baselines, target),
    }));
    const candidates = proposals.filter((p) => p.accepted);
    if (!candidates.length) {
      return {
        rows,
        steps,
        rejected: proposals.filter((p) => !p.accepted).map(({ id, reason }) => ({ id, reason })),
        baselines,
      };
    }
    candidates.sort(
      (a, b) =>
        b.rank[0] - a.rank[0] ||
        b.rank[1] - a.rank[1] ||
        b.rank[2] - a.rank[2] ||
        b.savedCents / b.addedCents - a.savedCents / a.addedCents ||
        a.id.localeCompare(b.id),
    );
    const chosen = candidates[0];
    rows = chosen.rows;
    steps.push({
      id: chosen.id,
      name: chosen.name,
      sets: chosen.sets,
      addedCents: chosen.addedCents,
      savedCents: chosen.savedCents,
    });
  }
  throw new Error("Достигнут предел расчета комплектов. Уточните сумму.");
}
