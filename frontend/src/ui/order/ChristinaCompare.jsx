import { useEffect, useState } from "react";
import { christinaFastJSON, christinaMismatchJSON, christinaStandardJSON } from "../../api/budget.js";

const FILES = [
  { filename: "christina-proff-текущий.json", label: "Текущий JSON", build: christinaStandardJSON },
  { filename: "christina-proff-быстрый.json", label: "Быстрый JSON", build: christinaFastJSON },
  { filename: "christina-proff-расхождения.json", label: "Расхождения JSON", build: christinaMismatchJSON },
];

export function ChristinaCompare({ plan }) {
  if (!plan?.compare) return null;
  const { compare } = plan;
  return (
    <div className="mt-3 space-y-2">
      {compare.match ? (
        <p role="status" className="rounded-xl border border-[var(--color-line)] bg-[var(--color-brand-soft)] px-4 py-3 text-[13px] text-[var(--color-ink)]">
          Сверка CHRISTINA: совпало. Текущий {compare.standardMs} мс, быстрый {compare.fastMs} мс.
        </p>
      ) : (
        <div role="status" className="rounded-xl border border-[var(--color-danger)] bg-[var(--color-surface)] px-4 py-3 text-[13px] text-[var(--color-danger)]">
          <p>
            Сверка CHRISTINA: не совпало. Текущий {compare.standardMs} мс, быстрый {compare.fastMs} мс. Применяется текущий расчёт.
          </p>
          {compare.mismatches.length ? (
            <ul className="mt-2 list-disc space-y-1 pl-4 font-mono text-[12px] text-[var(--color-ink)]">
              {compare.mismatches.map((item, index) => (
                <li key={`${item.where}-${item.key}-${index}`}>
                  {item.where}
                  {item.key ? ` / ${item.key}` : ""}
                  {item.field ? ` ${item.field}` : ""}: {item.want} → {item.got}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        {FILES.map((file) => (
          <BlobDownload key={file.filename} filename={file.filename} label={file.label} data={file.build(plan)} />
        ))}
      </div>
    </div>
  );
}

function BlobDownload({ filename, label, data }) {
  const json = JSON.stringify(data, null, 2);
  const [href, setHref] = useState("");
  useEffect(() => {
    const url = URL.createObjectURL(new Blob([json], { type: "application/json" }));
    setHref(url);
    return () => URL.revokeObjectURL(url);
  }, [json]);
  if (!href) return null;
  return (
    <a
      href={href}
      download={filename}
      className="rounded-control border border-[var(--color-line)] bg-[var(--color-surface)] px-3 py-2 text-[13px] font-medium text-[var(--color-ink-soft)] hover:border-[var(--color-ink-faint)] hover:text-[var(--color-ink)]"
    >
      {label}
    </a>
  );
}
