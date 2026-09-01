import { NORTH_CITIES, formatNorthQuantity, northPlanComment } from "../../features/north/northPlan.js";
import { IconDownload } from "../icons.jsx";
import { PrimaryButton } from "../widgets.jsx";

export function NorthPlanTable({
  plan,
  downloads,
  supplierRows,
  tyumenCovered,
  busy,
  displayRow,
  cityQuantities,
  actualValue,
  onCityChange,
  onActualChange,
  onDownload,
}) {
  return (
    <section className="mt-8 overflow-hidden rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-line)] px-5 py-4">
        <div>
          <h2 className="text-[16px] font-semibold">Расчет заказа</h2>
          <p className="mt-1 text-[12.5px] text-[var(--color-ink-soft)]">
            Города: {plan.uploadedCities.join(", ") || "—"}. К заказу у поставщика: {supplierRows}. Закрыто остатком Тюмени: {tyumenCovered}. Перемещений: {plan.transfers.length}.
            {plan.hasTyumenSource ? "" : " Таблица Тюмени не загружена."}
          </p>
        </div>
        <PrimaryButton onClick={onDownload} disabled={busy}>
          <IconDownload className="h-4 w-4" />
          Скачать файлы
        </PrimaryButton>
      </div>
      <div className="max-h-[520px] overflow-auto">
        <table className="w-full min-w-[980px] border-separate border-spacing-0 text-[13px]">
          <thead>
            <tr className="text-left text-[11.5px] font-medium text-[var(--color-ink-faint)]">
              {["Позиция", "По городам", "Нужно северу", "Остаток Тюмени", "Свободно", "Из Тюмени", "Нужно заказать", "Факт. у поставщика", "Комментарий"].map((header) => (
                <th key={header} className="sticky top-0 bg-[var(--color-ground)] px-3 py-2">{header}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {plan.planRows.map((source) => {
              const row = displayRow(source);
              const actual = actualValue(row);
              return (
                <tr key={source.key} className="align-top">
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 font-medium">
                    {row.name}
                    {plan.hasTyumenSource && !row.hasTyumenSource && (
                      <div className="mt-1 text-[11px] font-semibold text-[var(--color-warn)]">нет строки в таблице Тюмени</div>
                    )}
                  </td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2">
                    <div className="grid gap-1.5">
                      {NORTH_CITIES.filter((city) => (cityQuantities(source)[city.key] != null) || (source.cities || []).some((item) => item.key === city.key)).map((city) => (
                        <label key={city.key} className="grid grid-cols-[74px_minmax(54px,1fr)] items-center gap-1.5 text-[11px] text-[var(--color-ink-soft)]">
                          <span className="truncate">{city.label}</span>
                          <input
                            type="number"
                            min="0"
                            step="1"
                            value={cityQuantities(source)[city.key] ?? ""}
                            onChange={(event) => {
                              const next = { ...cityQuantities(source), [city.key]: event.target.value === "" ? 0 : Number(event.target.value) };
                              onCityChange(source.key, next, source, plan);
                            }}
                            className="h-7 rounded-md border border-[var(--color-line)] px-2 font-mono"
                          />
                        </label>
                      ))}
                    </div>
                  </td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 font-mono">{formatNorthQuantity(row.northNeed)}</td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 text-[12px] text-[var(--color-ink-soft)]">
                    ост. {formatNorthQuantity(row.tyumenStock) || "0"}
                    {Number(row.tyumenInTransit || 0) > 0 ? `, в пути ${formatNorthQuantity(row.tyumenInTransit)}` : ""}
                    {Number(row.tyumenTarget || 0) > 0 ? `, цель ${formatNorthQuantity(row.tyumenTarget)}` : ""}
                  </td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 font-mono">{formatNorthQuantity(row.tyumenFree)}</td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 font-mono">{formatNorthQuantity(row.fromTyumen)}</td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 font-mono">{formatNorthQuantity(row.supplierNeed)}</td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2">
                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={actual}
                      onChange={(event) => onActualChange(source.key, event.target.value)}
                      className="h-8 w-24 rounded-md border border-[var(--color-line)] px-2 font-mono"
                    />
                  </td>
                  <td className="border-b border-[var(--color-line-soft)] px-3 py-2 whitespace-pre-line text-[12px] text-[var(--color-ink-soft)]">
                    {northPlanComment(row, actual === "" ? null : Number(actual))}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {downloads.length > 0 && (
        <div className="flex flex-wrap gap-2 border-t border-[var(--color-line)] bg-[var(--color-ground)] px-5 py-3">
          {downloads.map((file) => (
            <a key={file.id} href={file.downloadUrl} download={file.name} className="flex items-center gap-1.5 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] px-3 py-2 text-[12.5px] font-medium text-[var(--color-ink-soft)]">
              <IconDownload className="h-4 w-4" />
              {file.label}
            </a>
          ))}
        </div>
      )}
    </section>
  );
}
