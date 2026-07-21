import fs from "node:fs/promises";
import path from "node:path";
import { strFromU8, unzipSync } from "fflate";
import { read as readSpreadsheet, write as writeSpreadsheet } from "xlsx";

import { adjustQuantityForBrand, applyFinalEdits, detectColumns, fillWorkbook, loadXlsx, normalizeArticle, saveXlsx } from "../src/workbookProcessor.js";

const sourcePath = "/Users/igorfrumes/Downloads/агио артикул.xlsx";
const blankPath = "/Users/igorfrumes/Downloads/2026 06 23 Бланк заказа ANGIOPHARM (1).xlsx";
const homeBlankPath = "/Users/igorfrumes/Downloads/Актуальный бланк HOME 17.06.2026.xlsx";
const proffBlankPath = "/Users/igorfrumes/Downloads/Актуальный_бланк_PROFF_1 от 17.06.26.xlsx";
const levissimeBlankPath = "/private/tmp/03.07.2026 LeviSsime БЛАНК ЗАКАЗА.xlsx";
const levissimeLegacyBlankPath = "/Users/igorfrumes/Downloads/03.07.2026 LeviSsime БЛАНК ЗАКАЗА.XLS";
const levissimeCurrentSourcePath = "/Users/igorfrumes/Downloads/lev.xlsx";
const levissimeCurrentBlankPath = "/Users/igorfrumes/Downloads/03.07.2026 LeviSsime БЛАНК ЗАКАЗА (2).XLS";
const sothysLegacyBlankPath = "/Users/igorfrumes/Downloads/Сотис бланк (1).xls";
const urengoySourcePath = "/Users/igorfrumes/Downloads/уренгой ангио.xlsx";
const currentAngioSourcePath = "/Users/igorfrumes/Downloads/ангио заполненная таблица.xlsx";
const currentAngioFilledBlankPath = "/Users/igorfrumes/Downloads/2026 07 07 Бланк заказа ANGIOPHARM 6 заполненный (1).xlsx";
const blankOutputPath = path.resolve("test-output/browser-filled-blank.xlsx");
const sourceOutputPath = path.resolve("test-output/browser-filled-source.xlsx");

function convertLegacyXlsToXlsx(buffer) {
  const workbook = readSpreadsheet(buffer, {
    type: "buffer",
    cellFormula: true,
    cellStyles: true,
    cellDates: false,
  });
  return writeSpreadsheet(workbook, {
    bookType: "xlsx",
    type: "buffer",
  });
}

const [sourceWorkbook, blankWorkbook] = await Promise.all([
  fs.readFile(sourcePath).then((buffer) => loadXlsx(buffer)),
  fs.readFile(blankPath).then((buffer) => loadXlsx(buffer)),
]);

const [homeBlankWorkbook, proffBlankWorkbook] = await Promise.all([
  fs.readFile(homeBlankPath).then((buffer) => loadXlsx(buffer)),
  fs.readFile(proffBlankPath).then((buffer) => loadXlsx(buffer)),
]);

const homeDetection = detectColumns(homeBlankWorkbook, "blank", { requireBox: false });
const proffDetection = detectColumns(proffBlankWorkbook, "blank", { requireBox: false });
if (homeDetection.headerRow !== 7 || homeDetection.columns.quantity !== 5) throw new Error("HOME blank columns were not detected.");
if (proffDetection.headerRow !== 7 || proffDetection.columns.quantity !== 5) throw new Error("PROFF blank columns were not detected.");

const christinaUp = adjustQuantityForBrand(8, "christina");
if (christinaUp.inserted !== 9 || christinaUp.autoComment !== "до кратности 3") throw new Error("Christina should round up to a multiple of 3 first.");
const christinaNoAdjust = adjustQuantityForBrand(13, "christina");
if (christinaNoAdjust.inserted !== 13 || christinaNoAdjust.autoComment !== "") throw new Error("Christina should keep the value when neither multiple direction fits thresholds.");
const christinaSmall = adjustQuantityForBrand(1, "christina");
if (christinaSmall.inserted !== null) throw new Error("Christina should skip recommendations below 1.5.");

