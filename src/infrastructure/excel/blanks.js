import { copyCellStyle, setTextCell, sheetBounds, sheetCellValue } from "./xml.js";
import { asText, normalizeHeader, parseNumber } from "./../../shared/text.js";
import { normalizeArticle } from "./../../domain/matching/identity.js";
import { articleNormalizeOptions, brandRule } from "./../../domain/brands/rules.js";
import { demandPiecesFromSupplierUnits, novacutanMinimumQuantity, novacutanSupplierUnitSize } from "./../../domain/order/quantities.js";
import { blankMatchers, detectColumns } from "./columns.js";
import { northNameKey } from "./../../domain/north/comments.js";
import { novacutanPositionKey } from "./../../domain/matching/novacutan.js";
import { budgetPrices, christinaProffLines } from "./pricing.js";

export function genericBlankPositions(blank, blankId, blankLabel, rule) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const blankArticleRaw = asText(sheetCellValue(blank.sheet, row, blank.columns.article));
    const blankArticle = normalizeArticle(blankArticleRaw, articleNormalizeOptions(rule));
    if (!blankArticle) continue;
    rows.push({
      key: `${blankId}:${row}`,
      blankId,
      blankLabel,
      blankRow: row,
      blankQuantityCol: blank.columns.quantity,
      blankArticleRaw,
      blankArticle,
      blankName: asText(sheetCellValue(blank.sheet, row, blank.columns.name)),
      blankUnit: asText(sheetCellValue(blank.sheet, row, blank.columns.unit)),
      blankBoxSize: rule.adjustment === "box" ? sheetCellValue(blank.sheet, row, blank.columns.boxSize) : rule.multiple,
    });
  }
  return rows;
}

export function duplicateArticleWarnings(positions) {
  const groups = new Map();
  for (const position of positions) {
    if (!position.blankArticle) continue;
    if (!groups.has(position.blankArticle)) groups.set(position.blankArticle, []);
    groups.get(position.blankArticle).push(position.blankRow);
  }
  return Array.from(groups.entries())
    .filter(([, rows]) => rows.length > 1)
    .map(([article, rows]) => ({ type: "blank_duplicate_article", article, rows }));
}

export function sourceItemsForMissingBlank(source) {
  return source.items
    .filter((item) => item.recommended > 0)
    .map((item) => ({
      sourceRow: item.rowIndex,
      sourceArticle: item.articleRaw,
      sourceName: item.name,
      recommended: item.recommended,
      rounded: item.rounded,
      stock: item.stock,
      inTransit: item.inTransit,
      hasOrderedFact: item.hasOrderedFact,
      orderedFact: item.hasOrderedFact ? item.orderedFact : null,
      sourceComment: item.sourceComment,
    }));
}

export function detectSothysVariantBlank(workbook) {
  let best = null;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 220); row += 1) {
      const blocks = [];
      for (let col = 1; col <= maxColumn - 4; col += 1) {
        const articleHeader = normalizeHeader(sheetCellValue(sheet, row, col));
        const volumeHeader = normalizeHeader(sheetCellValue(sheet, row, col + 1));
        const quantityHeader = normalizeHeader(sheetCellValue(sheet, row, col + 4));
        if ((articleHeader.includes("арт") || articleHeader.includes("артикул")) && (volumeHeader.includes("объем") || volumeHeader.includes("объм") || volumeHeader.includes("обьм")) && quantityHeader === "заказ") {
          blocks.push({
            article: col,
            volume: col + 1,
            unit: col + 2,
            quantity: col + 4,
          });
        }
      }
      if (blocks.length > (best?.blocks.length || 0)) {
        best = { sheet, sheetName: sheet.name, headerRow: row, blocks };
      }
      if (blocks.length >= 2) return { sheet, sheetName: sheet.name, headerRow: row, blocks };
    }
  }
  if (best?.blocks.length) return best;
  throw new Error("Не удалось найти блоки SOTHYS: Артикул, Объем, Заказ.");
}

