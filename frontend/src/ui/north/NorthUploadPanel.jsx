import { useRef } from "react";
import { excelAcceptHint, northUploadSteps } from "../../features/jobs/uploadCopy.js";
import { IconCheck, IconFile, IconUpload, IconX } from "../icons.jsx";

export function NorthUploadPanel({ christina, files, homeFiles, proffFiles, tyumenFile, onAdd, onRemove, onPickTyumen }) {
  const steps = northUploadSteps();
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {christina ? (
        <>
          <MultiDropzone title={`1. Бланки HOME`} hint={excelAcceptHint} files={homeFiles} onAdd={(incoming) => onAdd("home", incoming)} onRemove={(index) => onRemove("home", index)} />
          <MultiDropzone title={`1. Бланки PROFF`} hint={excelAcceptHint} files={proffFiles} onAdd={(incoming) => onAdd("proff", incoming)} onRemove={(index) => onRemove("proff", index)} />
        </>
      ) : (
        <MultiDropzone title={`${steps[0].n}. ${steps[0].title}`} hint={excelAcceptHint} files={files} onAdd={(incoming) => onAdd("default", incoming)} onRemove={(index) => onRemove("default", index)} />
      )}
      <SingleDropzone title={`${steps[1].n}. ${steps[1].title}`} hint={excelAcceptHint} file={tyumenFile} onPick={onPickTyumen} />
    </div>
  );
}

function MultiDropzone({ title, hint, files, onAdd, onRemove }) {
  const inputRef = useRef(null);
  return (
    <div className="rounded-xl border-2 border-dashed border-[var(--color-line)] bg-[var(--color-surface)] p-5">
      <input ref={inputRef} type="file" accept=".xlsx,.xlsm,.xls" multiple className="hidden" onChange={(event) => {
        onAdd(Array.from(event.target.files || []));
        event.target.value = "";
      }} />
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[13px] font-semibold">{title}</span>
        {files.length > 0 && <span className="font-mono text-[11px] text-[var(--color-ok)]">{files.length}</span>}
      </div>
      <button type="button" onClick={() => inputRef.current?.click()} className="flex w-full flex-col items-start gap-1 text-left">
        <IconUpload className="h-8 w-8 text-[var(--color-ink-faint)]" />
        <span className="mt-1 text-[12.5px] font-medium text-[var(--color-brand)]">Выбрать файлы</span>
        <span className="text-[11.5px] text-[var(--color-ink-faint)]">{hint}</span>
      </button>
      {files.length > 0 && (
        <div className="mt-3 grid gap-2">
          {files.map((file, index) => (
            <div key={`${file.name}-${index}`} className="flex items-center gap-2 rounded-lg border border-[var(--color-line)] bg-[var(--color-ground)] px-3 py-2">
              <IconFile className="h-4 w-4 text-[var(--color-ok)]" />
              <span className="min-w-0 flex-1 truncate font-mono text-[12px]">{file.name}</span>
              <button type="button" onClick={() => onRemove(index)} className="text-[var(--color-ink-faint)] hover:text-[var(--color-danger)]" aria-label={`Удалить ${file.name}`}>
                <IconX className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SingleDropzone({ title, hint, file, onPick }) {
  const inputRef = useRef(null);
  return (
    <div className={`rounded-xl border-2 border-dashed p-5 ${file ? "border-[var(--color-ok)] bg-[var(--color-ok-soft)]" : "border-[var(--color-line)] bg-[var(--color-surface)]"}`}>
      <input ref={inputRef} type="file" accept=".xlsx,.xlsm,.xls" className="hidden" onChange={(event) => onPick(event.target.files?.[0] || null)} />
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[13px] font-semibold">{title}</span>
        {file && (
          <span className="grid h-5 w-5 place-items-center rounded-full bg-[var(--color-ok)] text-white">
            <IconCheck className="h-3 w-3" />
          </span>
        )}
      </div>
      {file ? (
        <div className="flex items-center gap-2.5">
          <IconFile className="h-8 w-8 text-[var(--color-ok)]" />
          <div className="min-w-0 flex-1">
            <div className="truncate font-mono text-[12px] font-medium">{file.name}</div>
            <button type="button" onClick={() => { if (inputRef.current) inputRef.current.value = ""; onPick(null); }} className="mt-0.5 text-[11px] font-medium text-[var(--color-ink-soft)] underline-offset-2 hover:underline">
              Заменить
            </button>
          </div>
        </div>
      ) : (
        <button type="button" onClick={() => inputRef.current?.click()} className="flex w-full flex-col items-start gap-1 text-left">
          <IconUpload className="h-8 w-8 text-[var(--color-ink-faint)]" />
          <span className="mt-1 text-[12.5px] font-medium text-[var(--color-brand)]">Выбрать файл</span>
          <span className="text-[11.5px] text-[var(--color-ink-faint)]">{hint}</span>
        </button>
      )}
    </div>
  );
}
