import { useMemo, useState } from "react";
import { budgetTargetValue, discountValue, planBudget } from "../../features/order/budgetPlanner.js";
import { budgetPatches, budgetRowsFromReport } from "../../features/order/budgetWorkflow.js";
import { GhostButton, Modal } from "../widgets.jsx";

const money = (value) => Number(value || 0).toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const targetInput = (value) => /^\d{0,12}(?:[.,]\d{0,2})?$/.test(String(value).replace(/\s/g, ""));
const discountInput = (value) => /^\d{0,2}(?:[.,]\d{0,2})?%?$/.test(String(value).trim());

export function BudgetPanel({ brand, deliveryWeeks, rows, edits, onEdit }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState("");
  const [pricingMode, setPricingMode] = useState("net");
  const [discount, setDiscount] = useState("30");
  const [appliedDiscount, setAppliedDiscount] = useState(0);
  const [plan, setPlan] = useState(null);
  const [error, setError] = useState("");
  const [inputError, setInputError] = useState("");
  const [undo, setUndo] = useState(null);
  const [allowOverSix, setAllowOverSix] = useState(false);
  const [allowBelowOne, setAllowBelowOne] = useState(false);
  const input = useMemo(
    () => budgetRowsFromReport(rows, edits, { brand, deliveryWeeks, discount: appliedDiscount }),
    [appliedDiscount, brand, deliveryWeeks, edits, rows],
  );
  if (!input.length) return null;

  const total = input.reduce((sum, row) => sum + row.price * row.quantity, 0);
  const changed = plan?.rows.filter((row) => row.quantity !== row.before) || [];

  function resetPreview() {
    setPlan(null);
    setError("");
  }

  function currentDraftDiscount() {
    return pricingMode === "gross" ? discountValue(discount) : 0;
  }

  function calculate() {
    try {
      const value = budgetTargetValue(target);
      const nextDiscount = currentDraftDiscount();
      const pricedRows = budgetRowsFromReport(rows, edits, { brand, deliveryWeeks, discount: nextDiscount });
      const next = planBudget(pricedRows, value, { allowOverSix, allowBelowOne });
      setPlan({ ...next, discount: nextDiscount });
      setError(next.reason ? budgetReason(next.reason) : "");
      setInputError("");
    } catch (err) {
      setPlan(null);
      setError(err.message || "Не удалось пересчитать заказ.");
    }
  }

  function apply() {
    const patches = budgetPatches(plan, edits);
    if (!patches.length) return;
    for (const patch of patches) onEdit(patch.key, patch.next);
    setUndo({ patches, discount: appliedDiscount });
    setAppliedDiscount(plan.discount);
    setOpen(false);
    resetPreview();
  }

  function undoLast() {
    for (const patch of undo?.patches || []) onEdit(patch.key, patch.previous);
    setAppliedDiscount(undo?.discount || 0);
    setUndo(null);
  }

  function openDialog() {
    setTarget(String(Math.round(total)));
    setPricingMode(appliedDiscount > 0 ? "gross" : "net");
    if (appliedDiscount > 0) setDiscount(String(appliedDiscount).replace(".", ","));
    setAllowOverSix(false);
    setAllowBelowOne(false);
    setInputError("");
    resetPreview();
    setOpen(true);
  }

  const canCalculate = validBudgetInputs(target, pricingMode, discount);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-[13px] text-[var(--color-ink-soft)]">
        Заказ: <strong className="font-mono text-[var(--color-ink)]">{money(total)} ₽</strong>
        {appliedDiscount > 0 ? <span> · скидка {String(appliedDiscount).replace(".", ",")}%</span> : null}
      </span>
      <GhostButton onClick={openDialog}>Заказ до суммы</GhostButton>
      {undo?.patches.length ? <button type="button" onClick={undoLast} className="cursor-pointer text-[13px] text-[var(--color-brand-strong)] hover:underline">Отменить перерасчёт</button> : null}
      {open ? (
        <Modal
          title="Заказ до суммы"
          cancelLabel="Закрыть"
          confirmLabel={plan ? "Применить" : "Рассчитать"}
          confirmDisabled={plan ? !plan.complete || !changed.length : !canCalculate}
          onCancel={() => setOpen(false)}
          onConfirm={plan ? apply : calculate}
          wide
        >
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3">
              <span className="block text-[13px]">Текущий заказ</span>
              <strong className="mt-1 block font-mono text-[20px] text-[var(--color-ink)]">{money(total)} ₽</strong>
            </div>
            <label className="block rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 font-medium text-[var(--color-ink)]" htmlFor="budget-target">
              Целевая сумма, ₽
              <input
                id="budget-target"
                inputMode="decimal"
                autoComplete="off"
                aria-invalid={Boolean(inputError)}
                className="input mt-2 font-mono"
                value={target}
                onChange={(event) => {
                  const value = event.target.value;
                  if (!targetInput(value)) {
                    setInputError("Сумма: только цифры и не больше двух знаков после запятой.");
                    return;
                  }
                  setTarget(value);
                  setInputError("");
                  resetPreview();
                }}
              />
            </label>
          </div>

          <fieldset className="mt-4">
            <legend className="font-medium text-[var(--color-ink)]">Как указана цена в бланке</legend>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {[
                ["net", "Цена уже со скидкой", "Использовать цену без изменений"],
                ["gross", "Вычесть скидку", "Уменьшить цену для расчёта суммы"],
              ].map(([value, label, hint]) => (
                <label key={value} className={`flex min-h-16 cursor-pointer items-start gap-3 rounded-xl border px-4 py-3 transition-colors ${pricingMode === value ? "border-[var(--color-brand)] bg-[var(--color-brand-soft)]" : "border-[var(--color-line)] bg-[var(--color-surface)] hover:border-[var(--color-ink-faint)]"}`}>
                  <input type="radio" name="budget-price-mode" value={value} checked={pricingMode === value} onChange={() => { setPricingMode(value); setInputError(""); resetPreview(); }} />
                  <span><strong className="block font-medium text-[var(--color-ink)]">{label}</strong><span className="text-[13px]">{hint}</span></span>
                </label>
              ))}
            </div>
          </fieldset>

          {pricingMode === "gross" ? (
            <label className="mt-3 block max-w-xs font-medium text-[var(--color-ink)]" htmlFor="budget-discount">
              Скидка, %
              <input
                id="budget-discount"
                inputMode="decimal"
                autoComplete="off"
                aria-invalid={Boolean(inputError)}
                className="input mt-2 font-mono"
                value={discount}
                onChange={(event) => {
                  const value = event.target.value;
                  if (!discountInput(value)) {
                    setInputError("Скидка: число от 0 до 99,99 с двумя знаками после запятой.");
                    return;
                  }
                  setDiscount(value);
                  setInputError("");
                  resetPreview();
                }}
              />
            </label>
          ) : null}
          <p className="mt-2 text-[13px]">Скидка влияет на подбор количества. Исходные цены в Excel не меняются. Ручные правки остаются закреплёнными.</p>
          {inputError ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{inputError}</p> : null}
          {error ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{error}</p> : null}

          {plan?.reason === "overSix" ? (
            <label className="mt-3 flex cursor-pointer items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowOverSix} onChange={(event) => { setAllowOverSix(event.target.checked); resetPreview(); }} />
              Разрешить запас больше 6 месяцев и рассчитать снова
            </label>
          ) : null}
          {plan?.reason === "belowOne" ? (
            <label className="mt-3 flex cursor-pointer items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowBelowOne} onChange={(event) => { setAllowBelowOne(event.target.checked); resetPreview(); }} />
              Разрешить запас меньше 1 месяца и рассчитать снова
            </label>
          ) : null}

          {plan ? (
            <div className="mt-4" aria-label="Предпросмотр бюджета" aria-live="polite">
              <div className="flex flex-wrap items-end justify-between gap-2 rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3">
                <div>
                  <span className="block text-[13px]">После расчёта</span>
                  <strong className="font-mono text-[20px] text-[var(--color-ink)]">{money(plan.before)} ₽ → {money(plan.total)} ₽</strong>
                </div>
                <span className="text-[13px]">Изменено позиций: {changed.length}</span>
              </div>
              {changed.length ? (
                <div className="mt-2 space-y-1">
                  {changed.map((row) => (
                    <div key={row.key} className="grid gap-1 rounded-lg border border-[var(--color-line-soft)] bg-[var(--color-surface)] px-3 py-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-4">
                      <span className="truncate text-[var(--color-ink)]">{row.name} · {row.category}</span>
                      <span className="font-mono text-[var(--color-ink)]">{row.before} → {row.quantity}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="mt-2">Заказ уже соответствует указанной сумме.</p>}
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

function validBudgetInputs(target, pricingMode, discount) {
  try {
    budgetTargetValue(target);
    if (pricingMode === "gross") discountValue(discount);
    return true;
  } catch {
    return false;
  }
}