try {
  await fs.access(levissimeLegacyBlankPath);
  const levissimeLegacyWorkbook = loadXlsx(convertLegacyXlsToXlsx(await fs.readFile(levissimeLegacyBlankPath)));
  const levissimeLegacyDetection = detectColumns(levissimeLegacyWorkbook, "blank", {
    requireBox: true,
    quantityHeader: "order",
    boxHeader: "packageQuantity",
  });
  if (levissimeLegacyDetection.headerRow !== 29 || levissimeLegacyDetection.columns.quantity !== 7 || levissimeLegacyDetection.columns.boxSize !== 4) {
    throw new Error("Legacy XLS LeviSsime blank should be converted and detected.");
  }
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("Legacy LeviSsime XLS fixture is not available; skipping XLS conversion test.");
}

try {
  await fs.access(levissimeBlankPath);
  const [levissimeSourceWorkbook, levissimeBlankWorkbook] = await Promise.all([
    fs.readFile(sourcePath).then((buffer) => loadXlsx(buffer)),
    fs.readFile(levissimeBlankPath).then((buffer) => loadXlsx(buffer)),
  ]);
  const levissimeDetection = detectColumns(levissimeBlankWorkbook, "blank", {
    requireBox: true,
    quantityHeader: "order",
    boxHeader: "packageQuantity",
  });
  if (levissimeDetection.headerRow !== 29 || levissimeDetection.columns.quantity !== 7 || levissimeDetection.columns.boxSize !== 4) {
    throw new Error("LeviSsime blank columns were not detected.");
  }
  const levissimeSourceDetection = detectColumns(levissimeSourceWorkbook, "source");
  const sourceRow = levissimeSourceDetection.headerRow + 1;
  levissimeSourceDetection.sheet.cells.get(`${sourceRow}:${levissimeSourceDetection.columns.article}`).value = "МТ4532";
  levissimeSourceDetection.sheet.cells.get(`${sourceRow}:${levissimeSourceDetection.columns.name}`).value = "Крем для снятия макияжа Aqua Cleanser";
  levissimeSourceDetection.sheet.cells.get(`${sourceRow}:${levissimeSourceDetection.columns.recommended}`).value = 11;
  levissimeSourceDetection.sheet.cells.get(`${sourceRow}:${levissimeSourceDetection.columns.orderedFact}`).value = "";
  levissimeSourceDetection.sheet.cells.get(`${sourceRow}:${levissimeSourceDetection.columns.comment}`).value = "";

  const levissimeResult = fillWorkbook({
    sourceWorkbook: levissimeSourceWorkbook,
    blankWorkbook: levissimeBlankWorkbook,
    orderMonth: "2026-07",
    brand: "levissime",
  });
  const levissimeRow = levissimeResult.reportRows.find((row) => row.blankArticle === "4532");
  if (!levissimeRow || levissimeRow.inserted !== 12 || levissimeRow.autoComment !== "до коробки") {
    throw new Error("LeviSsime should match MT-prefixed articles and round to package quantity.");
  }
  const levissimeSheet = levissimeResult.blankWorkbook.sheets.find((item) => item.name === "Заказ");
  if (levissimeSheet.cells.get("44:7")?.value !== 12) throw new Error("LeviSsime order quantity should be written to column G.");
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("LeviSsime fixture is not available; skipping LeviSsime workbook test.");
}

