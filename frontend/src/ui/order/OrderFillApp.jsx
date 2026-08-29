import { useMemo, useState } from "react";
import {
  createOrderFillJob,
  getJobReport,
  listJobFiles,
  pollJob,
  submitJobEdits,
} from "../../api/jobs.js";
import { blankSlotsForBrand, brandLabel, usesChristinaSplitBlank } from "../../features/brands/brandPresentation.js";
import { runOrderFillJob } from "../../features/jobs/orderJobWorkflow.js";
import { defaultOrderMonth, formatOrderMonthLabel, sanitizeOrderMonth } from "../../features/order/monthPolicy.js";
import { collectReviewEdits, initialEditState, rowKey, validateReviewEdits } from "../../features/order/reviewEdits.js";
import { issueReportCsv } from "../../features/report/issueReport.js";
import { combinedSummary, jobStatusText } from "../../features/report/reportModel.js";
import { issueReportRows, qualityWarningLines, qualityWarningSummary } from "../../features/report/qualityWarnings.js";
import { StageRail, TopBar } from "../chrome.jsx";
import { Modal } from "../widgets.jsx";
import { FillStage } from "./FillStage.jsx";
import { SetupStage, UploadStage } from "./SetupUpload.jsx";

function triggerDownload(url, fileName) {
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

export function OrderFillApp({ mode, onMode }) {
  const [stage, setStage] = useState("setup");
  const [brand, setBrand] = useState("angiopharm");
  const [month, setMonth] = useState(() => defaultOrderMonth());
  const [sourceFile, setSourceFile] = useState(null);
  const [blankFiles, setBlankFiles] = useState({});
  const [processing, setProcessing] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [jobId, setJobId] = useState(null);
  const [rows, setRows] = useState([]);
  const [results, setResults] = useState([]);
  const [edits, setEdits] = useState(new Map());
  const [invalidKeys, setInvalidKeys] = useState(new Set());
  const [files, setFiles] = useState([]);
  const [busy, setBusy] = useState(false);
  const [confirmLines, setConfirmLines] = useState(null);

  const monthLabel = formatOrderMonthLabel(month);
  const filesReady = Boolean(sourceFile && blankSlotsForBrand(brand).every((slot) => blankFiles[slot.id]));
  const summary = useMemo(() => combinedSummary(results, rows), [results, rows]);

  function resetResult() {
    setJobId(null);
    setRows([]);
    setResults([]);
    setEdits(new Map());
    setInvalidKeys(new Set());
    setFiles([]);
    setError("");
    setStatus("");
    setProcessing(false);
  }

  function changeBrand(next) {
    setBrand(next);
    setBlankFiles({});
    resetResult();
    if (stage === "fill") setStage("upload");
  }

  function changeMonth(next) {
    setMonth(sanitizeOrderMonth(next));
    resetResult();
    if (stage === "fill") setStage("upload");
  }

  async function processFiles() {
    const slots = blankSlotsForBrand(brand);
    const blanks = usesChristinaSplitBlank(brand)
      ? slots.map((slot) => blankFiles[slot.id])
      : [blankFiles.main];
    if (!sourceFile || blanks.some((file) => !file)) return;

    setProcessing(true);
    setError("");
    setStatus("Обработка...");
    setFiles([]);
    try {
      const result = await runOrderFillJob({
        api: { createOrderFillJob, pollJob, getJobReport },
        command: {
          brand,
          orderMonth: month,
          sourceFile,
          blankFiles: blanks,
        },
        onStatus: (text) => setStatus(text),
      });
      setJobId(result.jobId);
      setRows(result.rows);
      setResults(result.results);
      setEdits(initialEditState(result.rows));
      setStage("fill");
      setStatus("");
    } catch (err) {
      setError(err.message || "Не удалось обработать файлы.");
      setStatus("");
    } finally {
      setProcessing(false);
    }
  }

  function updateEdit(key, next) {
    setEdits((prev) => {
      const copy = new Map(prev);
      copy.set(key, next);
      return copy;
    });
    setInvalidKeys((prev) => {
      if (!prev.has(key)) return prev;
      const copy = new Set(prev);
      copy.delete(key);
      return copy;
    });
  }

  function downloadIssueReport() {
    const issueRows = issueReportRows(rows);
    if (!issueRows.length) {
      window.alert("Нет спорных строк для отчета.");
      return;
    }
    const blob = new Blob([`\ufeff${issueReportCsv(issueRows, (row) => edits.get(rowKey(row)) || { comment: "" })}`], {
      type: "text/csv;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    triggerDownload(url, "отчет для исправления в 1С.csv");
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  async function finalizeDownloads() {
    if (!jobId) {
      window.alert("Сначала заполните бланк.");
      return;
    }
    const invalid = validateReviewEdits(rows, edits);
    if (invalid.length) {
      setInvalidKeys(new Set(invalid));
      window.alert("Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.");
      return;
    }
    const warnings = qualityWarningSummary({ rows, results, edits });
    const lines = qualityWarningLines(warnings);
    if (lines.length) {
      setConfirmLines(lines);
      return;
    }
    await submitAndDownload();
  }

  async function submitAndDownload() {
    setConfirmLines(null);
    setBusy(true);
    setStatus("Сохраняю правки...");
    try {
      const payload = collectReviewEdits(rows, edits);
      const editedJob = await submitJobEdits(jobId, payload);
      const finalJob = await pollJob(editedJob.id, {
        onUpdate: (job) => setStatus(jobStatusText(job)),
      });
      if (finalJob.status === "failed") {
        throw new Error(finalJob.error?.message || "Не удалось подготовить файлы.");
      }
      const listed = await listJobFiles(finalJob.id);
      setFiles(listed.files || []);
      setStatus("Файлы готовы");
    } catch (err) {
      setStatus("Ошибка");
      window.alert(err.message || "Не удалось сохранить правки.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex h-full flex-col bg-[var(--color-ground)]">
      <TopBar
        brandLabel={brandLabel(brand)}
        monthLabel={monthLabel}
        stage={stage}
        mode={mode}
        onMode={onMode}
      />
      <StageRail
        stage={stage}
        brandLabel={brandLabel(brand)}
        monthLabel={monthLabel}
        filesReady={filesReady}
        onGoto={(next) => {
          if (next === "upload" && stage === "fill") resetResult();
          setStage(next);
        }}
      />
      <main className="flex-1 overflow-hidden">
        {stage === "setup" && (
          <SetupStage
            brand={brand}
            month={month}
            onBrand={changeBrand}
            onMonth={changeMonth}
            onNext={() => setStage("upload")}
          />
        )}
        {(stage === "upload" || stage === "processing") && (
          <UploadStage
            brand={brand}
            sourceFile={sourceFile}
            blankFiles={blankFiles}
            onSource={(file) => {
              setSourceFile(file);
              resetResult();
            }}
            onBlank={(id, file) => {
              setBlankFiles((prev) => ({ ...prev, [id]: file }));
              resetResult();
            }}
            onBack={() => setStage("setup")}
            onProcess={processFiles}
            processing={processing}
            status={status}
            error={error}
          />
        )}
        {stage === "fill" && (
          <FillStage
            brand={brand}
            rows={rows}
            edits={edits}
            onEdit={updateEdit}
            invalidKeys={invalidKeys}
            summary={summary}
            files={files}
            status={status}
            busy={busy}
            onDownloadFiles={finalizeDownloads}
            onIssueReport={downloadIssueReport}
          />
        )}
      </main>
      {confirmLines && (
        <Modal
          title="Проверьте спорные строки"
          cancelLabel="Назад"
          confirmLabel="Продолжить скачивание"
          onCancel={() => setConfirmLines(null)}
          onConfirm={submitAndDownload}
        >
          {confirmLines.join("\n")}
        </Modal>
      )}
    </div>
  );
}
