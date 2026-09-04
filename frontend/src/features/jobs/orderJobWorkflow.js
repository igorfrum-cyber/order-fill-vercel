import { jobStatusText, reportSummaryFromRows } from "../report/reportModel.js";
import { userFacingError } from "../help/errors.js";

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
