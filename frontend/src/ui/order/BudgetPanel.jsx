import { useMemo, useState } from "react";
import { planBudget } from "../../features/order/budgetPlanner.js";
import { budgetPatches, budgetRowsFromReport } from "../../features/order/budgetWorkflow.js";
import { GhostButton, Modal } from "../widgets.jsx";

const money = (value) => Number(value || 0).toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function BudgetPanel({ brand, deliveryWeeks, rows, edits, onEdit }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState("");
  const [plan, setPlan] = useState(null);
  const [error, setError] = useState("");
  const [undo, setUndo] = useState(null);
  const [allowOverSix, setAllowOverSix] = useState(false);
  const [allowBelowOne, setAllowBelowOne] = useState(false);
  const input = useMemo(
    () => budgetRowsFromReport(rows, edits, { brand, deliveryWeeks }),
    [brand, deliveryWeeks, edits, rows],
  );
  if (!input.length) return null;

  const total = input.reduce((sum, row) => sum + row.price * row.quantity, 0);

  function calculate() {
    try {
      const value = Number(String(target).trim().replace(/\s/g, "").replace(",", "."));
      const next = planBudget(input, value, { allowOverSix, allowBelowOne });
      setPlan(next);
      setError(next.reason ? budgetReason(next.reason) : "");
    } catch (err) {
      setPlan(null);
      setError(err.message || "Не удалось пересчитать заказ.");
    }
  }

  function apply() {
    const patches = budgetPatches(plan, edits);
    for (const patch of patches) onEdit(patch.key, patch.next);
    setUndo(patches);
    setOpen(false);
    setPlan(null);
    setError("");
  }

  function undoLast() {
    for (const patch of undo || []) onEdit(patch.key, patch.previous);
    setUndo(null);
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-[13px] text-[var(--color-ink-soft)]">Заказ: <strong className="font-mono text-[var(--color-ink)]">{money(total)} ₽</strong></span>
      <GhostButton onClick={() => { setTarget(String(Math.round(total))); setPlan(null); setError(""); setAllowOverSix(false); setAllowBelowOne(false); setOpen(true); }}>
        Заказ до суммы
      </GhostButton>
      {undo?.length ? <button type="button" onClick={undoLast} className="text-[13px] text-[var(--color-brand-strong)] hover:underline">Отменить перерасчёт</button> : null}
      {open ? (
        <Modal
          title="Заказ до суммы"
          cancelLabel="Закрыть"
          confirmLabel={plan ? "Применить" : "Рассчитать"}
          confirmDisabled={plan ? !plan.complete : !String(target).trim()}
          onCancel={() => setOpen(false)}
          onConfirm={plan ? apply : calculate}
        >
          <label className="block font-medium text-[var(--color-ink)]" htmlFor="budget-target">Целевая сумма, ₽</label>
          <input id="budget-target" inputMode="decimal" className="input mt-2" value={target} onChange={(event) => { setTarget(event.target.value); setPlan(null); setError(""); }} />
          <p className="mt-2">Текущая сумма: {money(total)} ₽. Ручные правки остаются закреплёнными.</p>
          {error ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{error}</p> : null}
          {plan?.reason === "overSix" ? (
            <label className="mt-3 flex items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowOverSix} onChange={(event) => { setAllowOverSix(event.target.checked); setPlan(null); setError(""); }} />
              Разрешить запас больше 6 месяцев и рассчитать снова
            </label>
          ) : null}
          {plan?.reason === "belowOne" ? (
            <label className="mt-3 flex items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowBelowOne} onChange={(event) => { setAllowBelowOne(event.target.checked); setPlan(null); setError(""); }} />
              Разрешить запас меньше 1 месяца и рассчитать снова
            </label>
          ) : null}
          {plan ? (
            <div className="mt-4" aria-label="Предпросмотр бюджета">
              <p className="font-medium text-[var(--color-ink)]">Предпросмотр: {money(plan.before)} ₽ → {money(plan.total)} ₽</p>
              <div className="mt-2 space-y-1">
                {plan.rows.filter((row) => row.quantity !== row.before).map((row) => (
                  <div key={row.key} className="flex justify-between gap-4 rounded-lg bg-[var(--color-ground)] px-3 py-2">
                    <span className="truncate">{row.name} · {row.category}</span>
                    <span className="shrink-0 font-mono">{row.before} → {row.quantity}</span>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </Modal>
      ) : null}
    </div>
  );
}

function budgetReason(reason) {
  if (reason === "overSix") return "Расчёт остановлен у запаса в 6 месяцев.";
  if (reason === "belowOne") return "Расчёт остановлен у защитного запаса в 1 месяц.";
  return reason;
}
