export const EMPTY_ORDER_PROFILE = {
  legal_name: "",
  consignee: "",
  address: "",
  contact_name: "",
  contact_phone: "",
  carrier: "",
  delivery_payer: "",
  delivery_destination: "",
  brand_terms: [],
};

export const ORDER_PROFILE_BRANDS = [
  { id: "angiopharm", label: "ANGIOPHARM", fields: ["discount", "customer_type", "payment_method", "payment_control"] },
  { id: "skin_synergy", label: "Skin Synergy", fields: ["discount", "dealer_name"] },
  { id: "klapp", label: "KLAPP", fields: ["discount"] },
];

export function normalizeOrderProfile(payload) {
  const source = payload || {};
  return {
    ...EMPTY_ORDER_PROFILE,
    ...source,
    brand_terms: Array.isArray(source.brand_terms) ? source.brand_terms.map((item) => ({ ...item })) : [],
  };
}

export function termsFor(profile, brand) {
  return profile.brand_terms.find((item) => item.brand === brand) || {
    brand,
    dealer_name: "",
    payment_method: "",
    payment_control: "",
    customer_type: "",
    discount_basis_points: 0,
    discount_set: false,
  };
}

export function setTerms(profile, brand, patch) {
  const next = { ...termsFor(profile, brand), ...patch, brand };
  const others = profile.brand_terms.filter((item) => item.brand !== brand);
  return { ...profile, brand_terms: [...others, next] };
}

export function discountText(terms) {
  if (!terms.discount_set) return "";
  return String((Number(terms.discount_basis_points) || 0) / 100).replace(".", ",");
}

export function parseDiscount(value) {
  const text = String(value ?? "").trim();
  if (!text) return { set: false, basisPoints: 0, issue: "" };
  if (!/^\d{1,3}(?:[.,]\d{1,2})?$/.test(text)) {
    return { set: false, basisPoints: 0, issue: "Введите скидку от 0 до 100, не более двух знаков после запятой." };
  }
  const percent = Number(text.replace(",", "."));
  if (!Number.isFinite(percent) || percent < 0 || percent > 100) {
    return { set: false, basisPoints: 0, issue: "Скидка должна быть от 0 до 100%." };
  }
  return { set: true, basisPoints: Math.round(percent * 100), issue: "" };
}

export function orderProfileIssue(profile, discountInputs) {
  const phone = profile.contact_phone.trim();
  if (phone && (!/^[+\d][\d\s()+-]*$/.test(phone) || phone.replace(/\D/g, "").length < 6)) {
    return "Проверьте телефон: нужны минимум 6 цифр, допустимы +, скобки, пробелы и дефисы.";
  }
  for (const brand of ORDER_PROFILE_BRANDS) {
    const parsed = parseDiscount(discountInputs[brand.id]);
    if (parsed.issue) return `${brand.label}: ${parsed.issue}`;
  }
  return "";
}

export function profileForSave(profile, discountInputs) {
  return {
    ...profile,
    brand_terms: ORDER_PROFILE_BRANDS.map((brand) => {
      const parsed = parseDiscount(discountInputs[brand.id]);
      return {
        ...termsFor(profile, brand.id),
        discount_set: parsed.set,
        discount_basis_points: parsed.basisPoints,
      };
    }),
  };
}
