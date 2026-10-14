import { asText, normalizeHeader } from "./../../shared/text.js";

export const ARTICLE_TRANSLATION = new Map([
  ["А", "A"], ["В", "B"], ["Е", "E"], ["К", "K"], ["М", "M"], ["Н", "H"], ["О", "O"],
  ["Р", "P"], ["С", "C"], ["Т", "T"], ["Х", "X"], ["У", "Y"], ["а", "A"], ["в", "B"],
  ["е", "E"], ["к", "K"], ["м", "M"], ["н", "H"], ["о", "O"], ["р", "P"], ["с", "C"],
  ["т", "T"], ["х", "X"], ["у", "Y"],
]);

export function normalizeArticle(value, options = {}) {
  const allowed = options.preserveHyphen ? /[^A-Z0-9-]/g : /[^A-Z0-9]/g;
  return asText(value).replace(/[АВЕКМНОРСТХУавекмнорстху]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch).toUpperCase().replace(allowed, "");
}

export function normalizeName(value) {
  return normalizeHeader(value).replace(/\bан\b/g, " ").replace(/\bangiopharm\b/g, " ").replace(/\s+/g, " ").trim();
}

export function isCombinedChzName(value) {
  return /^\s*чз\s*\+/iu.test(asText(value));
}

export function isChzCloneName(value) {
  return normalizeHeader(value).startsWith("чз ") && !isCombinedChzName(value);
}

export function comparableChzName(value, rule) {
  let name = normalizeHeader(value)
    .replace(/^чз\s+/u, "")
    .replace(/\bчз\b/gu, " ")
    .replace(/\bан\b/gu, " ")
    .replace(/\bangiopharm\b/gu, " ")
    .replace(/\s+/gu, " ")
    .trim();
  if (rule.label === "LeviSsime") {
    name = name.replace(/^(?:levissime|lev)\s+/u, "")
      .replace(/отбеливающая/gu, "осветляющая");
  }
  return name;
}

export function chzNameSimilarity(left, right, rule) {
  return similarity(comparableChzName(left, rule), comparableChzName(right, rule));
}

export function canonicalProductArticle(article, rule) {
  // LeviSsime uses both numeric and MT-prefixed articles for one product.
  for (const prefix of rule.articlePrefixAliases || []) {
    if (article.startsWith(prefix) && /^\d+$/.test(article.slice(prefix.length))) {
      return article.slice(prefix.length);
    }
  }
  return article;
}

export function levissimeRussianIdentity(value, rule) {
  const name = asText(value).replace(/^чз\s*\+?\s*/iu, "");
  const parts = name.split("/");
  if (parts.length > 2 || (parts.length === 2 && !/^[A-Za-z0-9\s+().,-]+$/.test(parts[1]))) return null;
  const primary = parts[0];
  if (!/[а-яё]/iu.test(primary) || !/\d\s*(?:мл|мг|г|л)(?=$|[^а-яё])/iu.test(primary)) return null;
  return comparableChzName(primary, rule).replace(/[a-z][a-z0-9]*/giu, " ").replace(/\s+/gu, " ").trim();
}

export function levissimeVolumeSignature(value) {
  const volumes = asText(value).toLowerCase().match(/\d+(?:[.,]\d+)?\s*(?:мл|мг|г|л)(?=$|[^а-яё])/gu) || [];
  return [...new Set(volumes.map(volume => volume.replace(/\s+/gu, "").replace(",", ".")))].sort().join(";");
}

export function levissimeEnglishIdentity(value) {
  // The first English product segment precedes optional descriptive text.
  const segment = asText(value).split("/").slice(1).find(part => /^\s*[A-Za-z]/u.test(part));
  if (!segment) return null;
  const product = segment.replace(/\d+(?:[.,]\d+)?\s*(?:мл|мг|г|л)(?=$|[^а-яё]).*$/iu, "").trim();
  if (!/^[A-Za-z\s+().-]+$/u.test(product)) return null;
  const normalized = normalizeHeader(product);
  return normalized.split(" ").length >= 3 ? normalized : null;
}

export function matchingLevissimeEnglishNames(left, right) {
  const identity = levissimeEnglishIdentity(left);
  const volume = levissimeVolumeSignature(left);
  return Boolean(identity && volume && identity === levissimeEnglishIdentity(right)
    && volume === levissimeVolumeSignature(right));
}

export function similarity(left, right) {
  const a = normalizeName(left);
  const b = normalizeName(right);
  if (!a || !b) return 0;
  const rows = Array.from({ length: b.length + 1 }, () => 0);
  for (let i = 1; i <= a.length; i += 1) {
    let previous = 0;
    for (let j = 1; j <= b.length; j += 1) {
      const tmp = rows[j];
      rows[j] = a[i - 1] === b[j - 1] ? previous + 1 : Math.max(rows[j], rows[j - 1]);
      previous = tmp;
    }
  }
  return (2 * rows[b.length]) / (a.length + b.length);
}

export function extractVolumeKeys(...values) {
  const keys = new Set();
  const text = values.map((value) => normalizeHeader(value)).join(" ");
  const pattern = /(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)\b/giu;
  for (const match of text.matchAll(pattern)) {
    const amount = Number(match[1].replace(",", "."));
    if (!Number.isFinite(amount)) continue;
    const unit = match[2].toLowerCase();
    const normalizedUnit = unit === "ml" || unit === "мл" ? "мл" : "гр";
    keys.add(`${amount}:${normalizedUnit}`);
  }
  return keys;
}

export function volumeAwareSimilarity(blankName, sourceName, blankUnit = "") {
  const base = similarity(blankName, sourceName);
  const blankVolumes = extractVolumeKeys(blankName, blankUnit);
  const sourceVolumes = extractVolumeKeys(sourceName);
  if (!blankVolumes.size || !sourceVolumes.size) return base;

  const hasMatch = Array.from(blankVolumes).some((key) => sourceVolumes.has(key));
  if (hasMatch) return Math.min(1, base + 0.06);
  return Math.max(0, base - 0.35);
}

export function chooseCandidate(candidates, blankName, blankUnit = "") {
  return candidates.map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) })).sort((left, right) => right.score - left.score)[0];
}

export function chooseNameFallback(candidates, blankName, blankUnit = "") {
  const scored = candidates.filter((item) => !item.article).map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) })).sort((left, right) => right.score - left.score);
  if (!scored.length) return { item: null, score: 0 };
  const bestNonpositive = scored.find((entry) => entry.item.rounded <= 0 && entry.score >= 0.72);
  if (bestNonpositive) return bestNonpositive;
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return { item: null, score: scored[0].score };
  if (scored[0].score < 0.72) return { item: null, score: scored[0].score };
  return scored[0];
}

export function chooseSothysNameFallback(candidates, blankName, blankUnit) {
  const unitText = normalizeHeader(blankUnit);
  const scored = candidates
    .filter((item) => !item.article && (!unitText || normalizeHeader(item.name).includes(unitText)))
    .map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) }))
    .sort((left, right) => right.score - left.score);
  if (!scored.length || scored[0].score < 0.72) return { item: null, score: scored[0]?.score || 0 };
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return { item: null, score: scored[0].score };
  return scored[0];
}
