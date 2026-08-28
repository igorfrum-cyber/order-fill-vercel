const DEFAULT_API_BASE_URL = "http://127.0.0.1:8080";

export class ApiClient {
  constructor({ baseUrl = import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL, fetcher = fetch } = {}) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    this.fetcher = fetcher;
  }

  async request(path, options = {}) {
    const response = await this.fetcher(`${this.baseUrl}${path}`, {
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