try {
  await Promise.all([fs.access(levissimeCurrentSourcePath), fs.access(levissimeCurrentBlankPath)]);
  const [levissimeSourceWorkbook, levissimeBlankWorkbook] = await Promise.all([
    fs.readFile(levissimeCurrentSourcePath).then((buffer) => loadXlsx(buffer)),
    fs.readFile(levissimeCurrentBlankPath).then((buffer) => loadXlsx(convertLegacyXlsToXlsx(buffer))),
  ]);
  const levissimeResult = fillWorkbook({
    sourceWorkbook: levissimeSourceWorkbook,
    blankWorkbook: levissimeBlankWorkbook,
    orderMonth: "2026-07",
    brand: "levissime",
  });
  const serumRow = levissimeResult.reportRows.find((row) => row.blankArticle === "4543");
  if (!serumRow || serumRow.status === "not_in_source" || serumRow.sourceArticle !== "4543" || serumRow.inserted !== null) {
    throw new Error("LeviSsime item 4543 should be found in the source table and left blank when recommendation is empty.");
  }
  const rescueRow = levissimeResult.reportRows.find((row) => row.blankArticle === "4734");
  if (!rescueRow || rescueRow.status !== "not_in_source") {
    throw new Error("LeviSsime item 4734 should still be marked missing when it is absent from the source table.");
  }
  const currentDuplicateSignatures = new Map((levissimeResult.sourceDuplicateGroups || []).map((group) => [
    group.article,
    group.candidates.map((item) => item.sourceRow).sort((left, right) => left - right).join(":"),
  ]));
  if (currentDuplicateSignatures.get("5614") !== "51:83") {
    throw new Error("LeviSsime source duplicate 5614 should include source rows 51 and 83.");
  }
  if (currentDuplicateSignatures.get("5410") !== "129:160") {
    throw new Error("LeviSsime source duplicate 5410 should include source rows 129 and 160.");
  }
  levissimeBlankWorkbook.sheets[0].cells.get("69:1").value = "MTLS8017";
  levissimeBlankWorkbook.sheets[0].cells.get("69:2").value = "Lift total row glitch";
  const totalGlitchResult = fillWorkbook({
    sourceWorkbook: levissimeSourceWorkbook,
    blankWorkbook: levissimeBlankWorkbook,
    orderMonth: "2026-07",
    brand: "levissime",
  });
  const totalGlitchRow = totalGlitchResult.reportRows.find((row) => row.blankArticle === "MTLS8017");
  if (!totalGlitchRow || totalGlitchRow.sourceRow === 15 || totalGlitchRow.duplicateCandidates.some((item) => item.sourceRow === 15)) {
    throw new Error("Source total rows should be ignored even when the right-side product block is accidentally filled.");
  }
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("Current LeviSsime fixture is not available; skipping current LeviSsime matching test.");
}

try {
  await fs.access(sothysLegacyBlankPath);
  const [sothysSourceWorkbook, sothysBlankWorkbook] = await Promise.all([
    fs.readFile(sourcePath).then((buffer) => loadXlsx(buffer)),
    fs.readFile(sothysLegacyBlankPath).then((buffer) => loadXlsx(convertLegacyXlsToXlsx(buffer))),
  ]);
  const sothysSourceDetection = detectColumns(sothysSourceWorkbook, "source");
  const sothysFixtures = [
    ["160172", "Vitality Cleansing Milk 200 мл", 2.4],
    ["360173", "Vitality Cleansing Milk 500 мл", 4.5],
    ["350532-1", "Mask-Scrub Black tea 180 мл", 1.4],
  ];
  for (const [index, [article, name, recommended]] of sothysFixtures.entries()) {
    const sourceRow = sothysSourceDetection.headerRow + 2 + index;
    sothysSourceDetection.sheet.cells.get(`${sourceRow}:${sothysSourceDetection.columns.article}`).value = article;
    sothysSourceDetection.sheet.cells.get(`${sourceRow}:${sothysSourceDetection.columns.name}`).value = name;
    sothysSourceDetection.sheet.cells.get(`${sourceRow}:${sothysSourceDetection.columns.recommended}`).value = recommended;
    sothysSourceDetection.sheet.cells.get(`${sourceRow}:${sothysSourceDetection.columns.orderedFact}`).value = "";
    sothysSourceDetection.sheet.cells.get(`${sourceRow}:${sothysSourceDetection.columns.comment}`).value = "";
  }

  const sothysResult = fillWorkbook({
    sourceWorkbook: sothysSourceWorkbook,
    blankWorkbook: sothysBlankWorkbook,
    orderMonth: "2026-07",
    brand: "sothys",
  });
  const sothysRows = new Map(sothysResult.reportRows.map((row) => [row.blankArticle, row]));
  if (sothysRows.get("160172")?.inserted !== 2) throw new Error("SOTHYS should write rounded quantity for the left variant.");
  if (sothysRows.get("360173")?.inserted !== 5) throw new Error("SOTHYS should write rounded quantity for the right variant.");
  if (sothysRows.get("350532-1")?.inserted !== null) throw new Error("SOTHYS should keep recommendations below 1.5 blank and preserve hyphenated articles.");
  const sothysSheet = sothysResult.blankWorkbook.sheets.find((item) => item.name === "Прайс SOTHYS");
  if (sothysSheet.cells.get("35:8")?.value !== 2) throw new Error("SOTHYS left order should be written to column H.");
  if (sothysSheet.cells.get("35:14")?.value !== 5) throw new Error("SOTHYS right order should be written to column N.");
  if (sothysSheet.cells.get("227:14")?.value !== "") throw new Error("SOTHYS recommendation below 1.5 should clear the order cell.");
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("SOTHYS fixture is not available; skipping SOTHYS workbook test.");
}

