import { useEffect, useMemo, useState } from "react";
import { planOrderBudget } from "../../api/budget.js";
import { budgetTargetValue, discountValue } from "../../features/order/budgetInput.js";
import { GhostButton, Modal } from "../widgets.jsx";
import { ChristinaCompare } from "../order/ChristinaCompare.jsx";

const money = (value) => Number(value || 0).toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function NorthBudgetPanel({ brand, plan, rows, actualValue, lockedKeys, discounts, priceColumns = new Map(), onApply, onDiscount, onPriceColumn = () => {} }) {
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
          appliedPriceColumn={priceColumns.get(group) || 0}
          onPriceColumn={(column) => onPriceColumn(group, column)}
          onApply={onApply}
          onDiscount={(discount) => onDiscount(group, discount)}
        />
      ))}
    </div>
  );
}

function NorthBudgetGroup({ label, brand, deliveryWeeks, rows, actualValue, lockedKeys, appliedDiscount, appliedPriceColumn, onApply, onDiscount, onPriceColumn }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState("");
  const [priceColumn, setPriceColumn] = useState(String(appliedPriceColumn || ""));
  const options = brand === "sothys" ? [] : [...new Map(rows.flatMap((row) => row.budgetPrices || []).map((price) => [String(price.column), price])).values()];
  const [discount, setDiscount] = useState("0");
  const [preview, setPreview] = useState(null);
  const [error, setError] = useState("");
  const [allowOverSix,setAllowOverSix]=useState(false);
  const [allowBelowOne,setAllowBelowOne]=useState(false);
  const [quote,setQuote]=useState(null);
  const input=useMemo(()=>budgetRows(rows,actualValue,lockedKeys,open?priceColumn:appliedPriceColumn),[rows,actualValue,lockedKeys,open,priceColumn,appliedPriceColumn]);
  let quoteDiscount=appliedDiscount;
  if(open){try{quoteDiscount=discountValue(discount);}catch{quoteDiscount=null;}}
  const quoteKey=JSON.stringify({brand,discount:quoteDiscount,delivery_weeks:deliveryWeeks || 1,rows:input});
  useEffect(()=>{
    let active=true;const request=JSON.parse(quoteKey);
    if(request.discount==null || !request.rows.length)return;
    planOrderBudget({...request,target:0,quote_only:true})
     .then(result=>{if(active)setQuote({key:quoteKey,result});})
     .catch(err=>{if(active)setQuote({key:quoteKey,error:err.message || "Не удалось получить сумму заказа."});});
    return()=>{active=false;};
  },[quoteKey]);
  if(!input.length)return null;
  const current=quote?.key===quoteKey ? quote.result?.total : null;
  const currentLabel=current==null ? "Считаю сумму…" : `${money(current)} ₽`;
  const quoteError=quote?.key===quoteKey ? quote.error : "";
  function resetDraft(){setPreview(null);setError("");setAllowOverSix(false);setAllowBelowOne(false);}
  const changed = preview?.rows.filter((row) => row.quantity !== row.before) || [];

  async function calculate() {
    try {
      const percent = discountValue(discount);
      const value = budgetTargetValue(target);
      const planned = await planOrderBudget({
        brand,
        target: value,
        discount: percent,
        allow_over_six:allowOverSix,
        allow_below_one:allowBelowOne,
        delivery_weeks: deliveryWeeks || 1,
        rows: budgetRows(rows, actualValue, lockedKeys, priceColumn),
      });
      setPreview({ ...planned, discount: percent, priceColumn: Number(priceColumn) || 0 });
      setError(planned.reason ? budgetReason(planned.reason) : "");
    } catch (err) {
      setPreview(null);
      setError(err.message || "Не удалось пересчитать заказ.");
    }
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-[13px] text-[var(--color-ink-soft)]">{label ? `${label}: ` : ""}<strong className="font-mono text-[var(--color-ink)]">{currentLabel}</strong></span>
      <GhostButton onClick={() => { setTarget(current==null ? "" : String(Math.round(current))); setPriceColumn(String(appliedPriceColumn || "")); setDiscount(String(appliedDiscount).replace(".", ",")); resetDraft(); setOpen(true); }}>Заказ до суммы{label ? ` ${label}` : ""}</GhostButton>
      {open ? (
        <Modal
          title={`Заказ до суммы${label ? ` · ${label}` : ""}`}
          cancelLabel="Закрыть"
          confirmLabel={preview ? "Применить" : "Рассчитать"}
          confirmDisabled={preview ? !preview.complete : !target.trim()}
          onCancel={() => setOpen(false)}
          onConfirm={preview ? () => { changed.forEach((row) => onApply(row.key, row.quantity, row.comment)); onDiscount(preview.discount); onPriceColumn(preview.priceColumn); setOpen(false); } : calculate}
          wide
        >
          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label="Текущий заказ" value={currentLabel} />
            <label className="rounded-xl border border-[var(--color-line)] px-4 py-3 font-medium">Целевая сумма, ₽<input className="input mt-2 font-mono" inputMode="decimal" value={target} onChange={(event) => { setTarget(event.target.value); resetDraft(); }} /></label>
            <label className="rounded-xl border border-[var(--color-line)] px-4 py-3 font-medium">Скидка, %<input className="input mt-2 font-mono" inputMode="decimal" value={discount} onChange={(event) => { setDiscount(event.target.value); resetDraft(); }} /></label>
          </div>
          {options.length > 1 ? <label className="mt-3 block font-medium">Колонка цены
            <select className="input mt-2" value={priceColumn} onChange={(event) => { setPriceColumn(event.target.value); resetDraft(); }}>
              <option value="">Распознанная цена</option>
              {options.map((price) => <option key={price.column} value={price.column}>{price.label}</option>)}
            </select>
          </label> : null}
          <p className="mt-3 text-[13px] text-[var(--color-ink-soft)]">Ручные значения закреплены. Перемещения городам не меняются; добавка остаётся на складе доставки.</p>
          {quoteError ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{quoteError}</p> : null}
          {error ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{error}</p> : null}
          {preview?.reason==="overSix" ? <label className="mt-3 flex items-start gap-2"><input type="checkbox" checked={allowOverSix} onChange={event=>{setAllowOverSix(event.target.checked);setPreview(null);setError("");}}/>Разрешить запас больше 6 месяцев и рассчитать снова</label> : null}
          {preview?.reason==="belowOne" ? <label className="mt-3 flex items-start gap-2"><input type="checkbox" checked={allowBelowOne} onChange={event=>{setAllowBelowOne(event.target.checked);setPreview(null);setError("");}}/>Разрешить запас меньше 1 месяца и рассчитать снова</label> : null}
          {preview ? (
            <div aria-label="Предпросмотр бюджета" className="mt-4 space-y-2">
              <Metric label="После расчёта" value={`${money(preview.before)} ₽ → ${money(preview.total)} ₽`} />
              <ChristinaCompare plan={preview} />
              {changed.map((row) => (
                <div key={row.key} className="flex justify-between rounded-lg border border-[var(--color-line-soft)] px-3 py-2">
                  <span>{row.name} · {row.category}</span>
                  <span className="font-mono">{row.before} → {row.quantity}</span>
                </div>
              ))}
            </div>
          ) : null}
        </Modal>
      ) : null}
    </div>
  );
}

// budgetRows maps North plan rows to raw budget-plan API rows. Pricing and brand
// order rules run in calculation-service; the browser only shapes data. Priced
// CHRISTINA PROFF line members are included even without sales demand so the
// line stays complete and earns the set discount (mirrors origin/main north).
function budgetRows(rows, actualValue, lockedKeys, priceColumn = "") {
  return rows
    .map((row) => ({
      key: row.key,
      name: row.name,
      category: row.budgetCategory,
      quantity: Number(actualValue(row) || 0),
      base_price: Number(priceColumn ? row.budgetPrices?.find((price) => String(price.column) === String(priceColumn))?.price ?? 0 : row.budgetPrice || 0),
      demand: Number(row.budgetDemand || 0),
      stock: Number(row.tyumenStock || 0),
      transit: Number(row.tyumenInTransit || 0),
      outbound: Number(row.northNeed || 0),
      box_size: Number(row.blankBoxSize || 0),
      group: row.variant || "main",
      line: row.christinaLine || null,
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
