import { useRef, useState } from "react";
import { blankSlotsForSource } from "../../features/brands/brandPresentation.js";
import { IconCheck, IconChevron, IconFile, IconUpload } from "../icons.jsx";
import { PrimaryButton, ProgressBar, StageHeading } from "../widgets.jsx";

export function UploadStage({
  sourceFile,
  blankFiles,
  onSource,
  onBlank,
  onHome,
  onProcess,
  processing,
  status,
  progress,
  error,
}) {
  const slots = blankSlotsForSource(sourceFile?.name);
  const ready = Boolean(sourceFile && slots.filter((slot) => !slot.optional).every((slot) => blankFiles[slot.id]));
  return (
    <div className="mx-auto flex h-full max-w-2xl flex-col justify-center px-6">
      <StageHeading index="01" kicker="Бланк закупки" title="Загрузите файлы">
        <p className="mt-3 max-w-lg text-[16px] leading-relaxed text-[var(--color-ink-soft)]">
          Таблица продаж из 1С и бланк поставщика. Бренд и месяц заказа читаются из таблицы. Для CHRISTINA приложите HOME и PROFF.
        </p>
      </StageHeading>

      <div className={`mt-9 grid gap-4 ${slots.length > 1 ? "sm:grid-cols-3" : "sm:grid-cols-2"}`}>
        <Dropzone
          title="Таблица продаж"
          hint="Excel с остатками и рекомендациями"
          file={sourceFile}
          accept=".xlsx,.xlsm,.xls"
          onPick={onSource}
        />
        {slots.map((slot) => (
          <Dropzone
            key={slot.id}
            title={slot.optional ? `${slot.label} (необязательно)` : slot.label}
            hint={slot.hint}
            file={blankFiles[slot.id] || null}
            accept={slot.accept}
            onPick={(file) => onBlank(slot.id, file)}
          />
        ))}
      </div>

      {error && <p className="mt-4 text-[15px] text-[var(--color-danger)]">{error}</p>}
      {processing && <ProgressBar value={progress} label={status || "Обработка..."} />}

      <div className="mt-8 flex items-center justify-between">
        <button
          type="button"
          onClick={onHome}
          disabled={processing}
          className="flex items-center gap-1.5 rounded-xl px-4 py-2.5 text-[15px] font-medium text-[var(--color-ink-soft)] transition hover:bg-[var(--color-line-soft)] disabled:opacity-40"
        >
          <IconChevron className="h-4 w-4 rotate-90" />
          К выгрузкам
        </button>
        <PrimaryButton onClick={onProcess} disabled={!ready || processing}>
          {processing ? status || "Обработка..." : "Обработать"}
          <IconChevron className="h-4 w-4 -rotate-90" />
        </PrimaryButton>
      </div>
    </div>
  );
}

function Dropzone({ title, hint, file, accept, onPick }) {
  const [drag, setDrag] = useState(false);
  const inputRef = useRef(null);
  return (
    <div
      onDragOver={(event) => {
        event.preventDefault();
        setDrag(true);
      }}
      onDragLeave={() => setDrag(false)}
      onDrop={(event) => {
        event.preventDefault();
        setDrag(false);
        const next = event.dataTransfer.files[0];
        if (next) onPick(next);
      }}
      className={`relative rounded-xl border-2 border-dashed p-6 transition ${
        file
          ? "border-[var(--color-ok)] bg-[var(--color-ok-soft)]"
          : drag
            ? "border-[var(--color-brand)] bg-[var(--color-brand-soft)]"
            : "border-[var(--color-line)] bg-[var(--color-surface)] hover:border-[var(--color-ink-faint)]"
      }`}
    >
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="hidden"
        onChange={(event) => onPick(event.target.files?.[0] || null)}
      />
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[15px] font-semibold">{title}</span>
        {file && (
          <span className="grid h-5 w-5 place-items-center rounded-full bg-[var(--color-ok)] text-white">
            <IconCheck className="h-3 w-3" />
          </span>
        )}
      </div>
      {file ? (
        <div className="flex items-center gap-2.5">
          <IconFile className="h-8 w-8 shrink-0 text-[var(--color-ok)]" />
          <div className="min-w-0 flex-1">
            <div className="truncate font-mono text-[13px] font-medium">{file.name}</div>
            <button
              type="button"
              onClick={() => {
                if (inputRef.current) inputRef.current.value = "";
                onPick(null);
              }}
              className="mt-0.5 text-[11px] font-medium text-[var(--color-ink-soft)] underline-offset-2 hover:underline"
            >
              Заменить
            </button>
          </div>
        </div>
      ) : (
        <button type="button" onClick={() => inputRef.current?.click()} className="flex w-full flex-col items-start gap-1 text-left">
          <IconUpload className="h-8 w-8 text-[var(--color-ink-faint)]" />
          <span className="mt-1 text-[14px] font-medium text-[var(--color-brand)]">Выбрать файл</span>
          <span className="text-[13px] text-[var(--color-ink-faint)]">{hint}</span>
        </button>
      )}
    </div>
  );
}
