export const ORDER_BRANDS = [
  { id: "angiopharm", label: "ANGIOPHARM" },
  { id: "christina", label: "CHRISTINA" },
  { id: "levissime", label: "LeviSsime" },
  { id: "sothys", label: "SOTHYS" },
  { id: "novacutan", label: "NOVACUTAN" },
  { id: "skin_synergy", label: "Skin Synergy" },
  { id: "klapp", label: "KLAPP" },
];

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

export function brandLabel(brand) {
  return ORDER_BRANDS.find((item) => item.id === brand)?.label || brand;
}

export function blankSlotsForBrand(_brand) {
  return [
    {
      id: "main",
      label: "Текущий бланк",
      hint: `Бланк для заполнения количеств`,
      accept: ".xlsx,.xlsm,.xls",
    },
  ];
}

export function looksLikeChristinaSource(fileName) {
  const value = String(fileName || "").toLowerCase();
  return value.includes("кристин") || value.includes("christina");
}

export function blankSlotsForSource(_fileName) {
  return [
    {
      id: "main",
      label: "Бланк",
      hint: "Бланк для заполнения количеств",
      accept: ".xlsx,.xlsm,.xls",
    },
  ];
}
