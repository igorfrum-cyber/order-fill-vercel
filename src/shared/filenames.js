import { asText, normalizeHeader } from "./text.js";

export function outputFileName(originalName, cityName = "") {
  const text = asText(originalName);
  const extension = outputExtension(text);
  const stem = text.replace(/\.(xlsx|xlsm|xls)$/i, "").replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim() || "blank";
  const city = asText(cityName).replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim();
  const cityAlreadyInName = city && normalizeHeader(stem).includes(normalizeHeader(city));
  const namedStem = city && !cityAlreadyInName ? `${stem} ${city}` : stem;
  return `${namedStem} заполненный.${extension}`;
}

export function sourceOutputFileName(originalName) {
  const text = asText(originalName);
  const extension = outputExtension(text);
  const stem = text.replace(/\.(xlsx|xlsm|xls)$/i, "").replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim() || "order";
  return `${stem} заполненная таблица.${extension}`;
}

export function outputExtension(fileName) {
  return /\.xlsm$/i.test(asText(fileName)) ? "xlsm" : "xlsx";
}
