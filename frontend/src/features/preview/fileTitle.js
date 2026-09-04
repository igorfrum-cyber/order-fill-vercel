export function previewFileTitle(file) {
  const label = String(file?.label || "").toLowerCase();
  const name = String(file?.name || "").toLowerCase();
  if (label.includes("бланк") || name.includes("бланк")) return "Бланк";
  if (label.includes("таблиц") || name.includes("таблиц")) return "Таблица 1С";
  const shortName = String(file?.name || "").replace(/\.[^.]+$/, "");
  return shortName || file?.label || "Файл";
}
