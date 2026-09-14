import { useMemo, useState } from "react";
import { budgetChangeComment, budgetOrderRules, budgetTargetValue, discountValue, planBudget } from "../../features/order/budgetPlanner.js";
import { GhostButton, Modal } from "../widgets.jsx";

const money = (value) => Number(value || 0).toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function NorthBudgetPanel({ brand, plan, rows, actualValue, lockedKeys, discounts, onApply, onDiscount }) {
  const groups = [...new Set(rows.map((row) => row.variant || "main"))];
  return (
    <div className="flex flex-wrap gap-3">
      {groups.map((group) => (
        <NorthBudgetGroup
          key={group}
          label={group === "main" ? "" : group.toUpperCase()}
          brand={brand}
          deliveryWeeks={plan.summary?.deliveryWeeks || 1}
          rows={rows.filter((row) => (row.variant || "main") === group)}
          actualValue={actualValue}
          lockedKeys={lockedKeys}
          appliedDiscount={discounts.get(group) || 0}
          onApply={onApply}
          onDiscount={(discount) => onDiscount(group, discount)}
        />
      ))}
    </div>
  );
}

function NorthBudgetGroup({ label, brand, deliveryWeeks, rows, actualValue, lockedKeys, appliedDiscount, onApply, onDiscount }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState("");
  const [discount, setDiscount] = useState("0");
  const [preview, setPreview] = useState(null);
  const [error, setError] = useState("");
  const input = useMemo(() => budgetRows(rows, actualValue, lockedKeys, brand, deliveryWeeks, appliedDiscount), [rows, actualValue, lockedKeys, brand, deliveryWeeks, appliedDiscount]);
  if (!input.length) return null;
  const current = input.reduce((sum, row) => sum + row.price * row.quantity, 0);
  const changed = preview?.rows.filter((row) => row.quantity !== row.before) || [];

  function calculate() {
    try {
      const percent = discountValue(discount);
      const planned = planBudget(budgetRows(rows, actualValue, lockedKeys, brand, deliveryWeeks, percent), budgetTargetValue(target));
      setPreview({ ...planned, discount: percent });
      setError(planned.reason ? budgetReason(planned.reason) : "");
    } catch (err) {
      setPreview(null);
      setError(err.message || "Не удалось пересчитать заказ.");
    }
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-[13px] text-[var(--color-ink-soft)]">{label ? `${label}: ` : ""}<strong className="font-mono text-[var(--color-ink)]">{money(current)} ₽</strong></span>
      <GhostButton onClick={() => { setTarget(String(Math.round(current))); setDiscount(String(appliedDiscount).replace(".", ",")); setPreview(null); setError(""); setOpen(true); }}>Заказ до суммы{label ? ` ${label}` : ""}</GhostButton>
      {open ? (
        <Modal
          title={`Заказ до суммы${label ? ` · ${label}` : ""}`}
          cancelLabel="Закрыть"
          confirmLabel={preview ? "Применить" : "Рассчитать"}
          confirmDisabled={preview ? !preview.complete || !changed.length : !target.trim()}
          onCancel={() => setOpen(false)}
          onConfirm={preview ? () => { changed.forEach((row) => onApply(row.key, row.quantity, budgetChangeComment(row))); onDiscount(preview.discount); setOpen(false); } : calculate}
          wide
        >
          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label="Текущий заказ" value={`${money(current)} ₽`} />
            <label className="rounded-xl border border-[var(--color-line)] px-4 py-3 font-medium">Целевая сумма, ₽<input className="input mt-2 font-mono" inputMode="decimal" value={target} onChange={(event) => { setTarget(event.target.value); setPreview(null); }} /></label>
            <label className="rounded-xl border border-[var(--color-line)] px-4 py-3 font-medium">Скидка, %<input className="input mt-2 font-mono" inputMode="decimal" value={discount} onChange={(event) => { setDiscount(event.target.value); setPreview(null); }} /></label>
          </div>
          <p className="mt-3 text-[13px] text-[var(--color-ink-soft)]">Ручные значения закреплены. Перемещения городам не меняются; добавка остаётся на складе доставки.</p>
          {error ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{error}</p> : null}
          {preview ? <div aria-label="Предпросмотр бюджета" className="mt-4 space-y-2"><Metric label="После расчёта" value={`${money(preview.before)} ₽ → ${money(preview.total)} ₽`} />{changed.map((row) => <div key={row.key} className="flex justify-between rounded-lg border border-[var(--color-line-soft)] px-3 py-2"><span>{row.name} · {row.category}</span><span className="font-mono">{row.before} → {row.quantity}</span></div>)}</div> : null}
        </Modal>
      ) : null}
    </div>
  );
}

function budgetRows(rows, actualValue, lockedKeys, brand, deliveryWeeks, discount) {
  const factor = 1 - discount / 100;
  return rows.filter((row) => row.hasBudgetData).map((row) => ({
    key: row.key,
    name: row.name,
    category: row.budgetCategory,
    quantity: Number(actualValue(row) || 0),
    price: Math.round(Number(row.budgetPrice || 0) * factor * 100) / 100,
    demand: Number(row.budgetDemand || 0),
    delivery: Number(deliveryWeeks || 1) * 0.25,
    stock: Number(row.tyumenStock || 0),
    transit: Number(row.tyumenInTransit || 0),
    outbound: Number(row.northNeed || 0),
    ...budgetOrderRules(brand, row.name, row.blankBoxSize),
    locked: lockedKeys.has(row.key),
    excluded: false,
    unsafe: false,
  }));
}

function Metric({ label, value }) {
  return <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3"><span className="block text-[13px]">{label}</span><strong className="mt-1 block font-mono text-[20px]">{value}</strong></div>;
}

function budgetReason(reason) {
  if (reason === "overSix") return "Расчёт остановлен у запаса в 6 месяцев.";
  if (reason === "belowOne") return "Расчёт остановлен у защитного запаса в 1 месяц.";
  return reason;
}
