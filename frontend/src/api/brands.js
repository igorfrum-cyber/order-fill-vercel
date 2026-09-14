import { apiClient } from "./client.js";

export function listBrandRules() {
  return apiClient.request("/api/v1/brand-rules");
}

export function updateBrandRule(brand, rule) {
  return apiClient.request(`/api/v1/brand-rules/${encodeURIComponent(brand)}`, {
    method: "POST",
    body: JSON.stringify(rule),
  });
}
