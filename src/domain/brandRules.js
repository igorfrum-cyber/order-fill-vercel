import { normalizeCategory, parseNumber, roundHalfUp } from "./normalization.js";

const BRAND_RULES = {
  angiopharm: {
    label: "ANGIOPHARM",
    adjustment: "box",
    adjustmentLabel: "Шт. в коробке",
    adjustmentComment: "до коробки",
  },
  christina: {
    label: "CHRISTINA",
    adjustment: "multiple",
    multiple: 3,
    adjustmentLabel: "Кратность",
    adjustmentComment: "до кратности 3",
  },
  levissime: {
    label: "LeviSsime",
    adjustment: "box",
    adjustmentLabel: "Кол-во в уп.",
    adjustmentComment: "до коробки",
    articlePrefixAliases: ["MT"],
    blankQuantityHeader: "order",
    blankBoxHeader: "packageQuantity",
  },
  sothys: {
    label: "SOTHYS",
    adjustment: "none",
    adjustmentLabel: "Без округления",
    preserveArticleHyphen: true,
    blankLayout: "splitVariants",
  },
  novacutan: {
    label: "NOVACUTAN",
    adjustment: "none",
    adjustmentLabel: "Мин. заказ",
    blankLayout: "novacutan",
  },
  skin_synergy: {
    label: "Skin Synergy",
    adjustment: "none",
    adjustmentLabel: "Без округления",
    blankQuantityHeader: "exactQuantity",
    requireUnit: false,
  },
  klapp: {
    label: "KLAPP",
    adjustment: "nearestMultiple",
    multiple: 3,
    adjustmentLabel: "Кратность",
    adjustmentComment: "до кратности 3",
    blankQuantityHeader: "order",
  },
};

export function brandRule(brand) {
  return BRAND_RULES[brand] || BRAND_RULES.angiopharm;
}

export function articleNormalizeOptions(rule) {
  return { preserveHyphen: Boolean(rule.preserveArticleHyphen) };
}

export function blankDetectionOptions(rule) {
  return {
    requireBox: rule.adjustment === "box",
    requireUnit: rule.requireUnit,
    quantityHeader: rule.blankQuantityHeader,
    boxHeader: rule.blankBoxHeader,
  };
}

export function categoryCoefficient(value, rule = null) {
  const category = normalizeCategory(value);
  if (rule?.label === "NOVACUTAN") {
    if (category === "C") return 1.5;
    if (category === "A+" || category === "A" || category === "B") return 2;
  }
  if (category === "A+") return 2;
  if (category === "A") return 1.75;
  if (category === "B") return 1.5;
  if (category === "C") return 1;
  return 1;
}

export function calculateAdjustedQuantity(recommended, rule, boxSizeValue) {
  const rounded = roundHalfUp(recommended);
  if (!rule.allowSmallPositiveOrder && recommended < 1.5) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };
  if (rounded <= 0) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };

  if (rule.adjustment === "none") {
    return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  }

  if (rule.adjustment === "multiple") {
    return calculateMultipleAdjustedQuantity(rounded, rule.multiple, rule.adjustmentComment);
  }

  if (rule.adjustment === "nearestMultiple") {
    return calculateNearestMultipleAdjustedQuantity(rounded, rule.multiple, rule.adjustmentComment);
  }

  if (rule.adjustment === "minimum") {
    const minimum = Math.round(parseNumber(boxSizeValue) || 0);
    if (minimum > 0 && rounded < minimum) {
      return { rounded, inserted: minimum, autoComment: rule.adjustmentComment, boxAdjusted: true };
    }
    return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  }

  const boxSize = parseNumber(boxSizeValue);
  if (!boxSize || boxSize <= 0) return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  const box = Math.round(boxSize);
  return calculateMultipleAdjustedQuantity(rounded, box, rule.adjustmentComment);
}

export function adjustQuantityForBrand(recommended, brand = "angiopharm", boxSizeValue = null) {
  const rule = brandRule(brand);
  return calculateAdjustedQuantity(recommended, rule, boxSizeValue ?? rule.multiple);
}

function calculateMultipleAdjustedQuantity(rounded, multiple, comment) {
  const step = Math.round(Number(multiple));
  if (step <= 0 || rounded % step === 0) return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };

  const lower = Math.floor(rounded / step) * step;
  const upper = Math.ceil(rounded / step) * step;
  const upPercent = (upper - rounded) / rounded;
  const downPercent = lower > 0 ? (rounded - lower) / rounded : Infinity;

  if (upPercent > 0 && upPercent <= 0.15) {
    return { rounded, inserted: upper, autoComment: comment, boxAdjusted: true };
  }
  if (downPercent > 0 && downPercent <= 0.05) {
    return { rounded, inserted: lower, autoComment: comment, boxAdjusted: true };
  }
  return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
}

function nearestMultipleValue(value, multiple) {
  const number = Number(value || 0);
  const step = Math.round(Number(multiple));
  if (!Number.isFinite(number) || number <= 0) return null;
  if (!Number.isFinite(step) || step <= 0) return Number(number.toFixed(2));
  const lower = Math.floor(number / step) * step;
  const upper = Math.ceil(number / step) * step;
  if (lower <= 0) return upper;
  const lowerDiff = number - lower;
  const upperDiff = upper - number;
  return upperDiff <= lowerDiff ? upper : lower;
}

function calculateNearestMultipleAdjustedQuantity(rounded, multiple, comment) {
  const inserted = nearestMultipleValue(rounded, multiple);
  if (inserted == null) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };
  return {
    rounded,
    inserted,
    autoComment: inserted !== rounded ? comment : "",
    boxAdjusted: inserted !== rounded,
  };
}
