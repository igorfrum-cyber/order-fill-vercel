import { NORTH_CITIES } from "./cities.js";
import { BRAND_RULES } from "./../brands/rules.js";
import { nearestMultipleValue, novacutanSupplierOrderQuantity } from "./../order/quantities.js";

export function allocateNorthNeedByCity(total, tyumenFree) {
  const supplierParts = [];
  const tyumenParts = [];
  let freeLeft = Math.max(0, tyumenFree || 0);

  for (const city of NORTH_CITIES.filter((item) => item.key !== "tyumen")) {
    const quantity = Number(total.cities.get(city.key) || 0);
    if (quantity <= 0) continue;
    const fromTyumen = Math.min(quantity, freeLeft);
    const fromSupplier = quantity - fromTyumen;
    freeLeft -= fromTyumen;
    if (fromTyumen > 0) tyumenParts.push({ key: city.key, label: city.label, quantity: Number(fromTyumen.toFixed(2)) });
    if (fromSupplier > 0) supplierParts.push({ key: city.key, label: city.label, quantity: Number(fromSupplier.toFixed(2)) });
  }

  return {
    fromTyumen: tyumenParts.reduce((sum, part) => sum + part.quantity, 0),
    supplierNorthNeed: supplierParts.reduce((sum, part) => sum + part.quantity, 0),
    supplierParts,
    tyumenParts,
  };
}

export function christinaNorthSupplierQuantity(need) {
  return need > 0 ? Math.ceil(Number(need) / 3 - 1e-10) * 3 : null;
}

export function defaultNorthActualSupplierOrder(summary, position, supplierNeed) {
  if (supplierNeed <= 0) return null;
  if (summary.brand === 'christina' || summary.variant === 'home' || summary.variant === 'proff') return christinaNorthSupplierQuantity(supplierNeed);
  if (summary.brand === 'novacutan' || summary.kind === "novacutan") return novacutanSupplierOrderQuantity(position, supplierNeed);
  if (summary.brand === 'klapp' || summary.kind === "klapp") return nearestMultipleValue(supplierNeed, 3);
  const rule=BRAND_RULES[summary.brand];
  const box=Number(position.blankBoxSize);
  const step=rule?.adjustment==='box' && Number.isFinite(box) && box>0 ? Math.ceil(box) : 1;
  // Fractional target stock is valid, but the supplier receives whole units/packs.
  return Math.ceil(supplierNeed/step-1e-10)*step;
}

export function northTotalCityParts(total) {
  return NORTH_CITIES
    .map((city) => ({
      key: city.key,
      label: city.label,
      quantity: Number((total.cities.get(city.key) || 0).toFixed(2)),
    }))
    .filter((city) => city.quantity > 0);
}
