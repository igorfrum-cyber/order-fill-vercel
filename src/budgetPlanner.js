import { procurementTotalCents, christinaLineGroups, completeChristinaLines } from './christinaLines.js';

export const NORMS = { C: 2, B: 2.5, A: 3, 'A+': 3.5 };
const categories = Object.keys(NORMS);
export function discountValue(value) {
  if (!String(value).trim()) throw new Error('Введите скидку числом.');
  const n = Number(String(value).trim().replace(/%$/, '').replace(',', '.'));
  if (!Number.isFinite(n) || n < 0 || n >= 100) throw new Error('Скидка должна быть от 0 до 99,99%.');
  return n;
}
export function coverage(row, quantity) {
  return row.demand > 0 ? (row.stock + row.transit - (row.outbound || 0) + quantity * row.unit) / row.demand : null;
}
function nextQuantity(row, q, direction) {
  if (direction > 0) return Math.max(row.minimum, (Math.floor(q / row.step + 1e-8) + 1) * row.step);
  const next = (Math.ceil(q / row.step - 1e-8) - 1) * row.step;
  return next < row.minimum ? 0 : Math.max(0, next);
}

/**
 * Pure preview: never mutates the input or changes locked quantities.
 * Row fields: key, name, category, quantity, price, demand, delivery,
 * stock, transit, outbound, unit (pieces/package), step, minimum, locked/excluded.
 * Quantities are supplier units; demand, stock and outbound are pieces.
 * Permissions are explicit per-preview opt-ins, never inferred from the target.
 */
function planBudgetCore(input, target, options = {}) {
  const rows = input.map(r => ({ ...r, before: Number(r.quantity || 0), quantity: Number(r.quantity || 0) }));
  const cents = n => Math.round(n * 100);
  if (!Number.isFinite(target) || target < 0) throw new Error('Введите неотрицательную сумму.');
  for (const r of rows) {
    if (!(r.price > 0) && r.quantity > 0) throw new Error(`Не найдена закупочная цена: ${r.name}`);
    if (![r.quantity, r.stock, r.transit, r.unit, r.step, r.minimum].every(Number.isFinite) || r.quantity < 0 || r.unit <= 0 || r.step <= 0) throw new Error(`Некорректные данные: ${r.name}`);
  }
  const total = () => procurementTotalCents(rows);
  // Changing the minimum of a PROFF line changes the discount on other rows.
  // Evaluate the complete order; a fixed unit-price delta is incorrect here.
  const changeCost = (r, q) => {
    if (r.group !== 'proff' || !r.line) return cents(r.price*q) - cents(r.price*r.quantity);
    const before = r.quantity, oldTotal = total();
    r.quantity = q;
    const cost = total() - oldTotal;
    r.quantity = before;
    return cost;
  };
  const start = total(), goal = cents(target), up = goal > start;
  let amount = start, reason = '', iterations = 0;
  const eligible = r => !r.locked && !r.excluded && !r.unsafe && r.price > 0 && r.demand > 0 && NORMS[r.category];
  while (up ? amount < goal : amount > goal) {
    if (++iterations > 250000) { reason = 'Достигнут предел вычислений. Уточните сумму.'; break; }
    let candidates = rows.filter(eligible).map(r => {
      const q = nextQuantity(r, r.quantity, up ? 1 : -1);
      return { r, q, months: coverage(r, r.quantity), nextMonths: coverage(r, q), cost: changeCost(r, q) };
    }).filter(c => c.q !== c.r.quantity);
    if (up) {
      // Enforce the absolute cap before category priorities. A pack crossing it
      // is not rounded down or split; seek another item, or ask permission.
      const belowCap = candidates.filter(c => c.nextMonths <= 6 + 1e-8);
      if (belowCap.length) candidates = belowCap;
      else if (!options.allowOverSix && candidates.length) { reason = 'overSix'; break; }
      candidates = candidates.filter(c => amount + c.cost <= Math.floor(goal * 1.05 + 1e-8));
      // A supplier pack must fit the category stage after purchase. Defer packs
      // that jump past it until every feasible stage has been considered.
      const stage = categories.find(cat => candidates.some(c => c.r.category === cat && c.months < NORMS[cat] + c.r.delivery - 1e-8 && c.nextMonths <= NORMS[cat] + c.r.delivery + 1e-8));
      if (stage) candidates = candidates.filter(c => c.r.category === stage && c.months < NORMS[stage] + c.r.delivery - 1e-8 && c.nextMonths <= NORMS[stage] + c.r.delivery + 1e-8);
      candidates.sort((a,b) => (stage ? a.months : a.nextMonths) / (NORMS[a.r.category] + a.r.delivery) - (stage ? b.months : b.nextMonths) / (NORMS[b.r.category] + b.r.delivery) || Math.abs(goal - amount - a.cost) - Math.abs(goal - amount - b.cost));
    } else {
      // Protect one month plus delivery before considering category cuts.
      const protectedRows = candidates.filter(c => c.nextMonths >= 1 + c.r.delivery - 1e-8);
      if (protectedRows.length) candidates = protectedRows;
      else if (!options.allowBelowOne && candidates.length) { reason = 'belowOne'; break; }
      else {
        const cat = categories.find(k => candidates.some(c => c.r.category === k));
        candidates = candidates.filter(c => c.r.category === cat);
      }
      candidates.sort((a,b) => b.months - a.months || Math.abs(goal - amount - a.cost) - Math.abs(goal - amount - b.cost));
    }
    if (!candidates.length) { reason = 'Нет допустимых изменений для достижения суммы с учетом цен, закреплений и партий.'; break; }
    const chosen = candidates[0];
    chosen.r.quantity = chosen.q;
    amount += chosen.cost;
  }
  // After a downward crossing, restore affordable supplier steps without exceeding the target.
  if (!up && amount < goal && !reason) {
    for (const r of [...rows].sort((a,b) => categories.indexOf(b.category)-categories.indexOf(a.category))) {
      while (r.quantity < r.before) {
        const q = nextQuantity(r,r.quantity,1), cost = changeCost(r,q);
        if (q > r.before || amount + cost > goal) break;
        r.quantity=q; amount+=cost;
      }
    }
  }
  return { rows, before: start/100, total: amount/100, target, reason, complete: !reason && (up ? amount >= goal && amount <= goal*1.05 : amount <= goal) };
}

