const BRAND_PRESENTATION = {
  christina: {
    adjustmentLabel: "Кратность",
    mainBlankLabel: "ANGIO",
    splitBlank: true,
  },
  levissime: {
    adjustmentLabel: "Кол-во в уп.",
    mainBlankLabel: "LeviSsime",
  },
  sothys: {
    adjustmentLabel: "Округление",
    mainBlankLabel: "SOTHYS",
  },
  novacutan: {
    adjustmentLabel: "Мин. заказ",
    mainBlankLabel: "NOVACUTAN",
  },
  skin_synergy: {
    adjustmentLabel: "Округление",
    mainBlankLabel: "Skin Synergy",
  },
  klapp: {
    adjustmentLabel: "Кратность",
    mainBlankLabel: "KLAPP",
  },
};

const DEFAULT_PRESENTATION = {
  adjustmentLabel: "Шт. в коробке",
  mainBlankLabel: "ANGIO",
  splitBlank: false,
};

export function presentationForBrand(brand) {
  return {
    ...DEFAULT_PRESENTATION,
    ...(BRAND_PRESENTATION[brand] || {}),
  };
}

export function adjustmentLabelForBrand(brand) {
  return presentationForBrand(brand).adjustmentLabel;
}

export function mainBlankLabelForBrand(brand) {
  return presentationForBrand(brand).mainBlankLabel;
}

export function usesChristinaSplitBlank(brand) {
  return Boolean(presentationForBrand(brand).splitBlank);
}
