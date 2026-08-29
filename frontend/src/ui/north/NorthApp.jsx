import { useRef, useState } from "react";
import {
  createNorthMergeJob,
  getJobReport,
  listJobFiles,
  pollJob,
  submitJobEdits,
} from "../../api/jobs.js";
import { ORDER_BRANDS, usesChristinaSplitBlank } from "../../features/brands/brandPresentation.js";
import { runNorthMergeJob } from "../../features/jobs/northJobWorkflow.js";
import { sameNorthFile, uniqueNorthFiles } from "../../features/north/northFiles.js";
import {
  NORTH_CITIES,
  defaultNorthActual,
  formatNorthQuantity,
  northPlanComment,
  recalculateNorthRow,
} from "../../features/north/northPlan.js";
import { jobStatusText } from "../../features/report/reportModel.js";
import { IconCheck, IconChevron, IconDownload, IconFile, IconUpload, IconX } from "../icons.jsx";
import { Field, GhostButton, Modal, PrimaryButton, Select } from "../widgets.jsx";
import { TopBar } from "../chrome.jsx";

export function NorthApp({ mode, onMode }) {
  const [brand, setBrand] = useState("angiopharm");
  const [files, setFiles] = useState([]);
  const [homeFiles, setHomeFiles] = useState([]);
  const [proffFiles, setProffFiles] = useState([]);
  const [tyumenFile, setTyumenFile] = useState(null);
  const [status, setStatus] = useState("Готов к загрузке");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [jobId, setJobId] = useState(null);
  const [plan, setPlan] = useState(null);
  const [downloads, setDownloads] = useState([]);
  const [mergePrompt, setMergePrompt] = useState(null);
  const [shortPrompt, setShortPrompt] = useState(null);
  const [cityEdits, setCityEdits] = useState(new Map());
  const [actualEdits, setActualEdits] = useState(new Map());
  const [manualActual, setManualActual] = useState(new Set());

  const christina = usesChristinaSplitBlank(brand);

  function resetPlan() {
    setPlan(null);
    setJobId(null);
    setDownloads([]);
    setCityEdits(new Map());
    setActualEdits(new Map());
    setManualActual(new Set());
    setError("");
  }

  function changeBrand(next) {
    setBrand(next);
    setFiles([]);
    setHomeFiles([]);
    setProffFiles([]);
    setTyumenFile(null);
    resetPlan();
    setStatus("Готов к загрузке");
  }

  function addFiles(group, incoming) {
    const current = group === "home" ? homeFiles : group === "proff" ? proffFiles : files;
    const uniqueIncoming = uniqueNorthFiles(incoming);
    const next = [...current];
    let added = 0;
    for (const file of uniqueIncoming) {
      if (next.some((item) => sameNorthFile(item, file))) continue;
      next.push(file);
      added += 1;
    }
    if (!added) {
      window.alert("Все выбранные бланки уже добавлены.");
      return;
    }
    if (group === "home") setHomeFiles(next);
    else if (group === "proff") setProffFiles(next);
    else setFiles(next);
    resetPlan();
  }

  function removeFile(group, index) {
    if (group === "home") setHomeFiles(homeFiles.filter((_, i) => i !== index));
    else if (group === "proff") setProffFiles(proffFiles.filter((_, i) => i !== index));
    else setFiles(files.filter((_, i) => i !== index));
    resetPlan();
  }

  function entriesForMerge() {
    if (christina) {
      return [
        ...uniqueNorthFiles(homeFiles).map((file) => ({ file, variant: "home", variantLabel: "HOME" })),
        ...uniqueNorthFiles(proffFiles).map((file) => ({ file, variant: "proff", variantLabel: "PROFF" })),
      ];
    }
    return uniqueNorthFiles(files).map((file) => ({ file }));
  }

  async function startMerge() {
    const entries = entriesForMerge();
    if (!entries.length) {
      window.alert("Добавьте хотя бы один бланк города.");
      return;
    }
    setBusy(true);
    setError("");
    setStatus("Проверяю бланки...");
    setDownloads([]);
    try {
      const jobResult = await runNorthMergeJob({
        api: { createNorthMergeJob, pollJob, getJobReport },
        command: { brand, blankFiles: entries, tyumenSourceFile: tyumenFile },
        onStatus: (text) => setStatus(text),
      });
      setMergePrompt({ entries, result: jobResult });
      setStatus("Готово");
    } catch (err) {
      setError(err.message || "Не удалось соединить бланки.");
      setStatus("Ошибка");
    } finally {
      setBusy(false);
    }
  }

  function acceptMerge() {
    const jobResult = mergePrompt.result;
    setMergePrompt(null);
    setJobId(jobResult.jobId);
    setPlan(jobResult.plan);
    const nextActual = new Map();
    for (const row of jobResult.plan.planRows) {
      nextActual.set(row.key, row.actualSupplierOrder == null ? "" : row.actualSupplierOrder);
    }
    setActualEdits(nextActual);
    setCityEdits(new Map());
    setManualActual(new Set());
    setStatus("Проверьте расчет");
  }

  function displayRow(source) {
    const quantities = cityEdits.get(source.key);
    return quantities ? recalculateNorthRow(source, quantities) : source;
  }

  function cityQuantities(source) {
    if (cityEdits.has(source.key)) return cityEdits.get(source.key);
    const quantities = {};
    for (const city of source.cities || []) quantities[city.key] = city.quantity;
    return quantities;
  }

  function actualValue(row) {
    if (actualEdits.has(row.key)) return actualEdits.get(row.key);
    return row.actualSupplierOrder == null ? "" : row.actualSupplierOrder;
  }

  function shortOrderWarnings() {
    if (!plan) return [];
    const warnings = [];
    for (const source of plan.planRows) {
      const calculated = displayRow(source);
      const actual = actualValue(calculated) === "" ? 0 : Number(actualValue(calculated));
      const need = Number(calculated.supplierNeed || 0);
      if (calculated.allowsRoundedShortSupplierOrder && actual === defaultNorthActual(calculated, need, plan.summary || {})) continue;
      if (Number.isFinite(actual) && actual < need) {
        warnings.push({ name: calculated.name, actual, need });
      }
    }
    return warnings;
  }

  async function downloadPlan() {
    if (!plan || !jobId) {
      window.alert("Сначала соедините бланки.");
      return;
    }
    const warnings = shortOrderWarnings();
    if (warnings.length) {
      setShortPrompt(warnings);
      return;
    }
    await submitNorthDownloads();
  }

  async function submitNorthDownloads() {
    setShortPrompt(null);
    setBusy(true);
    setStatus("Готовлю файлы...");
    setDownloads([]);
    try {
      const edits = plan.planRows.map((source) => ({
        key: source.key,
        cities: cityQuantities(source),
        actualSupplierOrder: actualValue(source) === "" ? null : Number(actualValue(source)),
      }));
      if (edits.some((edit) => edit.actualSupplierOrder != null && edit.actualSupplierOrder < 0)) {
        throw new Error("Фактический заказ у поставщика не может быть отрицательным.");
      }
      const editedJob = await submitJobEdits(jobId, edits);
      const finalJob = await pollJob(editedJob.id, {
        onUpdate: (job) => setStatus(jobStatusText(job)),
      });
      if (finalJob.status === "failed") {
        throw new Error(finalJob.error?.message || "Не удалось подготовить файлы.");
      }
      const payload = await listJobFiles(finalJob.id);
      setDownloads(payload.files || []);
      setStatus("Файлы готовы");
    } catch (err) {
      setStatus("Ошибка");
      window.alert(err.message || "Не удалось подготовить файлы.");
    } finally {
      setBusy(false);
    }
  }

  const supplierRows = plan?.planRows.filter((row) => Number(displayRow(row).supplierNeed || 0) > 0).length || 0;
  const tyumenCovered = plan?.planRows.filter((row) => Number(displayRow(row).fromTyumen || 0) > 0).length || 0;

  return (
    <div className="flex h-full flex-col bg-[var(--color-ground)]">
      <TopBar brandLabel="" monthLabel="" stage="setup" mode={mode} onMode={onMode} />
      <div className="flex-1 overflow-auto px-6 py-6">
        <div className="mx-auto max-w-6xl">
          <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
            <div>
              <div className="mb-2 flex items-center gap-2 text-[12px] font-medium text-[var(--color-brand)]">
                <span className="font-mono text-[11px]">СЕВЕР</span>
                <span className="h-px w-6 bg-[var(--color-brand)]/40" />
                Объединение бланков городов
              </div>
              <h1 className="text-[28px] font-semibold tracking-tight">Соединить бланки</h1>
              <p className="mt-2 max-w-xl text-[14px] text-[var(--color-ink-soft)]">
                Загрузите заполненные бланки северных городов и таблицу Тюмени — сервис посчитает перемещения и заказ у поставщика.
              </p>
            </div>
            <div className="w-64">
              <Field label="Бренд">
                <Select value={brand} onChange={changeBrand} options={ORDER_BRANDS.map((item) => ({ value: item.id, label: item.label }))} />
              </Field>
            </div>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            {christina ? (
              <>
                <MultiDropzone title="Бланки HOME" hint="HOME-бланки городов" files={homeFiles} onAdd={(incoming) => addFiles("home", incoming)} onRemove={(index) => removeFile("home", index)} />
                <MultiDropzone title="Бланки PROFF" hint="PROFF-бланки городов" files={proffFiles} onAdd={(incoming) => addFiles("proff", incoming)} onRemove={(index) => removeFile("proff", index)} />
              </>
            ) : (
              <MultiDropzone title="Бланк города" hint="Один или несколько заполненных бланков" files={files} onAdd={(incoming) => addFiles("default", incoming)} onRemove={(index) => removeFile("default", index)} />
            )}
            <SingleDropzone title="Заполненная таблица Тюмени" hint="Для учета остатков и в пути" file={tyumenFile} onPick={(file) => { setTyumenFile(file); resetPlan(); }} />
          </div>

          {error && <p className="mt-4 text-[13px] text-[var(--color-danger)]">{error}</p>}

          <div className="mt-6 flex items-center justify-between">
            <GhostButton onClick={() => onMode("order")}>
              <IconChevron className="h-4 w-4 rotate-90" />
              К бланку
            </GhostButton>
            <div className="flex items-center gap-3">
              <span className="font-mono text-[11px] text-[var(--color-ink-soft)]">{status}</span>
              <PrimaryButton onClick={startMerge} disabled={busy}>
                Соединить бланки
                <IconChevron className="h-4 w-4 -rotate-90" />
              </PrimaryButton>
            </div>
          </div>

          {plan && (
            <section className="mt-8 overflow-hidden rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-line)] px-5 py-4">
                <div>
                  <h2 className="text-[16px] font-semibold">Расчет заказа</h2>
                  <p className="mt-1 text-[12.5px] text-[var(--color-ink-soft)]">
                    Города: {plan.uploadedCities.join(", ") || "—"}. К заказу у поставщика: {supplierRows}. Закрыто остатком Тюмени: {tyumenCovered}. Перемещений: {plan.transfers.length}.
                    {plan.hasTyumenSource ? "" : " Таблица Тюмени не загружена."}
                  </p>
                </div>
                <PrimaryButton onClick={downloadPlan} disabled={busy}>
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
                                      setCityEdits(new Map(cityEdits).set(source.key, next));
                                      if (!manualActual.has(source.key)) {
                                        const calculated = recalculateNorthRow(source, next);
                                        setActualEdits(new Map(actualEdits).set(source.key, defaultNorthActual(calculated, calculated.supplierNeed, plan.summary || {})));
                                      }
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
                              onChange={(event) => {
                                setManualActual(new Set(manualActual).add(source.key));
                                setActualEdits(new Map(actualEdits).set(source.key, event.target.value));
                              }}
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
          )}
        </div>
      </div>

      {mergePrompt && (
        <Modal
          title="Соединить бланки городов"
          cancelLabel="Назад"
          confirmLabel="Соединить"
          onCancel={() => setMergePrompt(null)}
          onConfirm={acceptMerge}
        >
          <p className="mb-2">Будут объединены эти бланки:</p>
          {(mergePrompt.result.plan.confirmationGroups?.length
            ? mergePrompt.result.plan.confirmationGroups.map((group) => `${group.city.label}: ${group.variants.join(", ")}`)
            : mergePrompt.entries.map((entry) => entry.file.name)
          ).map((text) => (
            <div key={text} className="mt-1 rounded-lg border border-[var(--color-line)] bg-[var(--color-ground)] px-3 py-2 text-[13px] text-[var(--color-ink)]">
              {text}
            </div>
          ))}
        </Modal>
      )}

      {shortPrompt && (
        <Modal
          title="Есть позиции ниже потребности"
          cancelLabel="Назад"
          confirmLabel="Все равно продолжить"
          onCancel={() => setShortPrompt(null)}
          onConfirm={submitNorthDownloads}
        >
          {`По этим позициям факт у поставщика меньше общей нехватки Тюмени и северных городов.\n\n${shortPrompt.slice(0, 8).map((item) => `• ${item.name}: факт ${formatNorthQuantity(item.actual) || "0"}, нужно ${formatNorthQuantity(item.need)}`).join("\n")}${shortPrompt.length > 8 ? `\nЕще позиций: ${shortPrompt.length - 8}` : ""}\n\nМожно вернуться и поправить цифры или продолжить скачивание как есть.`}
        </Modal>
      )}
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