try {
  await fs.access(urengoySourcePath);
  const [urengoySourceWorkbook, urengoyBlankWorkbook] = await Promise.all([
    fs.readFile(urengoySourcePath).then((buffer) => loadXlsx(buffer)),
    fs.readFile(blankPath).then((buffer) => loadXlsx(buffer)),
  ]);
  const urengoyResult = fillWorkbook({
    sourceWorkbook: urengoySourceWorkbook,
    blankWorkbook: urengoyBlankWorkbook,
    orderMonth: "2026-07",
    brand: "angiopharm",
  });
  if (urengoyResult.summary.cityRule !== "Новый Уренгой") throw new Error("Urengoy source should be detected by parameters.");
  if (urengoyResult.summary.deliveryWeeks !== 2 || urengoyResult.summary.deliveryCoefficient !== 1.5) {
    throw new Error("Urengoy delivery coefficient should be detected from source parameters.");
  }
  const urengoySheet = urengoyResult.sourceWorkbook.sheets.find((item) => item.name === "Лист_1");
  if (urengoySheet.cells.get("16:23")?.value !== 136.5) {
    throw new Error(`Urengoy recommendation for CL23 should be recalculated to 136.5, got ${urengoySheet.cells.get("16:23")?.value}`);
  }
  const urengoyRow = urengoyResult.reportRows.find((row) => row.sourceArticle === "CL23");
  if (!urengoyRow || urengoyRow.recommended !== 136.5 || urengoyRow.rounded !== 137) {
    throw new Error("Urengoy recalculated recommendation should be used in the report.");
  }
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("Urengoy source fixture is not available; skipping Urengoy recalculation test.");
}

try {
  await Promise.all([fs.access(currentAngioSourcePath), fs.access(currentAngioFilledBlankPath)]);
  const [currentAngioSourceWorkbook, currentAngioBlankWorkbook] = await Promise.all([
    fs.readFile(currentAngioSourcePath).then((buffer) => loadXlsx(buffer)),
    fs.readFile(currentAngioFilledBlankPath).then((buffer) => loadXlsx(buffer)),
  ]);
  if (normalizeArticle("LС01") !== "LC01") {
    throw new Error("Mixed Latin/Cyrillic article LС01 should normalize to LC01.");
  }
  const currentAngioResult = fillWorkbook({
    sourceWorkbook: currentAngioSourceWorkbook,
    blankWorkbook: currentAngioBlankWorkbook,
    orderMonth: "2026-07",
    brand: "angiopharm",
  });
  const currentAngioSheet = currentAngioResult.blankWorkbook.sheets.find((item) => item.name === "Бланк");
  const lc01Row = currentAngioResult.reportRows.find((row) => row.blankArticle === "LC01");
  if (!lc01Row || lc01Row.sourceArticle !== "LС01" || lc01Row.inserted !== 98 || currentAngioSheet.cells.get("144:5")?.value !== 98) {
    throw new Error("Current ANGIOPHARM LC01 should match mixed Cyrillic article and be filled with 98.");
  }
  const mv07Row = currentAngioResult.reportRows.find((row) => row.blankArticle === "MV07");
  if (!mv07Row || mv07Row.status !== "warning_name_only" || currentAngioSheet.cells.get("196:5")?.value !== "") {
    throw new Error("Name-only ANGIOPHARM matches should clear stale blank quantities until reviewed.");
  }
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  console.warn("Current ANGIOPHARM fixture is not available; skipping current ANGIOPHARM matching test.");
}

const result = fillWorkbook({ sourceWorkbook, blankWorkbook, orderMonth: "2026-07" });
console.log(result.summary);

