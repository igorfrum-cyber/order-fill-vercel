import { apiClient } from "./client.js";

export function getInboundSettings() {
  return apiClient.request("/api/v1/inbound/settings");
}

export function updateInboundSettings({ enabled, receive_address: receiveAddress }) {
  return apiClient.request("/api/v1/inbound/settings", {
    method: "POST",
    body: JSON.stringify({ enabled, receive_address: receiveAddress || "" }),
  });
}

export function getInboundCompany(companyId) {
  return apiClient.request(`/api/v1/inbound/companies/${encodeURIComponent(companyId)}`);
}

export function updateInboundCompany(companyId, { receive_address: receiveAddress, sender_email: senderEmail, enabled }) {
  return apiClient.request(`/api/v1/inbound/companies/${encodeURIComponent(companyId)}`, {
    method: "POST",
    body: JSON.stringify({
      receive_address: receiveAddress || "",
      sender_email: senderEmail || "",
      enabled: Boolean(enabled),
    }),
  });
}

export function getInboundMessages(companyId) {
  return apiClient.request(`/api/v1/inbound/companies/${encodeURIComponent(companyId)}/messages`);
}

export function getInboundDeliveries() {
  return apiClient.request("/api/v1/inbound/deliveries");
}

export function downloadInboundFile(companyId, messageId, attachmentId) {
  return apiClient.requestDownload(`/api/v1/inbound/companies/${encodeURIComponent(companyId)}/messages/${encodeURIComponent(messageId)}/files/${encodeURIComponent(attachmentId)}`);
}