/** PROFF line completion is an atomic pre-pass, followed by normal coverage. */
export function planBudget(input, target, options = {}) {
  const original = input.map(r => ({ ...r, quantity: Number(r.quantity || 0) }));
  const before = procurementTotalCents(original) / 100;
  if (!original.some(r => r.group === 'proff' && r.line) || target < before) return planBudgetCore(input, target, options);
  // Validate the same row contract even when the line pass reaches the target.
  planBudgetCore(original, before, options);
  const baselines = new Map(christinaLineGroups(original).map(l => [l.id, l.netCents]));
  const starting = new Map(original.map(r => [r.key, r.quantity]));
  let rows = original, result;
  const steps = [];
  // A final first-set completion can reduce the total. Resume normal top-up
  // if necessary; a completed line cannot receive that exception a second time.
  for (let pass = 0; pass <= baselines.size + 1; pass++) {
    const lines = completeChristinaLines(rows, target, baselines);
    rows = lines.rows; steps.push(...lines.steps);
    const total = procurementTotalCents(rows) / 100;
    if (total >= target) { result = { rows, total, target, reason: '', complete: total <= target * 1.05 }; break; }
    result = planBudgetCore(rows, target, options);
    rows = result.rows;
    if (!result.complete) break;
    const finish = completeChristinaLines(rows, target, baselines);
    rows = finish.rows; steps.push(...finish.steps);
    result = { ...result, rows, total: procurementTotalCents(rows) / 100 };
    if (result.total >= target) break;
  }
  return { ...result, before, rows: rows.map(r => ({ ...r, before: starting.get(r.key),
    lineCompletion: steps.some(s => s.id === r.line?.id) && r.quantity > starting.get(r.key) ? r.line.name : undefined })),
    complete: Boolean(result?.complete) && procurementTotalCents(rows) >= Math.round(target * 100)
      && procurementTotalCents(rows) <= Math.floor(target * 105 + 1e-8), lineSteps: steps };
}
