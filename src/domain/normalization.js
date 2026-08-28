const ARTICLE_TRANSLATION = new Map([
  ["А", "A"], ["В", "B"], ["Е", "E"], ["К", "K"], ["М", "M"], ["Н", "H"], ["О", "O"],
  ["Р", "P"], ["С", "C"], ["Т", "T"], ["Х", "X"], ["У", "Y"], ["а", "A"], ["в", "B"],
  ["е", "E"], ["к", "K"], ["м", "M"], ["н", "H"], ["о", "O"], ["р", "P"], ["с", "C"],
  ["т", "T"], ["х", "X"], ["у", "Y"],
]);

export function asText(value) {
  if (value == null) return "";
  return String(value).normalize("NFC").replace(/\n/g, " ").trim();
}

export function normalizeHeader(value) {
  return asText(value).toLowerCase().replaceAll("ё", "е").replace(/[^\p{L}\p{N}%]+/gu, " ").replace(/\s+/g, " ").trim();
}

export function normalizeArticle(value, options = {}) {
  const allowed = options.preserveHyphen ? /[^A-Z0-9-]/g : /[^A-Z0-9]/g;
  return asText(value).replace(/[АВЕКМНОРСТХУавекмнорстху]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch).toUpperCase().replace(allowed, "");
}

export function normalizeName(value) {
  return normalizeHeader(value).replace(/\bан\b/g, " ").replace(/\bangiopharm\b/g, " ").replace(/\s+/g, " ").trim();
}

export function normalizeCategory(value) {
  return asText(value)
    .replace(/[АВСавс]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch)
    .toUpperCase()
    .replace(/\s+/g, "");
}

export function parseNumber(value) {
  if (value == null || asText(value) === "") return null;
  const normalized = asText(value).replace(/\s+/g, "").replace(",", ".");
  const number = Number(normalized);
  if (Number.isFinite(number)) return number;
  const match = normalized.match(/-?\d+(?:\.\d+)?/);
  if (!match) return null;
  const extracted = Number(match[0]);
  return Number.isFinite(extracted) ? extracted : null;
}

export function roundHalfUp(value) {
  return Math.floor(value + 0.5);
}
