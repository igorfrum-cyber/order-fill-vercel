import { useState } from "react";
import {
  createNorthMergeJob,
  FINALIZE_DONE_STATUSES,
  getJobReport,
  listJobFiles,
  pollJob,
  submitJobEdits,
} from "../../api/jobs.js";
import { ORDER_BRANDS, usesChristinaSplitBlank } from "../../features/brands/brandPresentation.js";
import { runNorthMergeJob } from "../../features/jobs/northJobWorkflow.js";
import { sameNorthFile, uniqueNorthFiles } from "../../features/north/northFiles.js";
import { defaultNorthActual, recalculateNorthRow } from "../../features/north/northPlan.js";
import { jobStatusText } from "../../features/report/reportModel.js";
import {
  excelAcceptHint,
  northDuplicateFileMessage,
  northMissingCityBlankMessage,
  northSelectedCount,
  northUploadSteps,
  selectedFileCountLabel,
} from "../../features/jobs/uploadCopy.js";
import { IconChevron } from "../icons.jsx";
import { Field, GhostButton, PrimaryButton, Select } from "../widgets.jsx";
import { TopBar } from "../chrome.jsx";
import { NorthPlanTable } from "./NorthPlanTable.jsx";
import { NorthPrompts } from "./NorthPrompts.jsx";
import { NorthUploadPanel } from "./NorthUploadPanel.jsx";

export function NorthApp({ companyId, onHome, onHelp }) {
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
      setError(northDuplicateFileMessage);
      return;
    }
    setError("");
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
      setError(northMissingCityBlankMessage);
      return;
    }
    if (!companyId) {
      setError("Сначала выберите компанию в ленте выгрузок.");
      return;
    }
    setBusy(true);
    setError("");
    setStatus("Проверяю бланки...");
    setDownloads([]);
    try {
      const jobResult = await runNorthMergeJob({
        api: { createNorthMergeJob, pollJob, getJobReport },
        command: { brand, blankFiles: entries, tyumenSourceFile: tyumenFile, companyId },
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
      setError("Сначала соедините бланки.");
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
        until: FINALIZE_DONE_STATUSES,
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
      setError(err.message || "Не удалось подготовить файлы.");
    } finally {
      setBusy(false);
    }
  }

  const supplierRows = plan?.planRows.filter((row) => Number(displayRow(row).supplierNeed || 0) > 0).length || 0;
  const tyumenCovered = plan?.planRows.filter((row) => Number(displayRow(row).fromTyumen || 0) > 0).length || 0;

  return (
    <div className="flex h-full flex-col bg-[var(--color-ground)]">
      <TopBar brandLabel="" monthLabel="" stage="setup" format="north" onHome={onHome} onHelp={onHelp} />
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
              <ol className="mt-2 max-w-xl list-none text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
                {northUploadSteps().map((step) => (
                  <li key={step.n}>
                    {step.n}. {step.title}
                  </li>
                ))}
              </ol>
              <p className="mt-2 text-[13px] text-[var(--color-ink-faint)]">
                {excelAcceptHint} {selectedFileCountLabel(northSelectedCount({ files, homeFiles, proffFiles, tyumenFile }))}
              </p>
            </div>
            <div className="w-64">
              <Field label="Бренд">
                <Select value={brand} onChange={changeBrand} options={ORDER_BRANDS.map((item) => ({ value: item.id, label: item.label }))} />
              </Field>
            </div>
          </div>

          <NorthUploadPanel
            christina={christina}
            files={files}
            homeFiles={homeFiles}
            proffFiles={proffFiles}
            tyumenFile={tyumenFile}
            onAdd={addFiles}
            onRemove={removeFile}
            onPickTyumen={(file) => {
              setTyumenFile(file);
              resetPlan();
            }}
          />

          {error ? (
            <div role="alert" className="mt-4 rounded-lg border border-[var(--color-danger)]/25 bg-[var(--color-danger-soft)] px-4 py-3 text-[14px] text-[var(--color-danger)]">
              {error}
            </div>
          ) : null}

          <div className="mt-6 flex items-center justify-between">
            <GhostButton onClick={onHome}>
              <IconChevron className="h-4 w-4 rotate-90" />
              К выгрузкам
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
            <NorthPlanTable
              plan={plan}
              downloads={downloads}
              supplierRows={supplierRows}
              tyumenCovered={tyumenCovered}
              busy={busy}
              displayRow={displayRow}
              cityQuantities={cityQuantities}
              actualValue={actualValue}
              onCityChange={(key, next, source, currentPlan) => {
                setCityEdits(new Map(cityEdits).set(key, next));
                if (!manualActual.has(key)) {
                  const calculated = recalculateNorthRow(source, next);
                  setActualEdits(new Map(actualEdits).set(key, defaultNorthActual(calculated, calculated.supplierNeed, currentPlan.summary || {})));
                }
              }}
              onActualChange={(key, value) => {
                setManualActual(new Set(manualActual).add(key));
                setActualEdits(new Map(actualEdits).set(key, value));
              }}
              onDownload={downloadPlan}
            />
          )}
        </div>
      </div>

      <NorthPrompts
        mergePrompt={mergePrompt}
        shortPrompt={shortPrompt}
        onCancelMerge={() => setMergePrompt(null)}
        onConfirmMerge={acceptMerge}
        onCancelShort={() => setShortPrompt(null)}
        onConfirmShort={submitNorthDownloads}
      />
    </div>
  );
}
