const FALLBACK = "Что-то пошло не так. Попробуйте ещё раз.";

const CODE_MESSAGES = {
  unauthorized: "Нужно войти заново.",
  conflict: "Такая запись уже есть.",
};

const PATTERN_MESSAGES = [
  [/authentication is required/i, "Нужно войти заново."],
  [/job was not found|report was not found/i, "Эту выгрузку уже нельзя открыть. Обновите список."],
  [/file was not found|preview was not found/i, "Файл уже недоступен. Обновите страницу."],
  [/\bnot found\b/i, "Не нашли то, что искали."],
  [/\.xlsx or \.xlsm/i, "Нужен файл Excel: .xlsx или .xlsm."],
  [/login slug must be latin|invalid login slug/i, "Адрес входа — латиницей, цифрами и дефисом."],
  [/login slug is reserved/i, "Этот адрес входа нельзя использовать. Выберите другой."],
  [/login slug is required/i, "Напишите латинский адрес входа компании."],
  [/invalid password/i, "Пароль не подходит. Проверьте требования и попробуйте снова."],
  [/company name is required/i, "Напишите название компании."],
  [/login is required/i, "Напишите логин."],
  [/unsupported role/i, "Эту роль здесь выбрать нельзя."],
  [/logo must be png jpeg or webp|invalid logo/i, "Логотип должен быть картинкой PNG, JPEG или WebP."],
  [/challenge expired/i, "Код или ссылка устарели. Попробуйте ещё раз."],
  [/нужные колонки|invalid workbook|article, name|orderedFact|blank_files|source_file/i, "Этот файл не похож на нужный бланк. Проверьте, что таблица из 1С и бланк поставщика не перепутаны."],
  [/API request failed|dial tcp|connection refused|sql:|pq:|EOF|ECONNREFUSED/i, FALLBACK],
];

export function userFacingError(error, fallback = FALLBACK) {
  const code = error?.payload?.code || "";
  const raw = String(error?.payload?.message || error?.message || "").trim();
  const stripped = stripTechnicalPrefix(raw);
  if (looksHumanRussian(stripped)) return stripped;
  if (code === "unauthorized") return CODE_MESSAGES.unauthorized;
  if (code === "conflict") return CODE_MESSAGES.conflict;
  if (code === "not_found") return matchPatterns(raw) || "Не нашли то, что искали.";
  return matchPatterns(raw) || matchPatterns(stripped) || fallback;
}

function matchPatterns(text) {
  for (const [pattern, message] of PATTERN_MESSAGES) {
    if (pattern.test(text)) return message;
  }
  return "";
}

function stripTechnicalPrefix(text) {
  return text.replace(/^invalid workbook:\s*/i, "").replace(/^invalid job:\s*/i, "").trim();
}

function looksHumanRussian(text) {
  if (!text) return false;
  if (!/[а-яё]/i.test(text)) return false;
  return !/article|orderedFact|inTransit|source_file|blank_files|invalid workbook|api request/i.test(text);
}
