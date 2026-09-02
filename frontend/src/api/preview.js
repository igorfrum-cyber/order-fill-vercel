import { apiClient } from "./client.js";

export function getPreviewMeta(jobId, fileId) {
  return apiClient.request(previewPath(jobId, fileId, ""));
}

export function getPreviewWindow(jobId, fileId, { sheet = 0, fromRow, toRow } = {}) {
  const query = new URLSearchParams({
    sheet: String(sheet),
    from_row: String(fromRow),
    to_row: String(toRow),
  });
  return apiClient.request(`${previewPath(jobId, fileId, "/window")}?${query}`);
}

export function findPreviewArticle(jobId, fileId, { sheet = 0, query } = {}) {
  const params = new URLSearchParams({
    sheet: String(sheet),
    q: query,
  });
  return apiClient.request(`${previewPath(jobId, fileId, "/find")}?${params}`);
}

function previewPath(jobId, fileId, suffix) {
  return `/api/v1/jobs/${encodeURIComponent(jobId)}/files/${encodeURIComponent(fileId)}/preview${suffix}`;
}
