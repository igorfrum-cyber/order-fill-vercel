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
export function planBudget(input, target, options = {}) {
  const rows = input.map(r => ({ ...r, before: Number(r.quantity || 0), quantity: Number(r.quantity || 0) }));
  const cents = n => Math.round(n * 100);
  if (!Number.isFinite(target) || target < 0) throw new Error('Введите неотрицательную сумму.');
  for (const r of rows) {
    if (!(r.price > 0) && r.quantity > 0) throw new Error(`Не найдена закупочная цена: ${r.name}`);
    if (![r.quantity, r.stock, r.transit, r.unit, r.step, r.minimum].every(Number.isFinite) || r.quantity < 0 || r.unit <= 0 || r.step <= 0) throw new Error(`Некорректные данные: ${r.name}`);
  }
  const total = () => rows.reduce((s,r) => s + cents((r.price || 0) * r.quantity), 0);
  const start = total(), goal = cents(target), up = goal > start;
  let amount = start, reason = '', iterations = 0;
  const eligible = r => !r.locked && !r.excluded && !r.unsafe && r.price > 0 && r.demand > 0 && NORMS[r.category];
  while (up ? amount < goal : amount > goal) {
    if (++iterations > 250000) { reason = 'Достигнут предел вычислений. Уточните сумму.'; break; }
    let candidates = rows.filter(eligible).map(r => {
      const q = nextQuantity(r, r.quantity, up ? 1 : -1);
      return { r, q, months: coverage(r, r.quantity), nextMonths: coverage(r, q), cost: cents(r.price*q) - cents(r.price*r.quantity) };
    }).filter(c => c.q !== c.r.quantity);
    if (up) {
      // Enforce the absolute cap before category priorities. A pack crossing it
      // is not rounded down or split; seek another item, or ask permission.
      const belowCap = candidates.filter(c => c.nextMonths <= 6 + 1e-8);
      if (belowCap.length) candidates = belowCap;
      else if (!options.allowOverSix && candidates.length) { reason = 'overSix'; break; }
      const stage = categories.find(cat => candidates.some(c => c.r.category === cat && c.months < NORMS[cat] + c.r.delivery - 1e-8));
      if (stage) candidates = candidates.filter(c => c.r.category === stage && c.months < NORMS[stage] + c.r.delivery - 1e-8);
      candidates = candidates.filter(c => amount + c.cost <= Math.floor(goal * 1.05 + 1e-8));
      candidates.sort((a,b) => a.months / (NORMS[a.r.category] + a.r.delivery) - b.months / (NORMS[b.r.category] + b.r.delivery) || Math.abs(goal - amount - a.cost) - Math.abs(goal - amount - b.cost));
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
        const q = nextQuantity(r,r.quantity,1), cost = cents(r.price*q)-cents(r.price*r.quantity);
        if (q > r.before || amount + cost > goal) break;
        r.quantity=q; amount+=cost;
      }
    }
  }
  return { rows, before: start/100, total: amount/100, target, reason, complete: !reason && (up ? amount >= goal && amount <= goal*1.05 : amount <= goal) };
}
