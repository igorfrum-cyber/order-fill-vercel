const NORMS = { C: 2, B: 2.5, A: 3, "A+": 3.5 };
const categories = Object.keys(NORMS);

export function discountValue(value) {
  if (!String(value).trim()) throw new Error("Введите скидку числом.");
  const n = Number(String(value).trim().replace(/%$/, "").replace(",", "."));
  if (!Number.isFinite(n) || n < 0 || n >= 100) throw new Error("Скидка должна быть от 0 до 99,99%.");
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

function ruMoney(value) {
  return Number(value).toLocaleString("ru-RU", { maximumFractionDigits: 2 });
}

export function budgetChangeComment(row) {
  if (row.before == null) return "";
  const delta = row.quantity - row.before;
  if (delta > 0) return `Добавилось ${ruMoney(delta)} ${row.unit > 1 ? "уп." : "шт."} Для закупа до суммы.`;
  if (delta < 0) return `Уменьшено на ${ruMoney(-delta)} ${row.unit > 1 ? "уп." : "шт."} Для снижения заказа до указанной суммы.`;
  return "";
}

export function appendBudgetComment(previous, note) {
  return [previous, note].filter(Boolean).join("; ");
}

/**
 * Pure preview: never mutates the input. Same logic as calculation-service PlanBudget
 * and origin/main budgetPlanner.planBudget.
 */
export function planBudget(input, target, options = {}) {
  const rows = input.map((r) => ({ ...r, before: Number(r.quantity || 0), quantity: Number(r.quantity || 0) }));
  const cents = (n) => Math.round(n * 100);
  if (!Number.isFinite(target) || target < 0) throw new Error("Введите неотрицательную сумму.");
  for (const r of rows) {
    if (!(r.price > 0) && r.quantity > 0) throw new Error(`Не найдена закупочная цена: ${r.name}`);
    if (![r.quantity, r.stock, r.transit, r.unit, r.step, r.minimum].every(Number.isFinite) || r.quantity < 0 || r.unit <= 0 || r.step <= 0) {
      throw new Error(`Некорректные данные: ${r.name}`);
    }
  }
  const total = () => rows.reduce((s, r) => s + cents((r.price || 0) * r.quantity), 0);
  const start = total();
  const goal = cents(target);
  const up = goal > start;
  let amount = start;
  let reason = "";
  let iterations = 0;
  const eligible = (r) => !r.locked && !r.excluded && !r.unsafe && r.price > 0 && r.demand > 0 && NORMS[r.category];
  while (up ? amount < goal : amount > goal) {
    if (++iterations > 250000) {
      reason = "Достигнут предел вычислений. Уточните сумму.";
      break;
    }
    let candidates = rows.filter(eligible).map((r) => {
      const q = nextQuantity(r, r.quantity, up ? 1 : -1);
      return { r, q, months: coverage(r, r.quantity), nextMonths: coverage(r, q), cost: cents(r.price * q) - cents(r.price * r.quantity) };
    }).filter((c) => c.q !== c.r.quantity);
    if (up) {
      const belowCap = candidates.filter((c) => c.nextMonths <= 6 + 1e-8);
      if (belowCap.length) candidates = belowCap;
      else if (!options.allowOverSix && candidates.length) {
        reason = "overSix";
        break;
      }
      const stage = categories.find((cat) => candidates.some((c) => c.r.category === cat && c.months < NORMS[cat] + c.r.delivery - 1e-8));
      if (stage) candidates = candidates.filter((c) => c.r.category === stage && c.months < NORMS[stage] + c.r.delivery - 1e-8);
      candidates = candidates.filter((c) => amount + c.cost <= Math.floor(goal * 1.05 + 1e-8));
      candidates.sort((a, b) => a.months / (NORMS[a.r.category] + a.r.delivery) - b.months / (NORMS[b.r.category] + b.r.delivery) || Math.abs(goal - amount - a.cost) - Math.abs(goal - amount - b.cost));
    } else {
      const protectedRows = candidates.filter((c) => c.nextMonths >= 1 + c.r.delivery - 1e-8);
      if (protectedRows.length) candidates = protectedRows;
      else if (!options.allowBelowOne && candidates.length) {
        reason = "belowOne";
        break;
      } else {
        const cat = categories.find((k) => candidates.some((c) => c.r.category === k));
        candidates = candidates.filter((c) => c.r.category === cat);
      }
      candidates.sort((a, b) => b.months - a.months || Math.abs(goal - amount - a.cost) - Math.abs(goal - amount - b.cost));
    }
    if (!candidates.length) {
      reason = "Нет допустимых изменений для достижения суммы с учетом цен, закреплений и партий.";
      break;
    }
    const chosen = candidates[0];
    chosen.r.quantity = chosen.q;
    amount += chosen.cost;
  }
  if (!up && amount < goal && !reason) {
    for (const r of [...rows].sort((a, b) => categories.indexOf(b.category) - categories.indexOf(a.category))) {
      while (r.quantity < r.before) {
        const q = nextQuantity(r, r.quantity, 1);
        const cost = cents(r.price * q) - cents(r.price * r.quantity);
        if (q > r.before || amount + cost > goal) break;
        r.quantity = q;
        amount += cost;
      }
    }
  }
  return { rows, before: start / 100, total: amount / 100, target, reason, complete: !reason && (up ? amount >= goal && amount <= goal * 1.05 : amount <= goal) };
}

const NON_BOX_BRANDS = new Set(["christina", "klapp", "novacutan", "sothys", "skin_synergy"]);

export function budgetOrderRules(brand, name, box) {
  if (brand === "novacutan") {
    const unit = novacutanSupplierUnitSize(name);
    const minimum = Number(box) > 0 ? Number(box) : novacutanMinimumQuantity(name);
    const step = unit > 1 ? 1 : 10;
    return { unit, minimum: Math.ceil(minimum / step) * step, step };
  }
  if (brand === "christina") return { unit: 1, step: 3, minimum: 3 };
  if (brand === "klapp") return { unit: 1, step: 3, minimum: 3 };
  if (!NON_BOX_BRANDS.has(brand)) {
    const step = Math.max(1, Number(box) || 1);
    return { unit: 1, step, minimum: step };
  }
  return { unit: 1, step: 1, minimum: 1 };
}

function normalizeHeader(value) {
  return String(value || "")
    .toLowerCase()
    .replaceAll("ё", "е")
    .replace(/[^\p{L}\p{N}%]+/gu, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function novacutanMatchKey(value) {
  const text = normalizeHeader(value).replace(/\bbiopro\b/g, "bio pro").replace(/\bbio\s*pro\b/g, "bio pro");
  if (!text || text.includes("термопакет") || text.includes("хладоэлемент")) return "";
  const hasFillerMask = text.includes("filler") || text.includes("филлер") || text.includes("маск") || text.includes("mask");
  if (hasFillerMask && (text.includes("глаз") || text.includes("eye"))) return "mask-eye";
  if (hasFillerMask && (text.includes("лица") || text.includes("face"))) return "mask-face";
  if (text.includes("sbio")) return "sbio";
  if (text.includes("ybio")) return "ybio";
  if (text.includes("bio pro")) return "bio-pro";
  if (text.includes("prima")) return "prima";
  if (text.includes("master")) return "master";
  if (text.includes("bright") || text.includes("брайт")) return "bright";
  if (text.includes("gentle") || text.includes("джентл") || text.includes("джентел")) return "gentle";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("light")) return "dvs-fbio-light";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("medium")) return "dvs-fbio-medium";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("volume")) return "dvs-fbio-volume";
  if (text.includes("fbio") && text.includes("light")) return "fbio-light";
  if (text.includes("fbio") && text.includes("medium")) return "fbio-medium";
  if (text.includes("fbio") && text.includes("volume")) return "fbio-volume";
  if (text.includes("eye")) return "eye";
  return "";
}

export function novacutanMinimumQuantity(name) {
  const text = normalizeHeader(name);
  if ((text.includes("mask") || text.includes("маск")) && (text.includes("filler") || text.includes("филлер"))) return 10;
  if (text.includes("fbio") || text.includes("bright") || text.includes("брайт") || text.includes("gentle") || text.includes("джентл") || text.includes("джентел")) {
    return 50;
  }
  return 100;
}

export function novacutanSupplierUnitSize(name) {
  const key = novacutanMatchKey(name);
  return key === "mask-eye" || key === "mask-face" ? 5 : 1;
}
