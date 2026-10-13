import { detectColumns } from "./../../infrastructure/excel/columns.js";
import { readSource } from "./read-source.js";
import { blankDetectionOptions, brandRule } from "./../../domain/brands/rules.js";
import { buildSourceContext, candidatesForArticle, duplicateCandidatesForReport } from "./../../domain/matching/source-index.js";
import { christinaProffLines } from "./../../infrastructure/excel/pricing.js";
import { novacutanOrderForItem, orderForItem } from "./../../domain/order/quantities.js";
import { chooseCandidate, chooseNameFallback, chooseSothysNameFallback } from "./../../domain/matching/identity.js";
import { detectKlappBlank, detectNovacutanBlank, detectSothysVariantBlank, duplicateArticleWarnings, ensureNovacutanQuantityColumn, genericBlankPositions, novacutanBlankPositions, prepareNovacutanQuantityCell, sothysPositions, sourceItemsForMissingBlank } from "./../../infrastructure/excel/blanks.js";
import { setNumericCell } from "./../../infrastructure/excel/xml.js";
import { chooseNovacutanCandidate } from "./../../domain/matching/novacutan.js";

export function fillWorkbook({ sourceWorkbook, sourceFileName = "", blankWorkbook, orderMonth, brand = "angiopharm", blankId = "blank", blankLabel = "" }) {
  const rule = brandRule(brand);
  const source = readSource(sourceWorkbook, orderMonth, rule, sourceFileName);
  const sourceContext = buildSourceContext(source, rule);
  if (rule.blankLayout === "novacutan") {
    return fillNovacutanWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel });
  }
  if (rule.blankLayout === "splitVariants") {
    return fillSplitVariantWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel });
  }
  const blank = rule.label === "KLAPP"
    ? detectKlappBlank(blankWorkbook)
    : detectColumns(blankWorkbook, "blank", blankDetectionOptions(rule));
  if (!blank) throw new Error("Не удалось распознать бланк KLAPP: не нашел основной лист с колонками Артикул, Наименование и Заказ.");
  const blankPositions = genericBlankPositions(blank, blankId, blankLabel, rule);
  const blankWarnings = duplicateArticleWarnings(blankPositions);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;
  for (const rowInfo of blankPositions) {
    const row = rowInfo.blankRow;
    const { blankArticle, blankName, blankUnit, blankBoxSize } = rowInfo;
    let selected;
    let score;
    let status;
    let order;
    const candidates = candidatesForArticle(sourceContext, blankArticle, rule);
    if (!candidates.length) {
      const fallback = chooseNameFallback(sourceContext.noArticleItems, blankName, blankUnit);
      if (!fallback.item) {
        unmatched += 1;
        setNumericCell(blank.sheet, row, blank.columns.quantity, null);
        reportRows.push(makeUnmatchedReportRow(rowInfo, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      selected = fallback.item;
      score = fallback.score;
      if (selected.rounded > 0) {
        suspicious += 1;
        order = orderForItem(selected, rule, blankBoxSize);
        setNumericCell(blank.sheet, row, blank.columns.quantity, null);
        reportRows.push(makeReportRow("warning_name_only", rowInfo, selected, score, { ...order, inserted: null, autoComment: "" }, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      status = "matched_by_name";
    } else {
      const isDuplicate = candidates.length > 1;
      if (isDuplicate) duplicates += 1;
      rowInfo.duplicateCandidates = isDuplicate ? duplicateCandidatesForReport(candidates) : [];
      const candidate = chooseCandidate(candidates, blankName, blankUnit);
      selected = candidate.item;
      score = candidate.score;
      status = "matched";
      if (score < 0.32) {
        status = "warning_name_differs";
        suspicious += 1;
      }
    }
    order = orderForItem(selected, rule, blankBoxSize);
    rowInfo.duplicate = candidates.length > 1;
    if (order.inserted == null) {
      setNumericCell(blank.sheet, row, blank.columns.quantity, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      setNumericCell(blank.sheet, row, blank.columns.quantity, order.inserted);
      filled += 1;
    }
    reportRows.push(makeReportRow(status, rowInfo, selected, score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }
  if (brand === 'christina' && blankId === 'proff') {
    const lines = christinaProffLines(blankWorkbook, blank.sheet);
    for (const row of reportRows) if (row.status === 'not_in_source' && lines.has(row.blankRow)) {
      // No invented sales: only an explicitly allowed first-set completion or
      // a manager edit may order these unmatched blank positions.
      row.editable = true;
      row.budget = { demand: 0, delivery: 0, category: null };
    }
  }
  return {
    blankId,
    blankLabel,
    sourceWorkbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings,
      blankDuplicateArticles: blankWarnings.length,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

export function fillNovacutanWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel }) {
  const blank = detectNovacutanBlank(blankWorkbook);
  if (!blank) throw new Error("Не удалось распознать бланк NOVACUTAN: не нашел описание товара и колонку «кол-во» или «Заказ от».");
  ensureNovacutanQuantityColumn(blank);
  const blankPositions = novacutanBlankPositions(blank, blankId, blankLabel);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;

  for (const rowInfo of blankPositions) {
    const candidate = chooseNovacutanCandidate(source.items, rowInfo.blankName, rowInfo.blankUnit);
    if (!candidate?.item || candidate.score < 0.72) {
      unmatched += 1;
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, null);
      reportRows.push(makeUnmatchedReportRow(rowInfo, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
      continue;
    }

    const order = novacutanOrderForItem(candidate.item, rowInfo, rule);
    let status = "matched_by_name";
    if (candidate.score < 0.85) {
      status = "warning_name_differs";
      suspicious += 1;
    }

    if (order.inserted == null) {
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, order.inserted);
      filled += 1;
    }

    reportRows.push(makeReportRow(status, rowInfo, candidate.item, candidate.score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }

  return {
    blankId,
    blankLabel,
    sourceWorkbook: source.workbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings: [],
      blankDuplicateArticles: 0,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

export function fillSplitVariantWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel }) {
  const blank = detectSothysVariantBlank(blankWorkbook);
  const blankPositions = sothysPositions(blank, blankId, blankLabel, rule);
  const blankWarnings = duplicateArticleWarnings(blankPositions);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;

  for (const position of blankPositions) {
    let selected;
    let score;
    let status;
    let order;
    const candidates = candidatesForArticle(sourceContext, position.blankArticle, rule);
    if (!candidates.length) {
      const fallback = chooseSothysNameFallback(sourceContext.noArticleItems, position.blankName, position.blankUnit);
      if (!fallback.item) {
        unmatched += 1;
        setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
        reportRows.push(makeUnmatchedReportRow(position, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      selected = fallback.item;
      score = fallback.score;
      if (selected.rounded > 0) {
        suspicious += 1;
        order = orderForItem(selected, rule, position.blankBoxSize);
        setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
        reportRows.push(makeReportRow("warning_name_only", position, selected, score, { ...order, inserted: null, autoComment: "" }, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      status = "matched_by_name";
    } else {
      const isDuplicate = candidates.length > 1;
      if (isDuplicate) duplicates += 1;
      position.duplicateCandidates = isDuplicate ? duplicateCandidatesForReport(candidates) : [];
      const candidate = chooseCandidate(candidates, position.blankName, position.blankUnit);
      selected = candidate.item;
      score = candidate.score;
      status = "matched";
      if (score < 0.32) {
        status = "warning_name_differs";
        suspicious += 1;
      }
    }

    order = orderForItem(selected, rule, position.blankBoxSize);
    position.duplicate = candidates.length > 1;
    if (order.inserted == null) {
      setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, order.inserted);
      filled += 1;
    }
    reportRows.push(makeReportRow(status, position, selected, score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }

  return {
    blankId,
    blankLabel,
    sourceWorkbook: source.workbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings,
      blankDuplicateArticles: blankWarnings.length,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

export function makeUnmatchedReportRow(rowInfo, context) {
  return {
    status: "not_in_source",
    blankId: context.blankId,
    blankLabel: context.blankLabel,
    adjustmentLabel: context.adjustmentLabel,
    key: rowInfo.key || `${context.blankId}:${rowInfo.blankRow}`,
    blankRow: rowInfo.blankRow,
    blankQuantityCol: rowInfo.blankQuantityCol,
    blankArticle: rowInfo.blankArticleRaw,
    blankName: rowInfo.blankName,
    blankUnit: rowInfo.blankUnit,
    blankBoxSize: rowInfo.blankBoxSize,
    sourceRow: null,
    sourceArticle: "",
    sourceName: "",
    hasOrderedFact: false,
    orderedFact: null,
    sourceComment: "",
    stock: "",
    inTransit: "",
    recommended: null,
    rounded: null,
    baseRounded: null,
    inserted: null,
    autoComment: "",
    boxAdjusted: false,
    duplicate: Boolean(rowInfo.duplicate),
    duplicateCandidates: rowInfo.duplicateCandidates || [],
    editable: false,
    similarity: 0,
  };
}

export function makeReportRow(status, rowInfo, selected, score, order, context) {
  return {
    budget: selected.budget,
    status,
    blankId: context.blankId,
    blankLabel: context.blankLabel,
    adjustmentLabel: context.adjustmentLabel,
    key: rowInfo.key || `${context.blankId}:${rowInfo.blankRow}`,
    blankRow: rowInfo.blankRow,
    blankQuantityCol: rowInfo.blankQuantityCol,
    blankArticle: rowInfo.blankArticleRaw,
    blankName: rowInfo.blankName,
    blankUnit: rowInfo.blankUnit,
    blankBoxSize: rowInfo.blankBoxSize,
    sourceRow: selected.rowIndex,
    sourceArticle: selected.articleRaw,
    sourceName: selected.name,
    hasOrderedFact: selected.hasOrderedFact,
    orderedFact: selected.hasOrderedFact ? selected.orderedFact : null,
    sourceComment: selected.sourceComment,
    stock: selected.stock,
    inTransit: selected.inTransit,
    recommended: selected.recommended,
    rounded: selected.rounded,
    baseRounded: order.rounded,
    inserted: order.inserted,
    autoComment: order.autoComment,
    boxAdjusted: order.boxAdjusted,
    duplicate: Boolean(rowInfo.duplicate),
    duplicateCandidates: rowInfo.duplicateCandidates || [],
    editable: true,
    similarity: Number(score.toFixed(4)),
  };
}
