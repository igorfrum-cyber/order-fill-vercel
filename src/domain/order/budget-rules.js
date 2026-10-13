import { brandRule } from "./../brands/rules.js";
import { novacutanMinimumQuantity, novacutanSupplierUnitSize } from "./quantities.js";

export function budgetOrderRules(brand, name, box) {
  const rule = brandRule(brand);
  if (brand === 'novacutan') {
    const unit = novacutanSupplierUnitSize(name);
    const minimum = Number(box) > 0 ? Number(box) : novacutanMinimumQuantity(name);
    const step = unit > 1 ? 1 : 10;
    return { unit, minimum: Math.ceil(minimum / step) * step, step };
  }
  const step = ['multiple', 'nearestMultiple'].includes(rule.adjustment) ? rule.multiple : rule.adjustment === 'box' ? Math.max(1, Number(box) || 1) : 1;
  return { unit: 1, step, minimum: rule.adjustment === 'minimum' ? Math.max(1, Number(box) || 1) : step };
}
