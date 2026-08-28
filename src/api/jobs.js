import { apiClient } from "./client.js";

const TERMINAL_STATUSES = new Set(["needs_review", "completed", "failed"]);

export function createOrderFillJob({ brand, orderMonth, sourceFile, blankFiles }) {
  const formData = new FormData();
  formData.append("brand", brand);
  formData.append("order_month", orderMonth);
  formData.append("source_file", sourceFile);
  for (const file of blankFiles) formData.append("blank_files", file);
  return apiClient.request("/api/v1/jobs/order-fill", {
    method: "POST",
    body: formData,
  });
}

export function createNorthMergeJob({ brand, blankFiles, tyumenSourceFile = null }) {
  const formData = new FormData();
  formData.append("brand", brand);
  for (const entry of blankFiles) formData.append("blank_files", entry.file || entry);
  if (tyumenSourceFile) formData.append("tyumen_source_file", tyumenSourceFile);
  return apiClient.request("/api/v1/jobs/north-merge", {
    method: "POST",
    body: formData,
  });
}

export function getJob(jobId) {
  return apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}`);
}

export function getJobReport(jobId) {
  return apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/report`);
}

export function submitJobEdits(jobId, edits) {
  return apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/edits`, {
    method: "POST",
    body: JSON.stringify({ edits }),
  });
}

export function listJobFiles(jobId) {
  return apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/files`);
}

export async function pollJob(jobId, { intervalMs = 1000, timeoutMs = 120000, onUpdate = () => {} } = {}) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const job = await getJob(jobId);
    onUpdate(job);
    if (TERMINAL_STATUSES.has(job.status)) return job;
    await delay(intervalMs);
  }
  throw new Error("Задача выполняется слишком долго. Попробуйте обновить статус позже.");
}

function delay(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