const [duplicateSourceWorkbook, duplicateBlankWorkbook] = await Promise.all([
  fs.readFile(sourcePath).then((buffer) => loadXlsx(buffer)),
  fs.readFile(blankPath).then((buffer) => loadXlsx(buffer)),
]);
const duplicateSourceDetection = detectColumns(duplicateSourceWorkbook, "source");
for (const [index, recommended] of [20, 21].entries()) {
  const sourceRow = duplicateSourceDetection.headerRow + 2 + index;
  duplicateSourceDetection.sheet.cells.get(`${sourceRow}:${duplicateSourceDetection.columns.article}`).value = "AG17";
  duplicateSourceDetection.sheet.cells.get(`${sourceRow}:${duplicateSourceDetection.columns.name}`).value = `Synthetic duplicate AG17 ${index + 1}`;
  duplicateSourceDetection.sheet.cells.get(`${sourceRow}:${duplicateSourceDetection.columns.recommended}`).value = recommended;
  duplicateSourceDetection.sheet.cells.get(`${sourceRow}:${duplicateSourceDetection.columns.orderedFact}`).value = "";
  duplicateSourceDetection.sheet.cells.get(`${sourceRow}:${duplicateSourceDetection.columns.comment}`).value = "";
}
const duplicateResult = fillWorkbook({ sourceWorkbook: duplicateSourceWorkbook, blankWorkbook: duplicateBlankWorkbook, orderMonth: "2026-07" });
const duplicateRow = duplicateResult.reportRows.find((row) => row.blankArticle === "AG17");
if (!duplicateRow?.duplicate || duplicateRow.duplicateCandidates.length < 2 || !duplicateRow.duplicateCandidates.every((item) => Number.isInteger(item.sourceRow))) {
  throw new Error("Duplicate rows should include all duplicate source candidates with source row numbers.");
}

const [sourceWithFact, blankWithFact] = await Promise.all([
  fs.readFile(sourcePath).then((buffer) => loadXlsx(buffer)),
  fs.readFile(blankPath).then((buffer) => loadXlsx(buffer)),
]);
const factSourceSheet = sourceWithFact.sheets.find((item) => item.name === "Лист_1");
factSourceSheet.cells.get("56:33").value = 25;
factSourceSheet.cells.get("56:34").value = "";
const factResult = fillWorkbook({ sourceWorkbook: sourceWithFact, blankWorkbook: blankWithFact, orderMonth: "2026-07" });
const factRow = factResult.reportRows.find((row) => row.blankArticle === "AG17");
if (!factRow || !factRow.hasOrderedFact || factRow.orderedFact !== 25 || factRow.inserted !== 25 || factRow.sourceComment !== "") {
  throw new Error("Pre-filled ordered fact should be used as inserted value.");
}
try {
  applyFinalEdits({
    blankWorkbook: factResult.blankWorkbook,
    sourceWorkbook: factResult.sourceWorkbook,
    reportRows: factResult.reportRows,
    edits: [{ key: `${factRow.blankId}:${factRow.blankRow}`, blankRow: factRow.blankRow, value: "25", comment: "" }],
  });
  throw new Error("Expected a comment validation error for pre-filled ordered fact.");
} catch (error) {
  if (!String(error.message).includes("комментарий")) throw error;
}
applyFinalEdits({
  blankWorkbook: factResult.blankWorkbook,
  sourceWorkbook: factResult.sourceWorkbook,
  reportRows: factResult.reportRows,
  edits: [{ key: `${factRow.blankId}:${factRow.blankRow}`, blankRow: factRow.blankRow, value: "25", comment: "ручная правка из таблицы" }],
});
if (factSourceSheet.cells.get("56:33")?.value !== 25) throw new Error("Ordered fact should remain 25 after valid comment.");
if (factSourceSheet.cells.get("56:34")?.value !== "ручная правка из таблицы") throw new Error("Source comment should be written for pre-filled ordered fact.");
applyFinalEdits({
  blankWorkbook: factResult.blankWorkbook,
  sourceWorkbook: factResult.sourceWorkbook,
  reportRows: factResult.reportRows,
  edits: [{ key: `${factRow.blankId}:${factRow.blankRow}`, blankRow: factRow.blankRow, value: "20", comment: "" }],
});
if (factSourceSheet.cells.get("56:33")?.value !== "") throw new Error("Ordered fact should be cleared when returned to recommendation.");
if (factSourceSheet.cells.get("56:34")?.value !== "") throw new Error("Source comment should be cleared when returned to recommendation.");

const sheet = result.blankWorkbook.sheets.find((item) => item.name === "Бланк");
function getValue(address) {
  const match = /^([A-Z]+)(\d+)$/.exec(address);
  const col = match[1].split("").reduce((acc, ch) => acc * 26 + ch.charCodeAt(0) - 64, 0);
  return sheet.cells.get(`${Number(match[2])}:${col}`)?.value;
}
const checks = [
  ["E60", 94],
  ["E127", 32],
  ["E33", ""],
  ["E79", ""],
  ["E249", 60],
];
for (const [address, expected] of checks) {
  const actual = getValue(address);
  if (actual !== expected) {
    throw new Error(`${address}: expected ${expected}, got ${actual}`);
  }
}

