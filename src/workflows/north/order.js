import { budgetNorthWorkbookPricing, transferFilesFromTotals, writeNorthSummaryWorkbook } from "./export.js";
import { novacutanNorthOrderTable } from "./../../domain/north/comments.js";
import { actualQuantityByKey, addNorthTotal, buildNorthPlan, editedNorthTotals, recalculateEditedNorthPlan, scopedNorthPositions } from "./plan.js";
import { outputExtension } from "./../../shared/filenames.js";
import { detectNorthCity, validateNorthTyumenSourceWorkbook } from "./../../infrastructure/excel/cities.js";
import { northPositions } from "./../../infrastructure/excel/blanks.js";
import { northAvailabilityForTotal, readNorthTyumenAvailability } from "./availability.js";

export function finalizeNorthOrderFiles(result, edits = [], options = {}) {
  const editsByKey = new Map(edits.map((edit) => [edit.key, edit]));
  const totals = editedNorthTotals(result, editsByKey);
  const planRows = recalculateEditedNorthPlan(result, totals);
  const actualByKey = actualQuantityByKey(planRows, editsByKey, options);
  const summaries = result.summaries?.length ? result.summaries : [result.summary];
  const summaryFiles = summaries.map((summary) => {
    const write = writeNorthSummaryWorkbook(summary, totals, actualByKey);
    return {
      workbook: budgetNorthWorkbookPricing(summary,result.budgetPricing),
      fileName: summary.summaryFileName || result.summaryFileName,
      label: summary.summaryLabel || "общий бланк",
      appended: write.appended,
      adjustedCount: write.adjustedCount,
    };
  });
  const adjustedCount = planRows.filter((row) => {
    const actual = Number(actualByKey.get(row.key) || 0);
    return actual > 0 && actual !== row.supplierNeed;
  }).length;

  return {
    summaryWorkbook: summaryFiles[0]?.workbook || result.summary.workbook,
    summaryFileName: summaryFiles[0]?.fileName || result.summaryFileName,
    summaryFiles,
    uploadedCities: result.uploadedCities,
    appendedToSummary: summaryFiles.flatMap((file) => file.appended),
    adjustedToMinimum: adjustedCount,
    totalsCount: Array.from(actualByKey.values()).filter((quantity) => quantity > 0).length,
    orderTable: novacutanNorthOrderTable(result.summary, planRows, actualByKey),
    transfers: transferFilesFromTotals(totals, result.transferCities),
  };
}

export function buildNorthOrderFiles(blanks, options = {}) {
  if (!Array.isArray(blanks) || !blanks.length) throw new Error("Загрузите хотя бы один бланк для раздела «Север».");
  if (options.tyumenSourceWorkbook) {
    validateNorthTyumenSourceWorkbook(options.tyumenSourceWorkbook, options.tyumenSourceFileName || "");
  }

  const northBrand = options.brand || "";
  const cityKeys = new Set();
  const prepared = blanks.map((blank, index) => {
    const city = detectNorthCity(blank.workbook, blank.fileName);
    const variant = blank.variant || "";
    const duplicateKey = `${city.key}:${variant || "default"}`;
    const duplicateLabel = variant ? `${city.label} ${blank.variantLabel || variant}` : city.label;
    if (cityKeys.has(duplicateKey)) throw new Error(`Загружено несколько бланков ${duplicateLabel}. Оставьте один файл на город и тип бланка.`);
    cityKeys.add(duplicateKey);
    const extracted = northPositions(blank.workbook, `north-${index}`, variant ? `${city.label} ${blank.variantLabel || variant}` : city.label, blank.fileName, northBrand);
    const positions = scopedNorthPositions(extracted.positions, variant, blank.variantLabel || "");
    return { ...blank, city, ...extracted, positions, brand:northBrand, variant, variantLabel: blank.variantLabel || "" };
  });

  const totals = new Map();
  for (const file of prepared) {
    for (const item of file.positions) addNorthTotal(totals, item, file.city);
  }

  const hasVariants = prepared.some((file) => file.variant);
  const summaries = hasVariants
    ? Array.from(new Map(prepared.map((file) => [file.variant, file])).entries())
      .map(([variant, firstFile]) => {
        const group = prepared.filter((file) => file.variant === variant);
        const summary = group.find((file) => file.city.key === "tyumen") || firstFile;
        return {
          ...summary,
          positions: summary.positions,
          summaryFileName: `Север общий бланк ${summary.variantLabel || variant}.${outputExtension(summary.fileName)}`,
          summaryLabel: `общий бланк ${summary.variantLabel || variant}`,
        };
      })
    : [];
  const summary = summaries[0] || prepared.find((file) => file.city.key === "tyumen") || prepared[0];
  if (!summary.summaryFileName) summary.summaryFileName = `Север общий бланк.${outputExtension(summary.fileName)}`;
  if (!summary.summaryLabel) summary.summaryLabel = "общий бланк";
  // Zero-demand items may participate in budget top-up, but never create city transfers.
  const allPositions = prepared.flatMap(file => file.positions);
  if (options.tyumenSourceWorkbook) {
    const available = readNorthTyumenAvailability(options.tyumenSourceWorkbook, summary.kind);
    for (const p of allPositions) {
      if (totals.has(p.key) || (!northAvailabilityForTotal(available, p, summary.kind)
        && !(p.variant === 'proff' && p.christinaLine))) continue;
      totals.set(p.key, { ...p, totalQuantity: 0, cities: new Map([['tyumen', 0]]) });
    }
  }
  const planRows = buildNorthPlan({ ...summary, positions: allPositions }, totals, options.tyumenSourceWorkbook || null);
  const transferCityMap = new Map(prepared.filter((file) => file.city.key !== "tyumen").map((file) => [file.city.key, file.city]));
  const transferCities = Array.from(transferCityMap.values());
  const confirmationGroups = Array.from(new Map(prepared.map((file) => [file.city.key, file.city])).values()).map((city) => ({
    city,
    variants: prepared
      .filter((file) => file.city.key === city.key)
      .map((file) => file.variantLabel || file.fileName),
  }));

  return {
    summary,
    summaries: summaries.length ? summaries : [summary],
    totals,
    summaryFileName: summary.summaryFileName,
    uploadedCities: Array.from(new Set(prepared.map((file) => file.city.label))),
    confirmationGroups,
    planRows,
    hasTyumenSource: Boolean(options.tyumenSourceWorkbook),
    transferCities,
    transfers: transferFilesFromTotals(totals, transferCities),
  };
}
