export function remoteDownloadLinksHtml(files) {
  return files
    .map((file) => {
      const href = file.download_url || file.downloadUrl || "";
      const label = file.label || file.name || "Скачать файл";
      return `<a class="file-link" href="${escapeHtml(href)}" download="${escapeHtml(file.name)}">${escapeHtml(label)}</a>`;
    })
    .join("");
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
