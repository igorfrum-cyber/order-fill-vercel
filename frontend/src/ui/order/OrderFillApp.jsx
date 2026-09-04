import { useEffect, useMemo, useState } from "react";
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
import { needsEditResubmit } from "../../features/preview/previewEdits.js";
import { issueReportCsv } from "../../features/report/issueReport.js";
import { combinedSummary, jobProgress, jobStatusText } from "../../features/report/reportModel.js";
import { issueReportRows, qualityWarningLines, qualityWarningSummary } from "../../features/report/qualityWarnings.js";
import { userFacingError } from "../../features/help/errors.js";
import { StageRail, TopBar } from "../chrome.jsx";
import { ErrorBoundary } from "../ErrorBoundary.jsx";
import { GhostButton, Modal } from "../widgets.jsx";
import { FillStage } from "./FillStage.jsx";
import { PreviewStage } from "./PreviewStage.jsx";
import { CommentGate } from "./review/CommentGate.jsx";
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

export function OrderFillApp({ companyId, resumeJob, onHome, onHelp, onStage }) {
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
  const [commentGateKeys, setCommentGateKeys] = useState(null);
  const [afterCommentGate, setAfterCommentGate] = useState("preview");
  const [banner, setBanner] = useState("");
  const [outputFiles, setOutputFiles] = useState(resumeJob?.outputFiles || []);
  const [finalized, setFinalized] = useState(Boolean(resumeJob?.finalized));
  const [editsDirty, setEditsDirty] = useState(false);
  const [previewEpoch, setPreviewEpoch] = useState(0);

  useEffect(() => {
    onStage?.(stage);
  }, [stage, onStage]);

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
    setCommentGateKeys(null);
    setOutputFiles([]);
    setFinalized(false);
    setEditsDirty(false);
    setPreviewEpoch(0);
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
      setEditsDirty(false);
      setStage("fill");
      setStatus("");
      setProgress(1);
    } catch (err) {
      setError(userFacingError(err, "Не удалось обработать файлы."));
      setStatus("");
      setProgress(0);
    } finally {
      setProcessing(false);
    }
  }

  function updateEdit(key, patch) {
    setEdits((prev) => patchEdit(prev, key, patch));
    setEditsDirty(true);
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
      setBanner("Нет спорных строк для отчета.");
      return;
    }
    const blob = new Blob([`\ufeff${issueReportCsv(issueRows, (row) => edits.get(rowKey(row)) || { comment: "" })}`], {
      type: "text/csv;charset=utf-8",
    });
    triggerBlobDownload(blob, "отчет для исправления в 1С.csv");
  }

  async function openPreview() {
    if (!jobId) {
      setBanner("Сначала заполните бланк.");
      return;
    }
    const invalid = validateReviewEdits(rows, edits);
    if (invalid.length) {
      setInvalidKeys(new Set(invalid));
      setCommentGateKeys(invalid);
      setAfterCommentGate("preview");
      setBanner("");
      return;
    }
    proceedToPreview();
  }

  function confirmCommentGate() {
    const invalid = validateReviewEdits(rows, edits);
    if (invalid.length) {
      setInvalidKeys(new Set(invalid));
      setCommentGateKeys(invalid);
      return;
    }
    setCommentGateKeys(null);
    setInvalidKeys(new Set());
    if (afterCommentGate === "download") {
      downloadArchive();
      return;
    }
    proceedToPreview();
  }

  function proceedToPreview() {
    setBanner("");
    const warnings = qualityWarningSummary({ rows, results, edits });
    const lines = qualityWarningLines(warnings, { skipDuplicates: true });
    if (lines.length) {
      setConfirmLines(lines);
      return;
    }
    submitAndPreview();
  }

  async function persistEditsIfNeeded() {
    if (!needsEditResubmit({
      finalized,
      dirty: editsDirty,
      hasDeviations: hasManualDeviations(rows, edits),
    })) {
      return false;
    }
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
    setEditsDirty(false);
    setFinalized(true);
    setPreviewEpoch((epoch) => epoch + 1);
    return true;
  }

  async function submitAndPreview() {
    setConfirmLines(null);
    setBusy(true);
    try {
      await persistEditsIfNeeded();
      setStatus("Открываю файлы...");
      const listed = await listJobFiles(jobId);
      setOutputFiles(listed.files);
      setFinalized(true);
      setStage("preview");
      setStatus("Собираю сетку...");
    } catch (err) {
      setStatus("Ошибка");
      setBanner(userFacingError(err, "Не удалось сохранить правки."));
    } finally {
      setBusy(false);
    }
  }

  function backToFill() {
    setStage("fill");
    setStatus("");
  }

  async function downloadArchive() {
    if (!jobId) return;
    const invalid = validateReviewEdits(rows, edits);
    if (invalid.length) {
      setInvalidKeys(new Set(invalid));
      setCommentGateKeys(invalid);
      setAfterCommentGate("download");
      return;
    }
    setBusy(true);
    try {
      await persistEditsIfNeeded();
      setStatus("Скачиваю архив...");
      const archive = await downloadJobArchive(jobId);
      triggerBlobDownload(archive.blob, archive.fileName);
      setStatus("Файлы готовы");
    } catch (err) {
      setStatus(userFacingError(err, "Не удалось скачать файлы."));
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
        onHelp={onHelp}
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
      <main className="relative min-h-0 flex-1 overflow-hidden">
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
            banner={banner}
            onDownloadFiles={openPreview}
            onIssueReport={downloadIssueReport}
          />
        )}
        {stage === "preview" && (
          <ErrorBoundary
            key={jobId || "preview"}
            fallback={(err) => (
              <div className="grid h-full place-items-center bg-[var(--color-ground)] px-6">
                <div className="max-w-md text-center">
                  <h2 className="text-[22px] font-semibold tracking-tight">Не получилось открыть файлы</h2>
                  <p className="mt-2 text-[15px] leading-relaxed text-[var(--color-ink-soft)]">
                    {userFacingError(err, "Попробуйте вернуться к правкам и открыть ещё раз.")}
                  </p>
                  {err?.message ? (
                    <p className="mt-3 break-all font-mono text-[12px] text-[var(--color-ink-faint)]">{String(err.message)}</p>
                  ) : null}
                  <div className="mt-5 flex justify-center">
                    <GhostButton onClick={backToFill}>Назад к правкам</GhostButton>
                  </div>
                </div>
              </div>
            )}
          >
            <PreviewStage
              files={outputFiles}
              jobId={jobId}
              status={status}
              busy={busy}
              rows={rows}
              edits={edits}
              onEdit={updateEdit}
              refreshKey={previewEpoch}
              onBack={backToFill}
              onReady={() => setStatus("")}
              onDownload={downloadArchive}
            />
          </ErrorBoundary>
        )}
      </main>
      {commentGateKeys && (
        <CommentGate
          rows={rows.filter((row) => commentGateKeys.includes(rowKey(row)))}
          edits={edits}
          onEdit={updateEdit}
          onCancel={() => setCommentGateKeys(null)}
          onConfirm={confirmCommentGate}
        />
      )}
      {confirmLines && !commentGateKeys && (
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
