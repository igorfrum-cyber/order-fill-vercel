import { jobStatusText } from "../report/reportModel.js";
import { normalizeNorthResult } from "../north/northPlan.js";

export async function runNorthMergeJob({ api, command, onStatus = () => {} }) {
  const created = await api.createNorthMergeJob(command);
  const job = await api.pollJob(created.id, {
    onUpdate: (updatedJob) => {
      onStatus(jobStatusText(updatedJob), updatedJob);
    },
  });
  if (job.status === "failed") {
    throw new Error(job.error?.message || "Задача завершилась с ошибкой.");
  }

  const report = await api.getJobReport(job.id);
  return {
    jobId: job.id,
    job,
    report,
    plan: normalizeNorthResult(report, job),
  };
}
