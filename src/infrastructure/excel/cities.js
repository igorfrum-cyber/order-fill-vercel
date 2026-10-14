import { sheetBounds, sheetCellValue } from "./xml.js";
import { northCityFromText } from "./../../domain/north/cities.js";

export function detectSourceCity(workbook, fileName = "") {
  for (const sheet of workbook.sheets) {
    const city = northCityFromText(sheet.name);
    if (city) return city;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 60); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const cityInCell = northCityFromText(sheetCellValue(sheet, row, col));
        if (cityInCell) return cityInCell;
      }
    }
  }
  return northCityFromText(fileName);
}

export function validateNorthTyumenSourceWorkbook(workbook, fileName = "") {
  const city = detectSourceCity(workbook, fileName);
  if (!city) {
    throw new Error("Не удалось понять город в заполненной таблице Тюмени. Загрузите именно тюменскую таблицу или добавьте «Тюмень» в название файла.");
  }
  if (city.key !== "tyumen") {
    throw new Error(`В поле «Заполненная таблица Тюмени» загружена таблица города ${city.label}. Нужна именно Тюмень.`);
  }
  return city;
}

export function detectNorthCity(workbook, fileName) {
  for (const sheet of workbook.sheets) {
    const city = northCityFromText(sheet.name);
    if (city) return city;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 50); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const cityInCell = northCityFromText(sheetCellValue(sheet, row, col));
        if (cityInCell) return cityInCell;
      }
    }
  }
  const cityInName = northCityFromText(fileName);
  if (cityInName) return cityInName;
  throw new Error(`Не понял город в файле «${fileName}». В параметрах или названии файла должен быть Сургут, Нижневартовск/Вартовск, Уренгой или Тюмень.`);
}
