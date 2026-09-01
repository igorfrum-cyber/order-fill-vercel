import { useMemo, useState } from "react";
import {
  createOrderFillJob,
  downloadJobArchive,
  FINALIZE_DONE_STATUSES,
  getJobReport,
  listJobFiles,
  pollJob,
  submitJobEdits,
} from "../../api/jobs.js";
import { blankSlotsForSource, brandLabel } from "../../features/brands/brandPresentation.js";
import { runOrderFillJob } from "../../features/jobs/orderJobWorkflow.js";
import { formatOrderMonthLabel } from "../../features/order/monthPolicy.js";
import { collectReviewEdits, hasManualDeviations, initialEditState, patchEdit, rowKey, validateReviewEdits } from "../../features/order/reviewEdits.js";
import { issueReportCsv } from "../../features/report/issueReport.js";
import { combinedSummary, jobProgress, jobStatusText } from "../../features/report/reportModel.js";
import { issueReportRows, qualityWarningLines, qualityWarningSummary } from "../../features/report/qualityWarnings.js";
import { StageRail, TopBar } from "../chrome.jsx";
import { Modal } from "../widgets.jsx";
import { FillStage } from "./FillStage.jsx";
import { PreviewStage } from "./PreviewStage.jsx";
import { UploadStage } from "./SetupUpload.jsx";