export function sothysPositions(blank, blankId, blankLabel, rule) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  let lastEnglishName = "";
  let lastRussianName = "";
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const rowHasHeader = blank.blocks.some((block) => normalizeHeader(sheetCellValue(blank.sheet, row, block.article)).includes("арт"));
    if (rowHasHeader) continue;

    const englishName = asText(sheetCellValue(blank.sheet, row, 2));
    const russianName = asText(sheetCellValue(blank.sheet, row, 3));
    const hasAnyArticle = blank.blocks.some((block) => asText(sheetCellValue(blank.sheet, row, block.article)));
    if (hasAnyArticle && (englishName || russianName)) {
      lastEnglishName = englishName || lastEnglishName;
      lastRussianName = russianName || lastRussianName;
    }

    for (const block of blank.blocks) {
      const blankArticleRaw = asText(sheetCellValue(blank.sheet, row, block.article));
      const blankArticle = normalizeArticle(blankArticleRaw, articleNormalizeOptions(rule));
      if (!blankArticle) continue;
      const nameParts = [englishName || lastEnglishName, russianName || lastRussianName].filter(Boolean);
      const blankName = nameParts.join(" / ");
      const blankUnit = [asText(sheetCellValue(blank.sheet, row, block.volume)), asText(sheetCellValue(blank.sheet, row, block.unit))].filter(Boolean).join(" ");
      rows.push({
        key: `${blankId}:${row}:${block.quantity}`,
        blankId,
        blankLabel,
        blankRow: row,
        blankQuantityCol: block.quantity,
        blankArticleRaw,
        blankArticle,
        blankName,
        blankUnit,
        blankBoxSize: "",
      });
    }
  }
  return rows;
}

export function novacutanBlankPositions(blank, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const blankName = asText(sheetCellValue(blank.sheet, row, blank.columns.name));
    const normalized = normalizeHeader(blankName);
    if (!blankName || normalized === "описание" || normalized === "novacutan") continue;
    const minimumFromBlank = blank.columns.minimum ? parseNumber(sheetCellValue(blank.sheet, row, blank.columns.minimum)) : null;
    rows.push({
      key: `${blankId}:${row}`,
      blankId,
      blankLabel,
      blankRow: row,
      blankQuantityCol: blank.columns.quantity,
      blankArticleRaw: "",
      blankArticle: "",
      blankName,
      blankUnit: "шт",
      blankBoxSize: minimumFromBlank ?? novacutanMinimumQuantity(blankName),
      supplierUnitSize: novacutanSupplierUnitSize(blankName),
    });
  }
  return rows;
}

export function northBlankDetection(workbook, fileName = "", brand = "") {
  if (brand === "novacutan") {
    const novacutan = detectNovacutanBlank(workbook);
    if (!novacutan) throw new Error("Не удалось распознать бланк NOVACUTAN: не нашел описание товара и колонку количества.");
    return { kind: "novacutan", detection: novacutan };
  }

  if (brand === "klapp") {
    const klapp = detectKlappBlank(workbook);
    if (!klapp) throw new Error("Не удалось распознать бланк KLAPP: не нашел основной лист с колонками Артикул, Наименование и Заказ.");
    return { kind: "klapp", detection: klapp };
  }

  if (brand === "skin_synergy") {
    return { kind: "generic", detection: detectNameOnlyNorthBlank(workbook, { quantityHeader: "skinSynergy" }) };
  }

  if (brand === "sothys") {
    return { kind: "splitVariants", detection: detectSothysVariantBlank(workbook) };
  }

  const novacutan = detectNovacutanBlank(workbook);
  if (novacutan) return { kind: "novacutan", detection: novacutan };

  const klapp = detectKlappBlank(workbook);
  if (klapp) return { kind: "klapp", detection: klapp };

  const skinSynergy = detectSkinSynergyNorthBlank(workbook, fileName);
  if (skinSynergy) return { kind: "generic", detection: skinSynergy };

  try {
    return { kind: "generic", detection: detectColumns(workbook, "blank", { requireBox: false, quantityHeader: "anyOrder" }) };
  } catch (error) {
    try {
      return { kind: "generic", detection: detectNameOnlyNorthBlank(workbook) };
    } catch (nameOnlyError) {
      if (brand) throw error;
      return { kind: "splitVariants", detection: detectSothysVariantBlank(workbook) };
    }
  }
}

