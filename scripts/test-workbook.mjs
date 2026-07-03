import fs from "node:fs/promises";
import path from "node:path";
import { strFromU8, unzipSync } from "fflate";

import { adjustQuantityForBrand, applyFinalEdits, detectColumns, fillWorkbook, loadXlsx, saveXlsx } from "../src/workbookProcessor.js";

const sourcePath = "/Users/igorfrumes/Downloads/агио артикул.xlsx";
const blankPath = "/Users/igorfrumes/Downloads/2026 06 23 Бланк заказа ANGIOPHARM (1).xlsx";
const homeBlankPath = "/Users/igorfrumes/Downloads/Актуальный бланк HOME 17.06.2026.xlsx";
const proffBlankPath = "/Users/igorfrumes/Downloads/Актуальный_бланк_PROFF_1 от 17.06.26.xlsx";
const blankOutputPath = path.resolve("test-output/browser-filled-blank.xlsx");
const sourceOutputPath = path.resolve("test-output/browser-filled-source.xlsx");

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

const result = fillWorkbook({ sourceWorkbook, blankWorkbook, orderMonth: "2026-07" });
console.log(result.summary);

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
