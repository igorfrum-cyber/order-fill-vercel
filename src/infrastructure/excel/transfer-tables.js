import { utils as spreadsheetUtils, write as writeSpreadsheet } from "xlsx";

export function todayRu() {
  const date = new Date();
  return `${String(date.getDate()).padStart(2, "0")}.${String(date.getMonth() + 1).padStart(2, "0")}.${date.getFullYear()}`;
}

export async function transferWorkbookBytes(transfer) {
    const utils = spreadsheetUtils, write = writeSpreadsheet;
    const rows = [
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", `Заказ на перемещение от ${todayRu()}`, "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "Отправитель:", "СКЛАД ДОСТАВКА", "Получатель:", transfer.city.warehouse, "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "№", "Товар", "Количество", "", "", "", ""],
      ...transfer.items.map((item, index) => ["", index + 1, item.name, item.quantity, item.unit || "шт", "", "", ""]),
      ["", "", "", "", "", "", "", ""],
      ["Менеджер", "", "", "", "", "", "", ""],
    ];
    const workbook = utils.book_new();
    const sheet = utils.aoa_to_sheet(rows);
    sheet["!cols"] = [
      { wch: 8 },
      { wch: 8 },
      { wch: 72 },
      { wch: 14 },
      { wch: 10 },
      { wch: 22 },
      { wch: 12 },
      { wch: 12 },
    ];
    utils.book_append_sheet(workbook, sheet, "Лист_1");
    return write(workbook, { bookType: "xlsx", type: "array" });
}

export async function northOrderTableBytes(table) {
    const utils = spreadsheetUtils, write = writeSpreadsheet;
    const rows = [
      ["Позиция", "Заказано", "Комментарий"],
      ...table.rows.map((item) => [item.name, item.quantity, item.comment]),
    ];
    const workbook = utils.book_new();
    const sheet = utils.aoa_to_sheet(rows);
    sheet["!cols"] = [
      { wch: 72 },
      { wch: 14 },
      { wch: 80 },
    ];
    utils.book_append_sheet(workbook, sheet, "Заказ");
    return write(workbook, { bookType: "xlsx", type: "array" });
}
