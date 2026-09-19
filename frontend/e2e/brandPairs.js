import fs from "node:fs";
import path from "node:path";

const BLANKS_DIR = "Бланки";
const SALES_DIR = "таблицы продаж";

export function norm(name) {
  return String(name || "")
    .normalize("NFC")
    .toLowerCase()
    .replaceAll("ё", "е")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();
}

export function brandKeyFromName(fileName) {
  const value = norm(fileName);
  const hits = new Set();
  if (value.includes("ангио") || value.includes("angio")) hits.add("angiopharm");
  if (value.includes("кристин") || value.includes("christina")) hits.add("christina");
  if (value.includes("klapp") || value.includes("клапп")) hits.add("klapp");
  if ((value.includes("skin") && value.includes("synerg")) || (value.includes("скин") && value.includes("синердж"))) {
    hits.add("skin_synergy");
  }
  if (value.includes("levissim") || value.includes("левисим")) hits.add("levissime");
  if (value.includes("sothys") || value.includes("сотис")) hits.add("sothys");
  if (value.includes("novacutan") || value.includes("новакутан")) hits.add("novacutan");
  if (value.includes("proff") || value.includes("проф") || value.includes("prof")) hits.add("christina");
  if (hits.size !== 1) return "";
  return [...hits][0];
}

export function isChristinaProffBlank(fileName) {
  if (brandKeyFromName(fileName) !== "christina") return false;
  const value = norm(fileName);
  return value.includes("proff") || value.includes("проф") || value.includes("prof");
}

export function matchBrandPairs(blankNames, saleNames) {
  const blanks = groupByBrand(blankNames);
  const sales = groupByBrand(saleNames);
  const pairs = [];
  for (const brand of Object.keys(blanks).sort()) {
    for (const blank of blanks[brand]) {
      for (const source of sales[brand] || []) {
        pairs.push({ brand, blank, source });
      }
    }
  }
  return pairs;
}

/** Canonical Christina PROFF UI fixtures — real Tyumen sales × PROFF (1) blank. */
export const CHRISTINA_PROFF_UI_BLANK = "Актуальный_бланк PROFF (1).xlsx";
export const CHRISTINA_PROFF_UI_SOURCE = "Кристина Тюмень .xlsx";

export function sameWorkbookName(left, right) {
  return norm(left) === norm(right);
}

export function resolvePrivateTestdata(repoRoot, env = process.env) {
  if (env.ORDER_FILL_PRIVATE_TESTDATA) return env.ORDER_FILL_PRIVATE_TESTDATA;
  const candidates = [
    path.join(repoRoot, "testdata/private"),
    // git worktree under .worktrees/<slug> shares private fixtures with the main checkout
    path.resolve(repoRoot, "../../testdata/private"),
  ];
  for (const root of candidates) {
    if (fs.existsSync(path.join(root, BLANKS_DIR)) && fs.existsSync(path.join(root, SALES_DIR))) {
      return root;
    }
  }
  return candidates[0];
}

export function scanPrivateBrandPairs(dataRoot) {
  const blankDir = path.join(dataRoot, BLANKS_DIR);
  const salesDir = path.join(dataRoot, SALES_DIR);
  if (!fs.existsSync(blankDir) || !fs.existsSync(salesDir)) {
    return { available: false, pairs: [] };
  }
  const pairs = matchBrandPairs(listXlsx(blankDir), listXlsx(salesDir)).map((pair) => ({
    ...pair,
    blankPath: path.join(blankDir, pair.blank),
    sourcePath: path.join(salesDir, pair.source),
  }));
  return { available: true, pairs };
}

/** One real Christina PROFF pair for UI / budget / compare checks. */
export function christinaProffUiPair(dataRoot) {
  const scanned = scanPrivateBrandPairs(dataRoot);
  if (!scanned.available) return { available: false, pair: null };
  const pair = scanned.pairs.find(
    (item) =>
      item.brand === "christina" &&
      sameWorkbookName(item.blank, CHRISTINA_PROFF_UI_BLANK) &&
      sameWorkbookName(item.source, CHRISTINA_PROFF_UI_SOURCE),
  );
  return { available: true, pair: pair || null };
}

function groupByBrand(names) {
  const groups = {};
  for (const name of names) {
    const brand = brandKeyFromName(name);
    if (!brand) continue;
    (groups[brand] ||= []).push(name);
  }
  for (const list of Object.values(groups)) list.sort((a, b) => a.localeCompare(b, "ru"));
  return groups;
}

function listXlsx(dir) {
  return fs.readdirSync(dir).filter((name) => /\.xlsx$/i.test(name) && !name.startsWith("~$"));
}
