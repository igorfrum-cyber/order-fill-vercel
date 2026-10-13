import { loadXlsx } from "./../../infrastructure/excel/archive.js";
import { read as readSpreadsheet, write as writeSpreadsheet } from "xlsx";

export function createFilesLoader(context) {
  const {elements, state, actions} = context;
async function loadWorkbook(file, options = {}) {
  const buffer = await file.arrayBuffer();
  return loadXlsx(await normalizeWorkbookBytes(buffer, file.name, options));
}

function assertDifferentTyumenFiles(office, warehouse) {
  if (!office || !warehouse) throw new Error("Загрузите обе таблицы Тюмени: офис и СКЛАД ДОСТАВКА.");
  if (office.name === warehouse.name && office.size === warehouse.size && office.lastModified === warehouse.lastModified) throw new Error("Выбран один и тот же файл для офиса и склада.");
}

function isLegacyXls(fileName) {
  return /\.xls$/i.test(fileName) && !/\.xlsx$/i.test(fileName) && !/\.xlsm$/i.test(fileName);
}

async function normalizeWorkbookBytes(buffer, fileName, options = {}) {
  if (!isLegacyXls(fileName)) return buffer;
  if (options.allowLegacyXls === false) {
    throw new Error(`Файл «${fileName}» в старом формате .xls нельзя использовать как бланк: при такой конвертации теряется оформление. Откройте этот бланк в Excel и сохраните как .xlsx, затем загрузите .xlsx.`);
  }
  try {
    const workbook = readSpreadsheet(buffer, {
      type: "array",
      cellFormula: true,
      cellStyles: true,
      cellDates: false,
    });
    return writeSpreadsheet(workbook, {
      bookType: "xlsx",
      type: "array",
    });
  } catch {
    throw new Error(`Файл «${fileName}» в старом формате .xls не удалось автоматически прочитать. Откройте его в Excel и сохраните как .xlsx.`);
  }
}
  return { loadWorkbook, assertDifferentTyumenFiles, isLegacyXls, normalizeWorkbookBytes };
}
