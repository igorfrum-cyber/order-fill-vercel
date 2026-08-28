import { jobStatusText, normalizeReportRow, reportSummaryFromRows } from "../report/reportModel.js";

export async function runOrderFillJob({ api, command, onStatus = () => {} }) {
  const created = await api.createOrderFillJob(command);
  const job = await api.pollJob(created.id, {
    onUpdate: (updatedJob) => {
      onStatus(jobStatusText(updatedJob), updatedJob);
    },
  });
  if (job.status === "failed") {
    throw new Error(job.error?.message || "Задача завершилась с ошибкой.");
  }

  const report = await api.getJobReport(job.id);
  const rows = (report.rows || []).map(normalizeReportRow);
  const summary = report.summary || reportSummaryFromRows(rows, job, {
    brand: command.brand,
    orderMonth: command.orderMonth,
  });
  return {
    jobId: job.id,
    job,
    report,
    rows,
    results: [{ summary, reportRows: rows }],
  };
}