export function detectSkinSynergyNorthBlank(workbook, fileName = "", options = {}) {
  const signal = normalizeHeader(fileName);
  let hasSignal = Boolean(options.force) || signal.includes("skin synergy") || signal.includes("скин синерджи");
  for (const sheet of workbook.sheets) {
    if (hasSignal) break;
    if (normalizeHeader(sheet.name).includes("skin synergy") || normalizeHeader(sheet.name).includes("скин синерджи")) {
      hasSignal = true;
      break;
    }
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 60); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const text = normalizeHeader(sheetCellValue(sheet, row, col));
        if (text.includes("skin synergy") || text.includes("скин синерджи")) {
          hasSignal = true;
          break;
        }
      }
      if (hasSignal) break;
    }
  }
  if (!hasSignal) return null;

  try {
    return detectNameOnlyNorthBlank(workbook, { quantityHeader: "exactQuantity" });
  } catch {
    return detectNameOnlyNorthBlank(workbook);
  }
}

export function detectNameOnlyNorthBlank(workbook, options = {}) {
  const quantityHeader = options.quantityHeader === "skinSynergy" ? "exactQuantity" : options.quantityHeader || "anyOrder";
  try {
    return detectColumns(workbook, "blank", {
      requireArticle: false,
      requireBox: false,
      requireUnit: false,
      quantityHeader,
    });
  } catch (error) {
    if (quantityHeader !== "exactQuantity") throw error;
    return detectColumns(workbook, "blank", {
      requireArticle: false,
      requireBox: false,
      requireUnit: false,
      quantityHeader: "anyOrder",
    });
  }
}

export function detectKlappBlank(workbook) {
  const hasKlappSignal = workbook.sheets.some((sheet) => {
    if (normalizeHeader(sheet.name).includes("klapp")) return true;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 40); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        if (normalizeHeader(sheetCellValue(sheet, row, col)).includes("klapp")) return true;
      }
    }
    return false;
  });
  if (!hasKlappSignal) return null;

  const preferredSheets = [
    ...workbook.sheets.filter((sheet) => normalizeHeader(sheet.name).includes("бланк") && !normalizeHeader(sheet.name).includes("промо")),
    ...workbook.sheets.filter((sheet) => !normalizeHeader(sheet.name).includes("промо")),
  ];
  const seen = new Set();
  for (const sheet of preferredSheets) {
    if (seen.has(sheet.path)) continue;
    seen.add(sheet.path);
    try {
      return detectColumns({ sheets: [sheet] }, "blank", { requireBox: true, quantityHeader: "order" });
    } catch {
      try {
        return detectColumns({ sheets: [sheet] }, "blank", { requireBox: false, quantityHeader: "order" });
      } catch {
        // Проверяем следующий лист.
      }
    }
  }
  return null;
}

export function detectNovacutanBlank(workbook) {
  let hasNovacutanSignal = false;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    if (normalizeHeader(sheet.name).includes("novacutan")) hasNovacutanSignal = true;
    for (let row = 1; row <= Math.min(maxRow, 80); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        if (normalizeHeader(sheetCellValue(sheet, row, col)).includes("novacutan")) {
          hasNovacutanSignal = true;
        }
      }
    }
  }
  if (!hasNovacutanSignal) return null;

  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 120); row += 1) {
      let name = null;
      let price = null;
      let minimum = null;
      let quantity = null;
      for (let col = 1; col <= maxColumn; col += 1) {
        const header = normalizeHeader(sheetCellValue(sheet, row, col));
        if (header.includes("описание") || header.includes("товар") || header.includes("название")) name = col;
        else if (header.includes("цена")) price = col;
        else if (header.includes("заказ") && header.includes("от")) minimum = col;
        else if (header === "заказ" || header.includes("кол во") || header.includes("количество") || header.includes("qty")) quantity = col;
      }
      if (name && (quantity || minimum)) {
        return {
          sheet,
          sheetName: sheet.name,
          headerRow: row,
          columns: {
            name,
            price,
            minimum,
            quantity: quantity || Math.max(name, minimum || name) + 1,
          },
          quantityColumnCreated: !quantity,
        };
      }
    }
  }

  return null;
}

export function ensureNovacutanQuantityColumn(blank) {
  if (!blank.quantityColumnCreated) return;
  copyCellStyle(blank.sheet, blank.headerRow, blank.columns.minimum || blank.columns.name, blank.columns.quantity);
  setTextCell(blank.sheet, blank.headerRow, blank.columns.quantity, "Заказ");
}

