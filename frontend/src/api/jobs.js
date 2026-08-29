import { apiClient } from "./client.js";
import { mapOutputFile, mapReport, toManualEditPayload } from "./mappers.js";

const TERMINAL_STATUSES = new Set(["needs_review", "completed", "failed"]);

export const DEFAULT_POLL_INTERVAL_MS = 200;
export const DEFAULT_POLL_TIMEOUT_MS = 120000;

const absoluteUrl = (path) => apiClient.absoluteUrl(path);

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

export async function getJobReport(jobId) {
  return mapReport(await apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/report`));
}

export function submitJobEdits(jobId, edits) {
  return apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/edits`, {
    method: "POST",
    body: JSON.stringify({ edits: edits.map(toManualEditPayload) }),
  });
}

export async function listJobFiles(jobId) {
  const payload = await apiClient.request(`/api/v1/jobs/${encodeURIComponent(jobId)}/files`);
  return { files: (payload.files || []).map((file) => mapOutputFile(file, absoluteUrl)) };
}

export function downloadJobArchive(jobId) {
  return apiClient.requestDownload(`/api/v1/jobs/${encodeURIComponent(jobId)}/archive`);
}

export async function pollJob(jobId, { intervalMs = DEFAULT_POLL_INTERVAL_MS, timeoutMs = DEFAULT_POLL_TIMEOUT_MS, onUpdate = () => {} } = {}) {
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
