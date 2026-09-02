export function remoteDownloadLinksHtml(files) {
  return files
    .map((file) => {
      const href = file.downloadUrl || file.download_url || "";
      const label = file.label || file.name || "Скачать файл";
      return `<a class="file-link" href="${escapeHtml(href)}" download="${escapeHtml(file.name)}">${escapeHtml(label)}</a>`;
    })
    .join("");
}

export function fileNameFromContentDisposition(header) {
  const value = String(header || "");
  const encoded = /filename\*=UTF-8''([^;]+)/i.exec(value);
  if (encoded) {
    try {
      return decodeURIComponent(encoded[1]);
    } catch {
      return encoded[1];
    }
  }
  const quoted = /filename="([^"]+)"/i.exec(value);
  if (quoted) return quoted[1];
  const plain = /filename=([^;]+)/i.exec(value);
  return plain ? plain[1].trim() : "";
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
