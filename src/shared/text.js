

export function asText(value) {
  if (value == null) return "";
  return String(value).normalize("NFC").replace(/\n/g, " ").trim();
}

export function normalizeHeader(value) {
  return asText(value).toLowerCase().replaceAll("ё", "е").replace(/[^\p{L}\p{N}%]+/gu, " ").replace(/\s+/g, " ").trim();
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