const boxAdjusted = result.reportRows.find((row) => row.blankArticle === "MV71");
if (!boxAdjusted || boxAdjusted.inserted !== 60 || boxAdjusted.autoComment !== "до коробки") {
  throw new Error("Box adjustment for MV71 was not applied.");
}
const unchanged = result.reportRows.find((row) => row.blankArticle === "AG17");
if (!unchanged || unchanged.rounded !== 20 || unchanged.inserted !== 20) {
  throw new Error("Unchanged AG17 fixture was not found.");
}
const underMinimum = result.reportRows.find((row) => row.blankArticle === "AG11");
if (!underMinimum || underMinimum.recommended !== 1 || underMinimum.rounded !== 1 || underMinimum.inserted !== null) {
  throw new Error("Recommendations below 1.5 should stay blank.");
}

try {
  applyFinalEdits({
    blankWorkbook: result.blankWorkbook,
    sourceWorkbook: result.sourceWorkbook,
    reportRows: result.reportRows,
    edits: [{ blankRow: 249, value: "61", comment: "до коробки" }],
  });
  throw new Error("Expected a comment validation error for changed box-adjusted row.");
} catch (error) {
  if (!String(error.message).includes("комментарий")) throw error;
}

applyFinalEdits({
  blankWorkbook: result.blankWorkbook,
  sourceWorkbook: result.sourceWorkbook,
  reportRows: result.reportRows,
  edits: [
    { blankRow: 60, value: "101", comment: "ручная правка" },
    { blankRow: 83, value: "20", comment: "" },
    { blankRow: 79, value: "", comment: "" },
    { blankRow: 33, value: "", comment: "" },
    { blankRow: 249, value: "61", comment: "ручная правка коробки" },
  ],
});
if (getValue("E60") !== 101) throw new Error("Manual edit for E60 was not applied.");
if (getValue("E33") !== "") throw new Error("Blank edit for E33 was not applied.");
if (getValue("E249") !== 61) throw new Error("Manual edit for E249 was not applied.");

const sourceSheet = result.sourceWorkbook.sheets.find((item) => item.name === "Лист_1");
const sourceFact = sourceSheet.cells.get("134:33")?.value;
const sourceComment = sourceSheet.cells.get("134:34")?.value;
if (sourceFact !== 61) throw new Error(`Source fact for MV71: expected 61, got ${sourceFact}`);
if (sourceComment !== "ручная правка коробки") throw new Error(`Source comment for MV71: expected manual comment, got ${sourceComment}`);
const unchangedFact = sourceSheet.cells.get("56:33")?.value;
const unchangedComment = sourceSheet.cells.get("56:34")?.value;
if (unchangedFact !== "") throw new Error(`Source fact for AG17 should stay empty, got ${unchangedFact}`);
if (unchangedComment !== "") throw new Error(`Source comment for AG17 should stay empty, got ${unchangedComment}`);
const underMinimumFact = sourceSheet.cells.get("80:33")?.value;
const underMinimumComment = sourceSheet.cells.get("80:34")?.value;
if (underMinimumFact !== "") throw new Error(`Source fact for AG11 should stay empty, got ${underMinimumFact}`);
if (underMinimumComment !== "") throw new Error(`Source comment for AG11 should stay empty, got ${underMinimumComment}`);

await fs.mkdir(path.dirname(blankOutputPath), { recursive: true });
const blankBytes = saveXlsx(result.blankWorkbook);
const blankZip = unzipSync(blankBytes);
const workbookXml = strFromU8(blankZip["xl/workbook.xml"]);
if (!/calcMode="auto"/.test(workbookXml) || !/fullCalcOnLoad="1"/.test(workbookXml) || !/forceFullCalc="1"/.test(workbookXml)) {
  throw new Error("Workbook should force formula recalculation on open.");
}
if (blankZip["xl/calcChain.xml"]) throw new Error("calcChain.xml should be removed after editing formula inputs.");
await fs.writeFile(blankOutputPath, blankBytes);
await fs.writeFile(sourceOutputPath, saveXlsx(result.sourceWorkbook));
console.log(blankOutputPath);
console.log(sourceOutputPath);