export function prepareNovacutanQuantityCell(blank, row) {
  if (!blank.quantityColumnCreated) return;
  copyCellStyle(blank.sheet, row, blank.columns.minimum || blank.columns.name, blank.columns.quantity);
}

export function northGenericPositions(detection, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const unit = detection.columns.unit ? asText(sheetCellValue(detection.sheet, row, detection.columns.unit)) : "";
    const articleRaw = detection.columns.article ? asText(sheetCellValue(detection.sheet, row, detection.columns.article)) : "";
    const article = detection.columns.article ? normalizeArticle(articleRaw, { preserveHyphen: true }) : "";
    const key = article || northNameKey(name, unit);
    if (!key) continue;
    rows.push({
      key,
      blankId,
      blankLabel,
      row,
      quantityCol: detection.columns.quantity,
      articleCol: detection.columns.article,
      nameCol: detection.columns.name,
      unitCol: detection.columns.unit,
      articleRaw,
      name,
      unit,
      quantity: parseNumber(sheetCellValue(detection.sheet, row, detection.columns.quantity)) ?? 0,
    });
  }
  return rows;
}

export function northNovacutanPositions(detection, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const minimum = detection.columns.minimum ? parseNumber(sheetCellValue(detection.sheet, row, detection.columns.minimum)) : null;
    const supplierUnitSize = novacutanSupplierUnitSize(name);
    const quantity = demandPiecesFromSupplierUnits(parseNumber(sheetCellValue(detection.sheet, row, detection.columns.quantity)) ?? 0, supplierUnitSize);
    if (!name) continue;
    rows.push({
      key: novacutanPositionKey(name),
      blankId,
      blankLabel,
      row,
      quantityCol: detection.columns.quantity,
      articleCol: null,
      nameCol: detection.columns.name,
      unitCol: null,
      articleRaw: "",
      name,
      unit: "шт",
      quantity,
      novacutanMinimum: minimum ?? novacutanMinimumQuantity(name),
      supplierUnitSize,
    });
  }
  return rows;
}

export function northSplitVariantPositions(detection, blankId, blankLabel) {
  const rows = [];
  const rule = { preserveArticleHyphen: true, adjustment: "none" };
  for (const position of sothysPositions(detection, blankId, blankLabel, rule)) {
    const quantity = parseNumber(sheetCellValue(detection.sheet, position.blankRow, position.blankQuantityCol)) ?? 0;
    rows.push({
      key: position.blankArticle,
      blankId,
      blankLabel,
      row: position.blankRow,
      quantityCol: position.blankQuantityCol,
      articleCol: null,
      nameCol: null,
      unitCol: null,
      articleRaw: position.blankArticleRaw,
      name: position.blankName,
      unit: position.blankUnit,
      quantity,
    });
  }
  return rows;
}

export function workbookPriceOptions(workbook, fileName, brand) {
  const {detection,positions}=northPositions(workbook,'pricing','',fileName,brand);
  return [...new Map(positions.flatMap(p=>budgetPrices(detection,p.row,p.quantityCol)).map(p=>[p.id,p])).values()];
}

export function northPositions(workbook, blankId, blankLabel, fileName = "", brand = "") {
  const detected = northBlankDetection(workbook, fileName, brand);
  const positions = detected.kind === "novacutan"
    ? northNovacutanPositions(detected.detection, blankId, blankLabel)
    : detected.kind === "splitVariants"
      ? northSplitVariantPositions(detected.detection, blankId, blankLabel)
      : northGenericPositions(detected.detection, blankId, blankLabel);
  const boxMatcher = blankMatchers({ boxHeader: brandRule(brand).blankBoxHeader }).boxSize;
  const boxCol = Array.from({length:sheetBounds(detected.detection.sheet).maxColumn}, (_,i)=>i+1)
    .find(col=>boxMatcher(normalizeHeader(sheetCellValue(detected.detection.sheet,detected.detection.headerRow,col))));
  const lines = brand === 'christina' ? christinaProffLines(workbook, detected.detection.sheet) : new Map();
  for (const p of positions) {
    p.budgetPrices = budgetPrices(detected.detection, p.row, p.quantityCol);
    p.blankBoxSize = boxCol ? parseNumber(sheetCellValue(detected.detection.sheet,p.row,boxCol)) : null;
    p.christinaLine = lines.get(p.row);
    p.preserveBlankPrices = brand === 'christina';
  }
  return { ...detected, positions };
}
