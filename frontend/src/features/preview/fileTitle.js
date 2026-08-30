export function previewFileTitle(file) {
  const label = String(file?.label || "").toLowerCase();
  if (label.includes("бланк")) return "Бланк";
  if (label.includes("таблиц")) return "Таблица 1С";
  const name = String(file?.name || "").replace(/\.[^.]+$/, "");
  return name || file?.label || "Файл";
}
