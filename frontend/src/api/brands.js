import { apiClient } from "./client.js";

export function listBrandRules() {
  return apiClient.request("/api/v1/brand-rules");
}
