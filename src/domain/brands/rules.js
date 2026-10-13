

export const BRAND_RULES = {
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
