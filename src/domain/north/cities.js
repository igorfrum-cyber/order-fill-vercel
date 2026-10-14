import { normalizeHeader } from "./../../shared/text.js";

export const NORTH_CITIES = [
  { key: "nizhnevartovsk", label: "Нижневартовск", warehouse: "Склад Нижневартовск", aliases: ["нижневартовск", "вартовск"] },
  { key: "urengoy", label: "Уренгой", warehouse: "Склад Уренгой", aliases: ["новый уренгой", "уренгой"] },
  { key: "surgut", label: "Сургут", warehouse: "Склад Сургут", aliases: ["сургут"] },
  { key: "tyumen", label: "Тюмень", warehouse: "Склад Тюмень", aliases: ["тюмень", "склад доставка"] },
];

export const NORTH_TRANSFER_DISPLAY_ORDER = ["surgut", "nizhnevartovsk", "urengoy"];

export function northCityFromText(value) {
  const text = normalizeHeader(value);
  if (!text) return null;
  return NORTH_CITIES.find((city) => city.aliases.some((alias) => text.includes(alias))) || null;
}
