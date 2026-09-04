import assert from "node:assert/strict";
import test from "node:test";

import { ApiError } from "../../api/client.js";
import { userFacingError } from "./errors.js";

test("userFacingError turns technical API text into Russian", () => {
  assert.equal(
    userFacingError(new ApiError(401, { code: "unauthorized", message: "authentication is required" })),
    "Нужно войти заново.",
  );
  assert.equal(
    userFacingError(new ApiError(404, { code: "not_found", message: "job was not found" })),
    "Эту выгрузку уже нельзя открыть. Обновите список.",
  );
  assert.equal(
    userFacingError(new ApiError(409, { code: "conflict", message: "conflict" })),
    "Такая запись уже есть.",
  );
  assert.equal(
    userFacingError(new ApiError(400, { code: "bad_request", message: "invalid job: file \"blank.pdf\" must be .xlsx or .xlsm" })),
    "Нужен файл Excel: .xlsx или .xlsm.",
  );
  assert.equal(
    userFacingError(new Error("invalid workbook: не удалось найти все нужные колонки: article, name. Найдено: ничего")),
    "Этот файл не похож на нужный бланк. Проверьте, что таблица из 1С и бланк поставщика не перепутаны.",
  );
  assert.equal(
    userFacingError(new Error("API request failed with status 500")),
    "Что-то пошло не так. Попробуйте ещё раз.",
  );
});

test("userFacingError keeps already human Russian messages", () => {
  assert.equal(
    userFacingError(new Error("Для Christina нужны два бланка: HOME и PROFF.")),
    "Для Christina нужны два бланка: HOME и PROFF.",
  );
});

test("userFacingError uses the fallback when nothing useful is present", () => {
  assert.equal(userFacingError(null, "Не удалось обработать файлы."), "Не удалось обработать файлы.");
});
