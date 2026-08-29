import { fileNameFromContentDisposition } from "../features/downloads/downloadLinks.js";

const DEFAULT_API_BASE_URL = "http://127.0.0.1:8080";

export class ApiClient {
  constructor({ baseUrl = apiBaseUrl(), fetcher = globalThis.fetch } = {}) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    this.fetcher = fetcher;
  }

  absoluteUrl(path) {
    if (!path) return "";
    if (/^https?:\/\//i.test(path)) return path;
    return `${this.baseUrl}${path.startsWith("/") ? path : `/${path}`}`;
  }

  async request(path, options = {}) {
    const response = await this.fetcher.call(globalThis, `${this.baseUrl}${path}`, {
      ...options,
      headers: {
        ...(options.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
        ...(options.headers || {}),
      },
    });
    if (!response.ok) {
      throw new ApiError(response.status, await parseError(response));
    }
    return parseResponse(response);
  }

  async requestDownload(path) {
    const response = await this.fetcher.call(globalThis, `${this.baseUrl}${path}`);
    if (!response.ok) {
      throw new ApiError(response.status, await parseError(response));
    }
    const contentType = response.headers.get("Content-Type") || "application/octet-stream";
    const fileName = fileNameFromContentDisposition(response.headers.get("Content-Disposition"));
    return {
      blob: await response.blob(),
      fileName,
      contentType,
    };
  }
}

function apiBaseUrl() {
  return import.meta.env?.VITE_API_BASE_URL || DEFAULT_API_BASE_URL;
}

export class ApiError extends Error {
  constructor(status, payload) {
    super(payload?.message || `API request failed with status ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

async function parseResponse(response) {
  if (response.status === 204) return null;
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) return response.json();
  return response.blob();
}

async function parseError(response) {
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) return response.json();
  return { code: "http_error", message: await response.text() };
}

export const apiClient = new ApiClient();