function triggerDownload(url, fileName) {
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

function triggerBlobDownload(blob, fileName) {
  const url = URL.createObjectURL(blob);
  triggerDownload(url, fileName || "заполненные-файлы.zip");
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function OrderFillApp({ companyId, resumeJob, onHome }) {
  const [stage, setStage] = useState(resumeJob ? (resumeJob.finalized ? "preview" : "fill") : "upload");
  const [brand, setBrand] = useState(resumeJob?.brand || "");
  const [month, setMonth] = useState(resumeJob?.month || "");
  const [sourceFile, setSourceFile] = useState(null);
  const [blankFiles, setBlankFiles] = useState({});
  const [processing, setProcessing] = useState(false);
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [jobId, setJobId] = useState(resumeJob?.jobId || null);
  const [rows, setRows] = useState(resumeJob?.rows || []);
  const [results, setResults] = useState(resumeJob?.results || []);
  const [edits, setEdits] = useState(() => resumeJob?.edits || new Map());
  const [invalidKeys, setInvalidKeys] = useState(new Set());
  const [busy, setBusy] = useState(false);
  const [confirmLines, setConfirmLines] = useState(null);
  const [outputFiles, setOutputFiles] = useState(resumeJob?.outputFiles || []);
  const [finalized, setFinalized] = useState(Boolean(resumeJob?.finalized));

  const monthLabel = formatOrderMonthLabel(month);
  const uploadSlots = blankSlotsForSource(sourceFile?.name);
  const filesReady = Boolean(sourceFile && uploadSlots.filter((slot) => !slot.optional).every((slot) => blankFiles[slot.id]));
  const summary = useMemo(() => combinedSummary(results, rows), [results, rows]);

  function resetResult() {
    setJobId(null);
    setRows([]);
    setResults([]);
    setEdits(new Map());
    setInvalidKeys(new Set());
    setOutputFiles([]);
    setFinalized(false);
    setError("");
    setStatus("");
    setProgress(0);
    setProcessing(false);
  }

  async function processFiles() {
    const blanks = uploadSlots.map((slot) => blankFiles[slot.id]).filter(Boolean);
    if (!sourceFile || blanks.length < 1) return;
    if (!companyId) {
      setError("Сначала выберите компанию в ленте выгрузок.");
      return;
    }

    setProcessing(true);
    setError("");
    setProgress(0.08);
    setStatus("Отправляю файлы...");
    try {
      const result = await runOrderFillJob({
        api: { createOrderFillJob, pollJob, getJobReport },
        command: {
          sourceFile,
          blankFiles: blanks,
          companyId,
        },
        onStatus: (text, job) => {
          setStatus(text);
          if (job) setProgress(jobProgress(job));
        },
      });
      setJobId(result.jobId);
      setBrand(result.job?.brand || "");
      setMonth(result.job?.order_month || "");
      setRows(result.rows);
      setResults(result.results);
      setEdits(initialEditState(result.rows));
      setStage("fill");
      setStatus("");
      setProgress(1);
    } catch (err) {
      setError(err.message || "Не удалось обработать файлы.");
      setStatus("");
      setProgress(0);
    } finally {
      setProcessing(false);
    }
  }

  function updateEdit(key, patch) {
    setEdits((prev) => patchEdit(prev, key, patch));
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
    triggerBlobDownload(blob, "отчет для исправления в 1С.csv");
  }

  async function openPreview() {
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
    const lines = qualityWarningLines(warnings, { skipDuplicates: true });
    if (lines.length) {
      setConfirmLines(lines);
      return;
    }
    await submitAndPreview();
  }

  async function submitAndPreview() {
    setConfirmLines(null);
    setBusy(true);
    try {
      if (!finalized && hasManualDeviations(rows, edits)) {
        setStatus("Сохраняю правки...");
        const payload = collectReviewEdits(rows, edits);
        const editedJob = await submitJobEdits(jobId, payload);
        const finalJob = await pollJob(editedJob.id, {
          until: FINALIZE_DONE_STATUSES,
          onUpdate: (job) => setStatus(jobStatusText(job)),
        });
        if (finalJob.status === "failed") {
          throw new Error(finalJob.error?.message || "Не удалось подготовить файлы.");
        }
      }
      setStatus("Открываю файлы...");
      const listed = await listJobFiles(jobId);
      setOutputFiles(listed.files);
      setFinalized(true);
      setStage("preview");
      setStatus("");
    } catch (err) {
      setStatus("Ошибка");
      window.alert(err.message || "Не удалось сохранить правки.");
    } finally {
      setBusy(false);
    }
  }

  async function downloadArchive() {
    if (!jobId) return;
    setBusy(true);
    try {
      setStatus("Скачиваю архив...");
      const archive = await downloadJobArchive(jobId);
      triggerBlobDownload(archive.blob, archive.fileName);
      setStatus("Файлы готовы");
    } catch (err) {
      setStatus("Ошибка");
      window.alert(err.message || "Не удалось скачать файлы.");
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
        format="order"
        onHome={onHome}
      />
      <StageRail
        stage={stage}
        brandLabel={brandLabel(brand)}
        monthLabel={monthLabel}
        filesReady={filesReady}
        onGoto={(next) => {
          if (next === "upload" && (stage === "fill" || stage === "preview")) resetResult();
          setStage(next);
        }}
      />
      <main className="flex-1 overflow-hidden">
        {(stage === "upload" || stage === "processing") && (
          <UploadStage
            sourceFile={sourceFile}
            blankFiles={blankFiles}
            onSource={(file) => {
              const previousSlots = blankSlotsForSource(sourceFile?.name)
                .map((slot) => slot.id)
                .join();
              const nextSlots = blankSlotsForSource(file?.name)
                .map((slot) => slot.id)
                .join();
              setSourceFile(file);
              if (previousSlots !== nextSlots) setBlankFiles({});
              resetResult();
            }}
            onBlank={(id, file) => {
              setBlankFiles((prev) => ({ ...prev, [id]: file }));
              resetResult();
            }}
            onHome={onHome}
            onProcess={processFiles}
            processing={processing}
            status={status}
            progress={progress}
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
            status={status}
            busy={busy}
            onDownloadFiles={openPreview}
            onIssueReport={downloadIssueReport}
          />
        )}
        {stage === "preview" && (
          <PreviewStage
            files={outputFiles}
            jobId={jobId}
            status={status}
            busy={busy}
            onBack={() => setStage("fill")}
            onDownload={downloadArchive}
          />
        )}
      </main>
      {confirmLines && (
        <Modal
          title="Проверьте спорные строки"
          cancelLabel="Назад"
          confirmLabel="Продолжить проверку"
          onCancel={() => setConfirmLines(null)}
          onConfirm={submitAndPreview}
        >
          {confirmLines.join("\n")}
        </Modal>
      )}
    </div>
  );
}
