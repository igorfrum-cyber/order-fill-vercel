import { jobStatusText, reportSummaryFromRows } from "../report/reportModel.js";
import { userFacingError } from "../help/errors.js";
import { initialEditState } from "../order/reviewEdits.js";
import { getJob, getJobReport, listJobFiles } from "../../api/jobs.js";

const MISSING_REPORT_ERROR = "Отчёт по этой выгрузке не сохранился. Сверку открыть нельзя.";

export function resumeStage(resume) {
  if (!resume) return "upload";
  if (resume.status === "failed" || resume.error) return "failed";
  if (resume.finalized) return "preview";
  return "fill";
}

function brokenResume(job, error) {
  return {
    jobId: job.id,
    brand: job.brand,
    month: job.order_month,
    status: job.status,
    error,
    rows: [],
    results: [],
    edits: initialEditState([]),
    outputFiles: [],
    finalized: false,
  };
}

export async function runOrderFillJob({ api, command, onStatus = () => {} }) {
  const created = await api.createOrderFillJob(command);
  const job = await api.pollJob(created.id, {
    onUpdate: (updatedJob) => {
      onStatus(jobStatusText(updatedJob), updatedJob);
    },
  });
  if (job.status === "failed") {
    throw new Error(userFacingError({ message: job.error?.message }, "Не удалось обработать файлы."));
  }

  const report = await api.getJobReport(job.id);
  const rows = report.rows || [];
  const summary = report.summary?.brand
    ? report.summary
    : reportSummaryFromRows(rows, job, { brand: command.brand, orderMonth: command.orderMonth });

  return {
    jobId: job.id,
    job,
    report,
    rows,
    results: [{ summary, reportRows: rows }],
  };
}

export async function loadOrderResume(jobId, api = { getJob, getJobReport, listJobFiles }) {
  const job = await api.getJob(jobId);
  if (job.status === "failed") {
    return brokenResume(job, job.error?.message || "");
  }
  const report = await api.getJobReport(jobId).catch(() => null);
  if (!report && job.status === "needs_review") {
    return brokenResume(job, job.error?.message || MISSING_REPORT_ERROR);
  }
  const files = job.status === "completed" ? await api.listJobFiles(jobId).catch(() => ({ files: [] })) : { files: [] };
  const rows = report?.rows || [];
  return {
    jobId: job.id,
    brand: job.brand,
    month: job.order_month,
    status: job.status,
    error: "",
    rows,
    results: report ? [{ summary: report.summary, reportRows: rows }] : [],
    edits: initialEditState(rows),
    outputFiles: files.files || [],
    finalized: job.status === "completed",
  };
}

