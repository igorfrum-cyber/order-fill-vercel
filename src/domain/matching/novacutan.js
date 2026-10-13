import { normalizeHeader } from "./../../shared/text.js";
import { chooseCandidate, normalizeName, volumeAwareSimilarity } from "./identity.js";

export function novacutanMatchKey(value) {
  const text = normalizeHeader(value)
    .replace(/\bbiopro\b/g, "bio pro")
    .replace(/\bbio\s*pro\b/g, "bio pro");
  if (!text || text.includes("термопакет") || text.includes("хладоэлемент")) return "";
  const hasFillerMask = text.includes("filler") || text.includes("филлер") || text.includes("маск") || text.includes("mask");
  if (hasFillerMask && (text.includes("глаз") || text.includes("eye"))) return "mask-eye";
  if (hasFillerMask && (text.includes("лица") || text.includes("face"))) return "mask-face";
  if (text.includes("sbio")) return "sbio";
  if (text.includes("ybio")) return "ybio";
  if (text.includes("bio pro")) return "bio-pro";
  if (text.includes("prima")) return "prima";
  if (text.includes("master")) return "master";
  if (text.includes("bright") || text.includes("брайт")) return "bright";
  if (text.includes("gentle") || text.includes("джентл") || text.includes("джентел")) return "gentle";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("light")) return "dvs-fbio-light";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("medium")) return "dvs-fbio-medium";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("volume")) return "dvs-fbio-volume";
  if (text.includes("fbio") && text.includes("light")) return "fbio-light";
  if (text.includes("fbio") && text.includes("medium")) return "fbio-medium";
  if (text.includes("fbio") && text.includes("volume")) return "fbio-volume";
  if (text.includes("eye")) return "eye";
  return "";
}

export function chooseNovacutanCandidate(sourceItems, blankName, blankUnit = "") {
  const key = novacutanMatchKey(blankName);
  if (key) {
    const keyed = sourceItems.filter((item) => novacutanMatchKey(item.name) === key);
    const blankText = normalizeHeader(blankName);
    const beautyBoxAligned = keyed.filter((item) => {
      const sourceText = normalizeHeader(item.name);
      const blankIsBeautyBox = blankText.includes("бьюти") || blankText.includes("beauty");
      const sourceIsBeautyBox = sourceText.includes("бьюти") || sourceText.includes("beauty");
      return blankIsBeautyBox === sourceIsBeautyBox;
    });
    const nonBonus = (beautyBoxAligned.length ? beautyBoxAligned : keyed).filter((item) => !normalizeHeader(item.name).includes("бонус"));
    const pool = nonBonus.length ? nonBonus : beautyBoxAligned.length ? beautyBoxAligned : keyed;
    if (pool.length) {
      return pool
        .map((item) => ({ item, score: Math.max(0.92, volumeAwareSimilarity(blankName, item.name, blankUnit)) }))
        .sort((left, right) => {
          const positiveDiff = Number(right.item.recommended > 0) - Number(left.item.recommended > 0);
          if (positiveDiff) return positiveDiff;
          return right.score - left.score;
        })[0];
    }
  }
  return chooseCandidate(sourceItems, blankName, blankUnit);
}

export function novacutanPositionKey(name) {
  const matched = novacutanMatchKey(name);
  return `novacutan:${matched || normalizeName(name).replace(/\bновакутан\b/g, "novacutan")}`;
}

export function novacutanNameTokens(value) {
  return normalizeHeader(value)
    .split(/\s+/u)
    .map((token) => token.trim())
    .filter((token) => token && token !== "novacutan" && token !== "мл" && token !== "гр");
}

export function novacutanNameScore(left, right, unit = "") {
  const base = volumeAwareSimilarity(left, right, unit);
  const leftTokens = new Set(novacutanNameTokens(left));
  const rightTokens = new Set(novacutanNameTokens(right));
  if (!leftTokens.size || !rightTokens.size) return base;
  const intersection = Array.from(leftTokens).filter((token) => rightTokens.has(token)).length;
  const coverage = intersection / Math.min(leftTokens.size, rightTokens.size);
  if (coverage >= 0.75) return Math.max(base, 0.9);
  return base;
}
