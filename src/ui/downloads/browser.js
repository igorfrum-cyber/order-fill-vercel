

export function createDownloadsBrowser(context) {
  const {elements, state, actions} = context;
function clearDownloadLinks() {
  for (const url of state.currentDownloadUrls) URL.revokeObjectURL(url);
  state.currentDownloadUrls = [];
  elements.downloadLinks.classList.add("hidden");
  elements.downloadLinks.innerHTML = "";
}

function clearNorthDownloadLinks() {
  for (const url of state.currentNorthDownloadUrls) URL.revokeObjectURL(url);
  state.currentNorthDownloadUrls = [];
  elements.northDownloadLinks.classList.add("hidden");
  elements.northDownloadLinks.innerHTML = "";
}

function triggerDownload(url, fileName) {
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

function downloadIssueReport() {
  const rows = actions.issueReportRows();
  if (!rows.length) {
    alert("Нет спорных строк для отчета.");
    return;
  }
  const blob = new Blob([`\ufeff${actions.issueReportCsv()}`], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  triggerDownload(url, "отчет для исправления в 1С.csv");
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function prepareDownloadLinks(files) {
  clearDownloadLinks();
  state.currentDownloadUrls = files.map((file) => URL.createObjectURL(file.blob));

  elements.downloadLinks.innerHTML = files
    .map((file, index) => `<a class="file-link" href="${state.currentDownloadUrls[index]}" download="${actions.escapeHtml(file.name)}">${actions.escapeHtml(file.label)}</a>`)
    .join("");
  elements.downloadLinks.classList.remove("hidden");

  state.currentDownloadUrls.forEach((url, index) => {
    window.setTimeout(() => triggerDownload(url, files[index].name), index * 250);
  });
}

function prepareNorthDownloadLinks(files) {
  clearNorthDownloadLinks();
  state.currentNorthDownloadUrls = files.map((file) => URL.createObjectURL(file.blob));
  elements.northDownloadLinks.innerHTML = files
    .map((file, index) => `<a class="file-link" href="${state.currentNorthDownloadUrls[index]}" download="${actions.escapeHtml(file.name)}">${actions.escapeHtml(file.label)}</a>`)
    .join("");
  elements.northDownloadLinks.classList.remove("hidden");
  state.currentNorthDownloadUrls.forEach((url, index) => {
    window.setTimeout(() => triggerDownload(url, files[index].name), index * 250);
  });
}
  return { clearDownloadLinks, clearNorthDownloadLinks, triggerDownload, downloadIssueReport, prepareDownloadLinks, prepareNorthDownloadLinks };
}
