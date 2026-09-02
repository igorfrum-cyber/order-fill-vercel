export const excelAcceptHint = "Подходят Excel-файлы.";

export const northDuplicateFileMessage = "Все выбранные бланки уже добавлены.";

export const northMissingCityBlankMessage = "Добавьте хотя бы один бланк города.";

export function orderUploadSteps() {
  return [
    { n: 1, title: "Таблица продаж из 1С" },
    { n: 2, title: "Бланк поставщика" },
  ];
}

export function northUploadSteps() {
  return [
    { n: 1, title: "Бланки городов" },
    { n: 2, title: "Таблица Тюмени, если нужно учесть остатки" },
  ];
}

export function selectedFileCountLabel(count) {
  const n = Number(count) || 0;
  if (n <= 0) return "Файлы не выбраны.";
  if (n === 1) return "Выбран 1 файл.";
  return `Выбрано файлов: ${n}.`;
}

export function orderSelectedCount(sourceFile, blankFiles = {}) {
  return Number(Boolean(sourceFile)) + Object.values(blankFiles).filter(Boolean).length;
}

export function northSelectedCount({ files = [], homeFiles = [], proffFiles = [], tyumenFile } = {}) {
  return files.length + homeFiles.length + proffFiles.length + Number(Boolean(tyumenFile));
}
